// Package middleware は handler の前後で共通に行う処理（CSRF対策、panic回復、ログ）。
// 認証は OpenAPI の security として ogen 側で扱う（internal/handler の HandleSessionCookie）。
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	SessionCookie = "session"
	// CSRFHeader は状態を変更するリクエストに必須のヘッダー。
	// 別オリジンのフォームからは付けられないため、CSRF対策になる。
	CSRFHeader = "X-Requested-With"
)

// CSRF は GET/HEAD/OPTIONS 以外のリクエストに CSRFHeader を要求する。
func CSRF(onFail http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
			default:
				if r.Header.Get(CSRFHeader) == "" {
					onFail(w, r)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recover は panic を 500 に変換してサーバーの停止を防ぐ。
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "err", rec, "path", r.URL.Path)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Logger はリクエストを構造化ログに出す。
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds())
	})
}
