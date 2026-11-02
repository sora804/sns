// ローカル確認用のテストデータを投入する。何度実行しても重複しない。
//
//	cd backend
//	DATABASE_URL="postgres://sns@127.0.0.1:5433/sns?sslmode=disable" go run ./cmd/seed
//
// 安全のため、接続先が localhost / 127.0.0.1 以外のときは実行しない。
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sora804/sns/backend/internal/domain"
	"github.com/sora804/sns/backend/internal/repository/postgres"
	"github.com/sora804/sns/backend/internal/security"
	"github.com/sora804/sns/backend/internal/usecase"
	"github.com/sora804/sns/backend/migrations"
)

// テストアカウント（ローカル確認専用。本番では絶対に使わない）。
const testPassword = "password123"

var users = []struct{ handle, name string }{
	{"alice", "アリス"},
	{"bob", "ボブ"},
	{"carol", "キャロル"},
}

var posts = []struct{ handle, body string }{
	{"alice", "はじめまして、アリスです。"},
	{"bob", "ボブです。よろしくお願いします。"},
	{"alice", "今日はいい天気ですね。☀️"},
	{"carol", "キャロルです。まだ誰もフォローしていません。"},
	{"bob", "140文字までの短い投稿ができます。"},
}

// フォロー関係: alice と bob は相互フォロー。carol は誰もフォローしない。
var follows = [][2]string{{"alice", "bob"}, {"bob", "alice"}}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	u, err := url.Parse(dsn)
	if err != nil || dsn == "" {
		return errors.New("DATABASE_URL が設定されていないか、形式が正しくありません")
	}
	if h := u.Hostname(); h != "localhost" && h != "127.0.0.1" && h != "::1" {
		return fmt.Errorf("接続先が %q です。シードはローカルのDBにだけ投入できます", h)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool, migrations.FS); err != nil {
		return err
	}

	repos := postgres.New(pool)
	auth := usecase.NewAuthUsecase(repos.Users, repos.Sessions, security.NewArgon2Hasher(security.DefaultParams))
	postUC := usecase.NewPostUsecase(repos.Posts, repos.Users)
	social := usecase.NewSocialUsecase(repos.Users, repos.Posts, repos.Follows, repos.Likes)

	ids := map[string]int64{}
	for _, s := range users {
		email := s.handle + "@example.com"
		created, err := auth.Register(ctx, s.handle, s.name, email, testPassword)
		switch {
		case err == nil:
			ids[s.handle] = created.ID
			fmt.Printf("作成: %s\n", email)
		case errors.Is(err, domain.ErrConflict):
			existing, err := repos.Users.FindByHandle(ctx, s.handle)
			if err != nil {
				return err
			}
			ids[s.handle] = existing.ID
			fmt.Printf("既存: %s\n", email)
		default:
			return err
		}
	}

	// 投稿は、まだ1件も無いユーザーにだけ入れる（再実行しても増えない）
	seeded := map[string]bool{}
	for _, p := range posts {
		if !seeded[p.handle] {
			seeded[p.handle] = true
			page, err := postUC.ByUser(ctx, p.handle, 0, 0, 1)
			if err != nil {
				return err
			}
			if len(page.Items) > 0 {
				seeded[p.handle] = false
			}
		}
		if !seeded[p.handle] {
			continue
		}
		if _, err := postUC.Create(ctx, ids[p.handle], p.body); err != nil {
			return err
		}
	}

	for _, f := range follows {
		if err := social.Follow(ctx, ids[f[0]], f[1]); err != nil {
			return err
		}
	}
	fmt.Println("完了。パスワードは cmd/seed/main.go の testPassword を参照")
	return nil
}
