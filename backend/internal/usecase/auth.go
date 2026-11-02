// Package usecase はアプリケーションの手順（ユースケース）を実装する。
package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/sora804/sns/backend/internal/domain"
)

const SessionTTL = 30 * 24 * time.Hour

type AuthUsecase struct {
	users    domain.UserRepository
	sessions domain.SessionRepository
	hasher   domain.PasswordHasher
	now      func() time.Time
}

func NewAuthUsecase(u domain.UserRepository, s domain.SessionRepository, h domain.PasswordHasher) *AuthUsecase {
	return &AuthUsecase{users: u, sessions: s, hasher: h, now: time.Now}
}

func (a *AuthUsecase) Register(ctx context.Context, handle, displayName, email, password string) (*domain.User, error) {
	if err := domain.ValidatePassword(password); err != nil {
		return nil, err
	}
	hash, err := a.hasher.Hash(password)
	if err != nil {
		return nil, err
	}
	u, err := domain.NewUser(handle, displayName, email, hash)
	if err != nil {
		return nil, err
	}
	if err := a.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login は成功するとセッショントークン（Cookie に入れる値）を返す。
// メール不一致とパスワード不一致は区別せず、どちらも ErrUnauthorized にする。
func (a *AuthUsecase) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	u, err := a.users.FindByEmail(ctx, strings.ToLower(email))
	if errors.Is(err, domain.ErrNotFound) {
		return "", nil, domain.ErrUnauthorized
	}
	if err != nil {
		return "", nil, err
	}
	if !a.hasher.Verify(u.PasswordHash, password) {
		return "", nil, domain.ErrUnauthorized
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if err := a.sessions.Create(ctx, hashToken(token), u.ID, a.now().Add(SessionTTL)); err != nil {
		return "", nil, err
	}
	return token, u, nil
}

func (a *AuthUsecase) Logout(ctx context.Context, token string) error {
	return a.sessions.Delete(ctx, hashToken(token))
}

// Authenticate はトークンからログイン中のユーザーを返す。無効なら ErrUnauthorized。
func (a *AuthUsecase) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	id, err := a.sessions.FindUserID(ctx, hashToken(token), a.now())
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	u, err := a.users.FindByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrUnauthorized
	}
	return u, err
}

// DB にはトークンそのものではなくハッシュを保存する（漏えい時にセッションを乗っ取られないため）。
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
