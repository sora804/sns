package usecase

import (
	"context"

	"github.com/sora804/sns/backend/internal/domain"
)

const (
	defaultLimit = 20
	maxLimit     = 50
)

// Page は一覧の1ページ分。NextCursor が 0 なら続きはない。
type Page struct {
	Items      []domain.PostView
	NextCursor int64
}

type PostUsecase struct {
	posts domain.PostRepository
	users domain.UserRepository
}

func NewPostUsecase(p domain.PostRepository, u domain.UserRepository) *PostUsecase {
	return &PostUsecase{posts: p, users: u}
}

func (u *PostUsecase) Create(ctx context.Context, userID int64, body string) (*domain.PostView, error) {
	p, err := domain.NewPost(userID, body)
	if err != nil {
		return nil, err
	}
	if err := u.posts.Create(ctx, p); err != nil {
		return nil, err
	}
	return u.posts.Get(ctx, p.ID, userID)
}

// Get の viewerID は未ログインなら 0。
func (u *PostUsecase) Get(ctx context.Context, id, viewerID int64) (*domain.PostView, error) {
	return u.posts.Get(ctx, id, viewerID)
}

// Delete は投稿者本人だけが行える。
func (u *PostUsecase) Delete(ctx context.Context, userID, id int64) error {
	p, err := u.posts.Get(ctx, id, userID)
	if err != nil {
		return err
	}
	if p.UserID != userID {
		return domain.ErrForbidden
	}
	return u.posts.Delete(ctx, id)
}

func (u *PostUsecase) Home(ctx context.Context, userID, cursor int64, limit int) (*Page, error) {
	limit = clampLimit(limit)
	items, err := u.posts.Home(ctx, userID, cursor, limit+1)
	if err != nil {
		return nil, err
	}
	return paginate(items, limit), nil
}

func (u *PostUsecase) ByUser(ctx context.Context, handle string, viewerID, cursor int64, limit int) (*Page, error) {
	author, err := u.users.FindByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	limit = clampLimit(limit)
	items, err := u.posts.ByUser(ctx, author.ID, viewerID, cursor, limit+1)
	if err != nil {
		return nil, err
	}
	return paginate(items, limit), nil
}

func clampLimit(n int) int {
	if n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

// paginate は limit+1 件取得した結果から、次ページの有無を判定する。
func paginate(items []domain.PostView, limit int) *Page {
	page := &Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		page.NextCursor = page.Items[limit-1].ID
	}
	return page
}
