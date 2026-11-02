package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/sora804/sns/backend/internal/domain"
	"github.com/sora804/sns/backend/internal/repository/memory"
	"github.com/sora804/sns/backend/internal/security"
	"github.com/sora804/sns/backend/internal/usecase"
)

type env struct {
	auth   *usecase.AuthUsecase
	posts  *usecase.PostUsecase
	social *usecase.SocialUsecase
}

func newEnv() *env {
	s := memory.New()
	hasher := security.NewArgon2Hasher(security.Argon2Params{Memory: 8, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8})
	return &env{
		auth:   usecase.NewAuthUsecase(s.Users, s.Sessions, hasher),
		posts:  usecase.NewPostUsecase(s.Posts, s.Users),
		social: usecase.NewSocialUsecase(s.Users, s.Posts, s.Follows, s.LikeStore),
	}
}

func (e *env) user(t *testing.T, handle string) *domain.User {
	t.Helper()
	u, err := e.auth.Register(context.Background(), handle, handle, handle+"@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestRegisterDuplicate(t *testing.T) {
	e := newEnv()
	e.user(t, "taro")
	_, err := e.auth.Register(context.Background(), "TARO", "x", "other@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("ハンドル重複(大文字小文字違い)が ErrConflict にならない: %v", err)
	}
	_, err = e.auth.Register(context.Background(), "jiro", "x", "taro@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("メール重複が ErrConflict にならない: %v", err)
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	_, err := newEnv().auth.Register(context.Background(), "taro", "太郎", "t@example.com", "short")
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("err = %v", err)
	}
}

func TestLoginAndAuthenticate(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	u := e.user(t, "taro")

	token, _, err := e.auth.Login(ctx, "TARO@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.auth.Authenticate(ctx, token)
	if err != nil || got.ID != u.ID {
		t.Fatalf("Authenticate = %v, %v", got, err)
	}

	if _, _, err := e.auth.Login(ctx, "taro@example.com", "wrong-password"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("誤ったパスワード: %v", err)
	}
	if _, _, err := e.auth.Login(ctx, "nobody@example.com", "password123"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("存在しないメール: %v", err)
	}

	if err := e.auth.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := e.auth.Authenticate(ctx, token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("ログアウト後も認証できた: %v", err)
	}
}

func TestDeletePostOnlyByOwner(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	a, b := e.user(t, "alice"), e.user(t, "bob")
	p, err := e.posts.Create(ctx, a.ID, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.posts.Delete(ctx, b.ID, p.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("他人の投稿を削除できた: %v", err)
	}
	if err := e.posts.Delete(ctx, a.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.posts.Get(ctx, p.ID, a.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("削除済みの投稿が取得できた: %v", err)
	}
}

func TestHomeTimelineShowsOnlyFollowed(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	me, followed, stranger := e.user(t, "myself"), e.user(t, "followed"), e.user(t, "stranger")

	mustPost := func(u *domain.User, body string) {
		t.Helper()
		if _, err := e.posts.Create(ctx, u.ID, body); err != nil {
			t.Fatal(err)
		}
	}
	mustPost(me, "mine")
	mustPost(followed, "followed-1")
	mustPost(stranger, "stranger-1")
	if err := e.social.Follow(ctx, me.ID, "followed"); err != nil {
		t.Fatal(err)
	}

	page, err := e.posts.Home(ctx, me.ID, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, p := range page.Items {
		bodies = append(bodies, p.Body)
	}
	want := []string{"followed-1", "mine"} // 新しい順
	if fmt.Sprint(bodies) != fmt.Sprint(want) {
		t.Errorf("timeline = %v, want %v", bodies, want)
	}

	// フォロー解除すると表示されなくなる
	_ = e.social.Unfollow(ctx, me.ID, "followed")
	page, _ = e.posts.Home(ctx, me.ID, 0, 20)
	if len(page.Items) != 1 || page.Items[0].Body != "mine" {
		t.Errorf("フォロー解除後の timeline = %v", page.Items)
	}
}

func TestTimelinePagination(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	u := e.user(t, "taro")
	for i := 1; i <= 5; i++ {
		if _, err := e.posts.Create(ctx, u.ID, fmt.Sprintf("post-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	p1, _ := e.posts.Home(ctx, u.ID, 0, 2)
	if len(p1.Items) != 2 || p1.NextCursor == 0 || p1.Items[0].Body != "post-5" {
		t.Fatalf("1ページ目 = %+v", p1)
	}
	p2, _ := e.posts.Home(ctx, u.ID, p1.NextCursor, 2)
	if len(p2.Items) != 2 || p2.Items[0].Body != "post-3" {
		t.Fatalf("2ページ目 = %+v", p2)
	}
	p3, _ := e.posts.Home(ctx, u.ID, p2.NextCursor, 2)
	if len(p3.Items) != 1 || p3.NextCursor != 0 || p3.Items[0].Body != "post-1" {
		t.Fatalf("3ページ目 = %+v", p3)
	}
}

func TestFollowSelfRejected(t *testing.T) {
	e := newEnv()
	u := e.user(t, "taro")
	if err := e.social.Follow(context.Background(), u.ID, "taro"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("自分をフォローできた: %v", err)
	}
	if err := e.social.Follow(context.Background(), u.ID, "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("存在しないユーザーをフォローできた: %v", err)
	}
}

func TestLikeIsIdempotentAndCounted(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	a, b := e.user(t, "alice"), e.user(t, "bob")
	p, _ := e.posts.Create(ctx, a.ID, "hello")

	for i := 0; i < 2; i++ { // 2回押しても1件
		if err := e.social.Like(ctx, b.ID, p.ID); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := e.posts.Get(ctx, p.ID, b.ID)
	if got.LikeCount != 1 || !got.LikedByMe {
		t.Errorf("like = %d, likedByMe = %v", got.LikeCount, got.LikedByMe)
	}
	other, _ := e.posts.Get(ctx, p.ID, a.ID)
	if other.LikedByMe {
		t.Error("他人のいいねが自分のものとして見えている")
	}

	_ = e.social.Unlike(ctx, b.ID, p.ID)
	got, _ = e.posts.Get(ctx, p.ID, b.ID)
	if got.LikeCount != 0 {
		t.Errorf("取り消し後 like = %d", got.LikeCount)
	}
	if err := e.social.Like(ctx, b.ID, 9999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("存在しない投稿へのいいね: %v", err)
	}
}

func TestProfileCounts(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	a, b := e.user(t, "alice"), e.user(t, "bob")
	_, _ = e.posts.Create(ctx, a.ID, "x")
	_ = e.social.Follow(ctx, b.ID, "alice")

	p, err := e.social.Profile(ctx, "alice", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.FollowerCount != 1 || p.PostCount != 1 || !p.IsFollowing {
		t.Errorf("profile = %+v", p)
	}
}
