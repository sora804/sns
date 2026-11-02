package handler

import (
	"context"
	"net/http"

	"github.com/sora804/sns/backend/internal/domain"
	"github.com/sora804/sns/backend/internal/gen"
	"github.com/sora804/sns/backend/internal/middleware"
	"github.com/sora804/sns/backend/internal/usecase"
)

// 各メソッドは「リクエストを usecase の引数に変換 → usecase を呼ぶ → 結果を生成型に変換」だけを行う。

// ---- 認証 ----

func (a *API) Register(ctx context.Context, req *gen.RegisterRequest) (*gen.User, error) {
	u, err := a.Auth.Register(ctx, req.Handle, req.DisplayName, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	out := toUser(u)
	return &out, nil
}

func (a *API) sessionCookie(value string, maxAge int) string {
	return (&http.Cookie{
		Name:     middleware.SessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   a.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

func (a *API) Login(ctx context.Context, req *gen.LoginRequest) (*gen.UserHeaders, error) {
	token, u, err := a.Auth.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	return &gen.UserHeaders{
		SetCookie: gen.NewOptString(a.sessionCookie(token, int(usecase.SessionTTL.Seconds()))),
		Response:  toUser(u),
	}, nil
}

func (a *API) Logout(ctx context.Context) (*gen.LogoutNoContent, error) {
	if s, ok := sessionFrom(ctx); ok {
		if err := a.Auth.Logout(ctx, s.token); err != nil {
			return nil, err
		}
	}
	return &gen.LogoutNoContent{SetCookie: gen.NewOptString(a.sessionCookie("", -1))}, nil
}

func (a *API) GetMe(ctx context.Context) (*gen.User, error) {
	s, ok := sessionFrom(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	out := toUser(s.user)
	return &out, nil
}

// ---- ユーザー ----

func (a *API) GetProfile(ctx context.Context, params gen.GetProfileParams) (*gen.Profile, error) {
	p, err := a.Social.Profile(ctx, params.Handle, viewerID(ctx))
	if err != nil {
		return nil, err
	}
	return &gen.Profile{
		Handle: p.Handle, DisplayName: p.DisplayName, Bio: p.Bio, CreatedAt: p.CreatedAt,
		FollowerCount: p.FollowerCount, FollowingCount: p.FollowingCount,
		PostCount: p.PostCount, IsFollowing: p.IsFollowing,
	}, nil
}

func (a *API) ListUserPosts(ctx context.Context, params gen.ListUserPostsParams) (*gen.PostPage, error) {
	cursor, err := parseCursor(params.Cursor)
	if err != nil {
		return nil, err
	}
	page, err := a.Posts.ByUser(ctx, params.Handle, viewerID(ctx), cursor, params.Limit.Or(0))
	if err != nil {
		return nil, err
	}
	return toPage(page), nil
}

func (a *API) Follow(ctx context.Context, params gen.FollowParams) error {
	return a.Social.Follow(ctx, viewerID(ctx), params.Handle)
}

func (a *API) Unfollow(ctx context.Context, params gen.UnfollowParams) error {
	return a.Social.Unfollow(ctx, viewerID(ctx), params.Handle)
}

// ---- 投稿 ----

func (a *API) CreatePost(ctx context.Context, req *gen.CreatePostRequest) (*gen.Post, error) {
	v, err := a.Posts.Create(ctx, viewerID(ctx), req.Body)
	if err != nil {
		return nil, err
	}
	out := toPost(*v)
	return &out, nil
}

func (a *API) GetPost(ctx context.Context, params gen.GetPostParams) (*gen.Post, error) {
	v, err := a.Posts.Get(ctx, params.ID, viewerID(ctx))
	if err != nil {
		return nil, err
	}
	out := toPost(*v)
	return &out, nil
}

func (a *API) DeletePost(ctx context.Context, params gen.DeletePostParams) error {
	return a.Posts.Delete(ctx, viewerID(ctx), params.ID)
}

func (a *API) LikePost(ctx context.Context, params gen.LikePostParams) error {
	return a.Social.Like(ctx, viewerID(ctx), params.ID)
}

func (a *API) UnlikePost(ctx context.Context, params gen.UnlikePostParams) error {
	return a.Social.Unlike(ctx, viewerID(ctx), params.ID)
}

// ---- タイムライン ----

func (a *API) GetHomeTimeline(ctx context.Context, params gen.GetHomeTimelineParams) (*gen.PostPage, error) {
	cursor, err := parseCursor(params.Cursor)
	if err != nil {
		return nil, err
	}
	page, err := a.Posts.Home(ctx, viewerID(ctx), cursor, params.Limit.Or(0))
	if err != nil {
		return nil, err
	}
	return toPage(page), nil
}
