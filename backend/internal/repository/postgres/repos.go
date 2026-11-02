package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sora804/sns/backend/internal/domain"
)

// ---- users ----

type UserRepo struct{ db *pgxpool.Pool }

const userCols = `id, handle, display_name, email, password_hash, bio, created_at`

func scanUser(row pgx.Row, u *domain.User) error {
	return row.Scan(&u.ID, &u.Handle, &u.DisplayName, &u.Email, &u.PasswordHash, &u.Bio, &u.CreatedAt)
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (handle, display_name, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		u.Handle, u.DisplayName, u.Email, u.PasswordHash).Scan(&u.ID, &u.CreatedAt)
	return mapErr(err)
}

func (r *UserRepo) findOne(ctx context.Context, where string, arg any) (*domain.User, error) {
	var u domain.User
	if err := scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE `+where, arg), &u); err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	return r.findOne(ctx, `id = $1`, id)
}

func (r *UserRepo) FindByHandle(ctx context.Context, handle string) (*domain.User, error) {
	return r.findOne(ctx, `lower(handle) = lower($1)`, handle)
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.findOne(ctx, `email = $1`, email)
}

func (r *UserRepo) Profile(ctx context.Context, handle string, viewerID int64) (*domain.UserProfile, error) {
	var p domain.UserProfile
	err := r.db.QueryRow(ctx, `
		SELECT `+userCols+`,
		  (SELECT count(*) FROM follows WHERE followee_id = u.id),
		  (SELECT count(*) FROM follows WHERE follower_id = u.id),
		  (SELECT count(*) FROM posts WHERE user_id = u.id AND deleted_at IS NULL),
		  EXISTS (SELECT 1 FROM follows WHERE follower_id = $2 AND followee_id = u.id)
		FROM users u WHERE lower(u.handle) = lower($1)`, handle, viewerID).
		Scan(&p.ID, &p.Handle, &p.DisplayName, &p.Email, &p.PasswordHash, &p.Bio, &p.CreatedAt,
			&p.FollowerCount, &p.FollowingCount, &p.PostCount, &p.IsFollowing)
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

// ---- sessions ----

type SessionRepo struct{ db *pgxpool.Pool }

func (r *SessionRepo) Create(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`, tokenHash, userID, expiresAt)
	return mapErr(err)
}

func (r *SessionRepo) FindUserID(ctx context.Context, tokenHash string, now time.Time) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > $2`, tokenHash, now).Scan(&id)
	return id, mapErr(err)
}

func (r *SessionRepo) Delete(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

// ---- posts ----

type PostRepo struct{ db *pgxpool.Pool }

// 投稿に投稿者・いいね情報を付けて取得するSELECT。$1 は閲覧者ID。
const postViewSelect = `
	SELECT p.id, p.user_id, p.body, p.created_at, u.handle, u.display_name,
	  (SELECT count(*) FROM likes l WHERE l.post_id = p.id),
	  EXISTS (SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = $1)
	FROM posts p JOIN users u ON u.id = p.user_id
	WHERE p.deleted_at IS NULL `

func scanPostView(row pgx.Row, v *domain.PostView) error {
	return row.Scan(&v.ID, &v.UserID, &v.Body, &v.CreatedAt, &v.AuthorHandle, &v.AuthorDisplayName, &v.LikeCount, &v.LikedByMe)
}

func (r *PostRepo) Create(ctx context.Context, p *domain.Post) error {
	err := r.db.QueryRow(ctx, `INSERT INTO posts (user_id, body) VALUES ($1, $2) RETURNING id, created_at`, p.UserID, p.Body).
		Scan(&p.ID, &p.CreatedAt)
	return mapErr(err)
}

func (r *PostRepo) Get(ctx context.Context, id, viewerID int64) (*domain.PostView, error) {
	var v domain.PostView
	if err := scanPostView(r.db.QueryRow(ctx, postViewSelect+`AND p.id = $2`, viewerID, id), &v); err != nil {
		return nil, mapErr(err)
	}
	return &v, nil
}

func (r *PostRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `UPDATE posts SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostRepo) query(ctx context.Context, sql string, args ...any) ([]domain.PostView, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PostView{}
	for rows.Next() {
		var v domain.PostView
		if err := scanPostView(rows, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PostRepo) Home(ctx context.Context, viewerID, before int64, limit int) ([]domain.PostView, error) {
	return r.query(ctx, postViewSelect+`
		AND ($2::bigint = 0 OR p.id < $2::bigint)
		AND (p.user_id = $1 OR p.user_id IN (SELECT followee_id FROM follows WHERE follower_id = $1))
		ORDER BY p.id DESC LIMIT $3`, viewerID, before, limit)
}

func (r *PostRepo) ByUser(ctx context.Context, authorID, viewerID, before int64, limit int) ([]domain.PostView, error) {
	return r.query(ctx, postViewSelect+`
		AND ($2::bigint = 0 OR p.id < $2::bigint) AND p.user_id = $4
		ORDER BY p.id DESC LIMIT $3`, viewerID, before, limit, authorID)
}

// ---- follows / likes ----

type FollowRepo struct{ db *pgxpool.Pool }

func (r *FollowRepo) Follow(ctx context.Context, followerID, followeeID int64) error {
	_, err := r.db.Exec(ctx, `INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, followerID, followeeID)
	return err
}

func (r *FollowRepo) Unfollow(ctx context.Context, followerID, followeeID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2`, followerID, followeeID)
	return err
}

type LikeRepo struct{ db *pgxpool.Pool }

func (r *LikeRepo) Like(ctx context.Context, userID, postID int64) error {
	_, err := r.db.Exec(ctx, `INSERT INTO likes (user_id, post_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, postID)
	return err
}

func (r *LikeRepo) Unlike(ctx context.Context, userID, postID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM likes WHERE user_id = $1 AND post_id = $2`, userID, postID)
	return err
}
