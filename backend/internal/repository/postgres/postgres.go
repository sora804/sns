// Package postgres は PostgreSQL を使ったリポジトリ実装。SQL はこのパッケージにだけ書く。
package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sora804/sns/backend/internal/domain"
)

// Repos は全リポジトリをまとめたもの。
type Repos struct {
	Users    *UserRepo
	Sessions *SessionRepo
	Posts    *PostRepo
	Follows  *FollowRepo
	Likes    *LikeRepo
}

func New(pool *pgxpool.Pool) *Repos {
	return &Repos{
		Users:    &UserRepo{pool},
		Sessions: &SessionRepo{pool},
		Posts:    &PostRepo{pool},
		Follows:  &FollowRepo{pool},
		Likes:    &LikeRepo{pool},
	}
}

// Migrate は未適用のマイグレーションをファイル名順に適用する。
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	files, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		var done bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		sql, err := fs.ReadFile(migrations, name)
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// mapErr は pgx のエラーを業務エラーに変換する。
func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return domain.ErrConflict
	}
	return err
}
