package domain

import (
	"context"
	"time"
)

// リポジトリのインターフェース。実装は internal/repository 以下にある。
// 一覧系の before は「この ID より古いもの」を返すカーソル。0 なら先頭から。

type UserRepository interface {
	// Create は ID と CreatedAt を設定する。ハンドルまたはメールが重複したら ErrConflict。
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByHandle(ctx context.Context, handle string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Profile(ctx context.Context, handle string, viewerID int64) (*UserProfile, error)
}

type SessionRepository interface {
	Create(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error
	// FindUserID は有効期限内のセッションのユーザー ID を返す。無ければ ErrNotFound。
	FindUserID(ctx context.Context, tokenHash string, now time.Time) (int64, error)
	Delete(ctx context.Context, tokenHash string) error
}

type PostRepository interface {
	Create(ctx context.Context, p *Post) error
	// Get は削除済みの投稿を返さない（ErrNotFound）。
	Get(ctx context.Context, id, viewerID int64) (*PostView, error)
	// Delete は論理削除する。
	Delete(ctx context.Context, id int64) error
	// Home は本人とフォロー中ユーザーの投稿を新しい順に返す。
	Home(ctx context.Context, viewerID, before int64, limit int) ([]PostView, error)
	ByUser(ctx context.Context, authorID, viewerID, before int64, limit int) ([]PostView, error)
}

// FollowRepository と LikeRepository の操作は冪等（既に済みでもエラーにしない）。
type FollowRepository interface {
	Follow(ctx context.Context, followerID, followeeID int64) error
	Unfollow(ctx context.Context, followerID, followeeID int64) error
}

type LikeRepository interface {
	Like(ctx context.Context, userID, postID int64) error
	Unlike(ctx context.Context, userID, postID int64) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) bool
}
