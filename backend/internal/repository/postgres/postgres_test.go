package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sora804/sns/backend/internal/domain"
	"github.com/sora804/sns/backend/internal/repository/postgres"
	"github.com/sora804/sns/backend/migrations"
)

// TEST_DATABASE_URL が設定されているときだけ実行する結合テスト（CIでは設定済み）。
// テストごとにテーブルを空にするので、本番や開発用のDBを指定しないこと。
func setup(t *testing.T) (*postgres.Repos, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL が未設定のためスキップ")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := postgres.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if err := postgres.Migrate(ctx, pool, migrations.FS); err != nil { // 2回目は何もしない
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, sessions, posts, follows, likes RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	return postgres.New(pool), ctx
}

func mkUser(t *testing.T, r *postgres.Repos, ctx context.Context, handle string) *domain.User {
	t.Helper()
	u, _ := domain.NewUser(handle, handle, handle+"@example.com", "hash")
	if err := r.Users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestUsersUniqueness(t *testing.T) {
	r, ctx := setup(t)
	mkUser(t, r, ctx, "alice")
	dup, _ := domain.NewUser("ALICE", "x", "other@example.com", "hash")
	if err := r.Users.Create(ctx, dup); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("ハンドル重複: %v", err)
	}
	if _, err := r.Users.FindByHandle(ctx, "Alice"); err != nil {
		t.Errorf("大文字小文字を無視した検索: %v", err)
	}
	if _, err := r.Users.FindByEmail(ctx, "nobody@example.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("存在しないメール: %v", err)
	}
}

func TestSessions(t *testing.T) {
	r, ctx := setup(t)
	u := mkUser(t, r, ctx, "alice")
	now := time.Now()
	_ = r.Sessions.Create(ctx, "tok", u.ID, now.Add(time.Hour))
	if id, err := r.Sessions.FindUserID(ctx, "tok", now); err != nil || id != u.ID {
		t.Fatalf("id = %d, err = %v", id, err)
	}
	if _, err := r.Sessions.FindUserID(ctx, "tok", now.Add(2*time.Hour)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("期限切れ: %v", err)
	}
	_ = r.Sessions.Delete(ctx, "tok")
	if _, err := r.Sessions.FindUserID(ctx, "tok", now); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("削除後: %v", err)
	}
}

func TestPostsTimelineLikesAndFollows(t *testing.T) {
	r, ctx := setup(t)
	a, b, c := mkUser(t, r, ctx, "alice"), mkUser(t, r, ctx, "bob"), mkUser(t, r, ctx, "carol")
	for _, p := range []*domain.Post{{UserID: a.ID, Body: "a1"}, {UserID: b.ID, Body: "b1"}, {UserID: c.ID, Body: "c1"}, {UserID: b.ID, Body: "b2"}} {
		if err := r.Posts.Create(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	_ = r.Follows.Follow(ctx, a.ID, b.ID)
	_ = r.Follows.Follow(ctx, a.ID, b.ID) // 冪等

	home, err := r.Posts.Home(ctx, a.ID, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(home) != 3 || home[0].Body != "b2" || home[2].Body != "a1" { // a(本人)+b、c は出ない
		t.Fatalf("home = %+v", home)
	}
	older, _ := r.Posts.Home(ctx, a.ID, home[0].ID, 10)
	if len(older) != 2 {
		t.Errorf("cursor 後 = %d件", len(older))
	}

	_ = r.Likes.Like(ctx, a.ID, home[0].ID)
	_ = r.Likes.Like(ctx, a.ID, home[0].ID) // 冪等
	v, _ := r.Posts.Get(ctx, home[0].ID, a.ID)
	if v.LikeCount != 1 || !v.LikedByMe {
		t.Errorf("like = %+v", v)
	}

	prof, _ := r.Users.Profile(ctx, "bob", a.ID)
	if prof.FollowerCount != 1 || prof.PostCount != 2 || !prof.IsFollowing {
		t.Errorf("profile = %+v", prof)
	}

	if err := r.Posts.Delete(ctx, home[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Posts.Get(ctx, home[0].ID, a.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("論理削除後: %v", err)
	}
	if err := r.Posts.Delete(ctx, home[0].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("二重削除: %v", err)
	}
}
