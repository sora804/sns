package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

const MaxPostLen = 140

type Post struct {
	ID        int64
	UserID    int64
	Body      string
	CreatedAt time.Time
}

// PostView は投稿に表示用の情報（投稿者、いいね数など）を加えたもの。
type PostView struct {
	Post
	AuthorHandle      string
	AuthorDisplayName string
	LikeCount         int
	LikedByMe         bool
}

func NewPost(userID int64, body string) (*Post, error) {
	if strings.TrimSpace(body) == "" {
		return nil, invalid("body", "本文を入力してください")
	}
	if utf8.RuneCountInString(body) > MaxPostLen {
		return nil, invalid("body", "140文字以内で入力してください")
	}
	return &Post{UserID: userID, Body: body}, nil
}
