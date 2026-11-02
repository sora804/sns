package domain

import (
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MinPasswordLen = 8
	MaxPasswordLen = 128
	MaxBioLen      = 160
)

var handleRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,15}$`)

type User struct {
	ID           int64
	Handle       string
	DisplayName  string
	Email        string
	PasswordHash string
	Bio          string
	CreatedAt    time.Time
}

// UserProfile はプロフィール画面用の表示データ。
type UserProfile struct {
	User
	FollowerCount  int
	FollowingCount int
	PostCount      int
	IsFollowing    bool
}

// NewUser は入力を検証して User を作る。パスワードはハッシュ済みのものを渡す。
func NewUser(handle, displayName, email, passwordHash string) (*User, error) {
	if !handleRe.MatchString(handle) {
		return nil, invalid("handle", "3〜15文字の英数字とアンダースコアで入力してください")
	}
	displayName = strings.TrimSpace(displayName)
	if n := utf8.RuneCountInString(displayName); n < 1 || n > 50 {
		return nil, invalid("displayName", "1〜50文字で入力してください")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return nil, invalid("email", "メールアドレスの形式が正しくありません")
	}
	return &User{
		Handle:       handle,
		DisplayName:  displayName,
		Email:        strings.ToLower(email),
		PasswordHash: passwordHash,
	}, nil
}

func ValidatePassword(pw string) error {
	if len(pw) < MinPasswordLen || len(pw) > MaxPasswordLen {
		return invalid("password", "8〜128文字で入力してください")
	}
	return nil
}
