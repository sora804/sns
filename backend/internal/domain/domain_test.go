package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewPost(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"通常", "こんにちは", false},
		{"ちょうど140文字", strings.Repeat("あ", 140), false},
		{"141文字", strings.Repeat("あ", 141), true},
		{"空", "", true},
		{"空白のみ", " \n\t ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPost(1, tt.body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Errorf("ErrValidation ではない: %v", err)
			}
		})
	}
}

func TestNewUser(t *testing.T) {
	tests := []struct {
		name                  string
		handle, display, mail string
		wantErr               bool
	}{
		{"正常", "taro_1", "太郎", "taro@example.com", false},
		{"ハンドルが短い", "ab", "太郎", "taro@example.com", true},
		{"ハンドルが長い", "abcdefghijklmnop", "太郎", "taro@example.com", true},
		{"ハンドルに不正文字", "taro-1", "太郎", "taro@example.com", true},
		{"表示名が空", "taro", "  ", "taro@example.com", true},
		{"メール形式が不正", "taro", "太郎", "not-an-email", true},
		{"名前付きメールは不可", "taro", "太郎", "Taro <taro@example.com>", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUser(tt.handle, tt.display, tt.mail, "hash")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewUserLowercasesEmail(t *testing.T) {
	u, err := NewUser("taro", "太郎", "Taro@Example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "taro@example.com" {
		t.Errorf("email = %q", u.Email)
	}
}

func TestValidatePassword(t *testing.T) {
	if ValidatePassword("short") == nil {
		t.Error("短いパスワードが通った")
	}
	if ValidatePassword("long-enough-password") != nil {
		t.Error("正しいパスワードが拒否された")
	}
}
