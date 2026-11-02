// Package handler は ogen が生成したインターフェース（internal/gen）を実装し、HTTPの入出力を担う。
// 業務ルールはここに書かない。
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/sora804/sns/backend/internal/domain"
	"github.com/sora804/sns/backend/internal/gen"
	"github.com/sora804/sns/backend/internal/middleware"
	"github.com/sora804/sns/backend/internal/usecase"
)

type Deps struct {
	Auth         *usecase.AuthUsecase
	Posts        *usecase.PostUsecase
	Social       *usecase.SocialUsecase
	CookieSecure bool
}

// API は gen.Handler と gen.SecurityHandler を実装する。
type API struct{ Deps }

var (
	_ gen.Handler         = (*API)(nil)
	_ gen.SecurityHandler = (*API)(nil)
)

// NewRouter は /api/v1 配下のルーティングを組み立てる。
func NewRouter(d Deps) (http.Handler, error) {
	api := &API{d}
	srv, err := gen.NewServer(api, api,
		gen.WithPathPrefix("/api/v1"),
		gen.WithErrorHandler(api.errorHandler),
		gen.WithNotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, api.NewError(r.Context(), domain.ErrNotFound))
		}),
		gen.WithMethodNotAllowed(func(w http.ResponseWriter, r *http.Request, _ string) {
			writeError(w, apiError(http.StatusMethodNotAllowed, "method_not_allowed", "許可されていないメソッドです", ""))
		}),
	)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", srv)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	var root http.Handler = mux
	root = middleware.CSRF(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, apiError(http.StatusForbidden, "csrf", "リクエストが不正です", ""))
	})(root)
	root = middleware.Logger(root)
	root = middleware.Recover(root)
	return root, nil
}

// ---- 認証（OpenAPI の security: sessionCookie） ----

type ctxKey struct{}

type session struct {
	user  *domain.User
	token string
}

func sessionFrom(ctx context.Context) (session, bool) {
	s, ok := ctx.Value(ctxKey{}).(session)
	return s, ok
}

// viewerID はログイン中のユーザーIDを返す。未ログインなら 0。
func viewerID(ctx context.Context) int64 {
	if s, ok := sessionFrom(ctx); ok {
		return s.user.ID
	}
	return 0
}

// HandleSessionCookie は Cookie からユーザーを解決する。
// 無効なセッションは「未ログイン」として扱う（任意認証のAPIは通し、必須のAPIは 401 になる）。
func (a *API) HandleSessionCookie(ctx context.Context, _ gen.OperationName, t gen.SessionCookie) (context.Context, error) {
	u, err := a.Auth.Authenticate(ctx, t.APIKey)
	if errors.Is(err, domain.ErrUnauthorized) {
		return ctx, ogenerrors.ErrSkipServerSecurity
	}
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, ctxKey{}, session{user: u, token: t.APIKey}), nil
}

// ---- エラー変換 ----

func apiError(status int, code, message, field string) *gen.ErrorStatusCode {
	e := gen.Error{Code: code, Message: message}
	if field != "" {
		e.Field = gen.NewOptString(field)
	}
	return &gen.ErrorStatusCode{StatusCode: status, Response: e}
}

// NewError は業務エラーなどを HTTP ステータスとエラー本文に変換する。想定外のエラーは詳細を隠す。
func (a *API) NewError(_ context.Context, err error) *gen.ErrorStatusCode {
	var (
		ve  *domain.ValidationError
		sec *ogenerrors.SecurityError
	)
	switch {
	case errors.As(err, &ve):
		return apiError(http.StatusBadRequest, "validation_error", ve.Message, ve.Field)
	case errors.As(err, &sec), errors.Is(err, domain.ErrUnauthorized):
		return apiError(http.StatusUnauthorized, "unauthorized", "ログインが必要です", "")
	case errors.Is(err, domain.ErrForbidden):
		return apiError(http.StatusForbidden, "forbidden", "この操作は許可されていません", "")
	case errors.Is(err, domain.ErrNotFound):
		return apiError(http.StatusNotFound, "not_found", "見つかりません", "")
	case errors.Is(err, domain.ErrConflict):
		return apiError(http.StatusConflict, "conflict", "すでに使われています", "")
	}
	// ogen がリクエストの解析・検証で返すエラー（JSON形式の誤り、制約違反など）は 400
	if ogenerrors.ErrorCode(err) == http.StatusBadRequest {
		return apiError(http.StatusBadRequest, "validation_error", "入力が正しくありません", "")
	}
	slog.Error("unhandled", "err", err)
	return apiError(http.StatusInternalServerError, "internal_error", "サーバーエラーが発生しました", "")
}

// errorHandler は ogen 内部で起きたエラー（認証・解析・検証）を同じ形式で返す。
func (a *API) errorHandler(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	writeError(w, a.NewError(ctx, err))
}

func writeError(w http.ResponseWriter, e *gen.ErrorStatusCode) {
	// 生成された MarshalJSON を使うため、ポインタで渡す（Field が未設定のときも正しく出力される）
	body, err := json.Marshal(&e.Response)
	if err != nil {
		slog.Error("encode error response", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(e.StatusCode)
	_, _ = w.Write(body)
}

// ---- 変換 ----

func toUser(u *domain.User) gen.User {
	return gen.User{Handle: u.Handle, DisplayName: u.DisplayName, Bio: u.Bio, CreatedAt: u.CreatedAt}
}

func toPost(v domain.PostView) gen.Post {
	return gen.Post{ID: v.ID, Body: v.Body, CreatedAt: v.CreatedAt, AuthorHandle: v.AuthorHandle,
		AuthorDisplayName: v.AuthorDisplayName, LikeCount: v.LikeCount, LikedByMe: v.LikedByMe}
}

func toPage(p *usecase.Page) *gen.PostPage {
	out := &gen.PostPage{Items: make([]gen.Post, 0, len(p.Items))}
	for _, v := range p.Items {
		out.Items = append(out.Items, toPost(v))
	}
	if p.NextCursor == 0 {
		out.NextCursor = gen.NilString{Null: true}
	} else {
		out.NextCursor = gen.NewNilString(strconv.FormatInt(p.NextCursor, 10))
	}
	return out
}

// parseCursor はクエリの cursor を数値にする。省略は 0（先頭から）。
func parseCursor(s gen.OptString) (int64, error) {
	if !s.IsSet() || s.Value == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s.Value, 10, 64)
	if err != nil || n < 0 {
		return 0, &domain.ValidationError{Field: "cursor", Message: "cursor が正しくありません"}
	}
	return n, nil
}
