package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sora804/sns/backend/internal/handler"
	"github.com/sora804/sns/backend/internal/repository/memory"
	"github.com/sora804/sns/backend/internal/security"
	"github.com/sora804/sns/backend/internal/usecase"
)

// client は Cookie を保持する簡易HTTPクライアント。
type client struct {
	t      *testing.T
	srv    http.Handler
	cookie *http.Cookie
}

func newServer() http.Handler {
	s := memory.New()
	hasher := security.NewArgon2Hasher(security.Argon2Params{Memory: 8, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8})
	h, err := handler.NewRouter(handler.Deps{
		Auth:   usecase.NewAuthUsecase(s.Users, s.Sessions, hasher),
		Posts:  usecase.NewPostUsecase(s.Posts, s.Users),
		Social: usecase.NewSocialUsecase(s.Users, s.Posts, s.Follows, s.LikeStore),
	})
	if err != nil {
		panic(err)
	}
	return h
}

func (c *client) do(method, path, body string, withCSRF bool) *httptest.ResponseRecorder {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if withCSRF {
		req.Header.Set("X-Requested-With", "test")
	}
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	rec := httptest.NewRecorder()
	c.srv.ServeHTTP(rec, req)
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "session" {
			if ck.MaxAge < 0 {
				c.cookie = nil
			} else {
				c.cookie = ck
			}
		}
	}
	return rec
}

func (c *client) mustStatus(rec *httptest.ResponseRecorder, want int) {
	c.t.Helper()
	if rec.Code != want {
		c.t.Fatalf("status = %d, want %d, body = %s", rec.Code, want, rec.Body.String())
	}
}

func (c *client) signup(handle string) {
	c.t.Helper()
	c.mustStatus(c.do("POST", "/api/v1/auth/register",
		`{"handle":"`+handle+`","displayName":"`+handle+`","email":"`+handle+`@example.com","password":"password123"}`, true), 201)
	c.mustStatus(c.do("POST", "/api/v1/auth/login",
		`{"email":"`+handle+`@example.com","password":"password123"}`, true), 200)
}

func TestFullFlow(t *testing.T) {
	srv := newServer()
	alice := &client{t: t, srv: srv}
	bob := &client{t: t, srv: srv}
	alice.signup("alice")
	bob.signup("bob")

	// alice が投稿 → bob のホームには、フォローするまで出ない
	rec := alice.do("POST", "/api/v1/posts", `{"body":"はじめての投稿"}`, true)
	alice.mustStatus(rec, 201)
	var post struct {
		ID           int64  `json:"id"`
		AuthorHandle string `json:"authorHandle"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &post)
	if post.AuthorHandle != "alice" || post.ID == 0 {
		t.Fatalf("post = %+v", post)
	}

	homeCount := func(c *client) int {
		rec := c.do("GET", "/api/v1/timeline/home", "", false)
		c.mustStatus(rec, 200)
		var page struct {
			Items []json.RawMessage `json:"items"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &page)
		return len(page.Items)
	}
	if n := homeCount(bob); n != 0 {
		t.Fatalf("フォロー前の bob のホーム = %d件", n)
	}
	bob.mustStatus(bob.do("PUT", "/api/v1/users/alice/follow", "", true), 204)
	if n := homeCount(bob); n != 1 {
		t.Fatalf("フォロー後の bob のホーム = %d件", n)
	}

	// いいね
	bob.mustStatus(bob.do("PUT", "/api/v1/posts/"+itoa(post.ID)+"/like", "", true), 204)
	rec = bob.do("GET", "/api/v1/posts/"+itoa(post.ID), "", false)
	if !strings.Contains(rec.Body.String(), `"likeCount":1`) || !strings.Contains(rec.Body.String(), `"likedByMe":true`) {
		t.Errorf("いいねが反映されていない: %s", rec.Body.String())
	}

	// 他人の投稿は削除できない
	bob.mustStatus(bob.do("DELETE", "/api/v1/posts/"+itoa(post.ID), "", true), 403)
	alice.mustStatus(alice.do("DELETE", "/api/v1/posts/"+itoa(post.ID), "", true), 204)
	alice.mustStatus(alice.do("GET", "/api/v1/posts/"+itoa(post.ID), "", false), 404)

	// ログアウト後は認証が必要なAPIが 401 になる
	alice.mustStatus(alice.do("POST", "/api/v1/auth/logout", "", true), 204)
	alice.mustStatus(alice.do("GET", "/api/v1/timeline/home", "", false), 401)
}

func TestCSRFHeaderRequired(t *testing.T) {
	c := &client{t: t, srv: newServer()}
	c.mustStatus(c.do("POST", "/api/v1/auth/register",
		`{"handle":"alice","displayName":"a","email":"a@example.com","password":"password123"}`, false), 403)
}

func TestAuthRequired(t *testing.T) {
	c := &client{t: t, srv: newServer()}
	c.mustStatus(c.do("POST", "/api/v1/posts", `{"body":"x"}`, true), 401)
	c.mustStatus(c.do("GET", "/api/v1/timeline/home", "", false), 401)
}

func TestValidationErrorShape(t *testing.T) {
	c := &client{t: t, srv: newServer()}
	rec := c.do("POST", "/api/v1/auth/register",
		`{"handle":"a","displayName":"a","email":"a@example.com","password":"password123"}`, true)
	c.mustStatus(rec, 400)
	// 仕様（pattern）に反する入力は ogen が検証して弾く
	if !strings.Contains(rec.Body.String(), `"code":"validation_error"`) {
		t.Errorf("body = %s", rec.Body.String())
	}
	// 仕様の範囲内で業務ルールに反する入力は domain が検証して弾く（field が付く）。
	// 140文字は仕様を満たすので、空白のみの本文で確認する
	alice := &client{t: t, srv: c.srv}
	alice.signup("alice")
	rec = alice.do("POST", "/api/v1/posts", `{"body":"   "}`, true)
	alice.mustStatus(rec, 400)
	if !strings.Contains(rec.Body.String(), `"code":"validation_error"`) || !strings.Contains(rec.Body.String(), `"field":"body"`) {
		t.Errorf("body = %s", rec.Body.String())
	}
	// 未知のフィールドは拒否
	c.mustStatus(c.do("POST", "/api/v1/auth/login", `{"email":"a@example.com","password":"x","admin":true}`, true), 400)
}

func TestLoginFailureIsGeneric(t *testing.T) {
	srv := newServer()
	c := &client{t: t, srv: srv}
	c.signup("alice")
	other := &client{t: t, srv: srv}
	c.mustStatus(other.do("POST", "/api/v1/auth/login", `{"email":"alice@example.com","password":"wrong-password"}`, true), 401)
	c.mustStatus(other.do("POST", "/api/v1/auth/login", `{"email":"nobody@example.com","password":"password123"}`, true), 401)
}

func TestSessionCookieAttributes(t *testing.T) {
	srv := newServer()
	c := &client{t: t, srv: srv}
	c.mustStatus(c.do("POST", "/api/v1/auth/register",
		`{"handle":"alice","displayName":"a","email":"a@example.com","password":"password123"}`, true), 201)
	rec := c.do("POST", "/api/v1/auth/login", `{"email":"a@example.com","password":"password123"}`, true)
	ck := rec.Result().Cookies()[0]
	if !ck.HttpOnly || ck.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie = %+v", ck)
	}
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
