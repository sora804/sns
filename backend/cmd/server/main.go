// 起動と依存の組み立て（DI）だけを行う。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sora804/sns/backend/internal/config"
	"github.com/sora804/sns/backend/internal/handler"
	"github.com/sora804/sns/backend/internal/repository/postgres"
	"github.com/sora804/sns/backend/internal/security"
	"github.com/sora804/sns/backend/internal/usecase"
	"github.com/sora804/sns/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("起動に失敗", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	if err := postgres.Migrate(ctx, pool, migrations.FS); err != nil {
		return err
	}

	repos := postgres.New(pool)
	hasher := security.NewArgon2Hasher(security.DefaultParams)
	router, err := handler.NewRouter(handler.Deps{
		Auth:         usecase.NewAuthUsecase(repos.Users, repos.Sessions, hasher),
		Posts:        usecase.NewPostUsecase(repos.Posts, repos.Users),
		Social:       usecase.NewSocialUsecase(repos.Users, repos.Posts, repos.Follows, repos.Likes),
		CookieSecure: cfg.CookieSecure,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	slog.Info("起動", "addr", cfg.Addr)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("停止します")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
