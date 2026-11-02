// Package memory はテストやDBなしの動作確認に使う、メモリ上のリポジトリ実装。
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sora804/sns/backend/internal/domain"
)

// Store は全リポジトリが共有するデータ。
type Store struct {
	mu        sync.Mutex
	nextUser  int64
	nextPost  int64
	users     map[int64]*domain.User
	sessions  map[string]session
	posts     map[int64]*domain.Post
	deleted   map[int64]bool
	follows   map[[2]int64]bool
	likes     map[[2]int64]bool
	Users     *UserRepo
	Sessions  *SessionRepo
	Posts     *PostRepo
	Follows   *FollowRepo
	LikeStore *LikeRepo
}

type session struct {
	userID    int64
	expiresAt time.Time
}

func New() *Store {
	s := &Store{
		users:    map[int64]*domain.User{},
		sessions: map[string]session{},
		posts:    map[int64]*domain.Post{},
		deleted:  map[int64]bool{},
		follows:  map[[2]int64]bool{},
		likes:    map[[2]int64]bool{},
	}
	s.Users = &UserRepo{s}
	s.Sessions = &SessionRepo{s}
	s.Posts = &PostRepo{s}
	s.Follows = &FollowRepo{s}
	s.LikeStore = &LikeRepo{s}
	return s
}

// ---- users ----

type UserRepo struct{ s *Store }

func (r *UserRepo) Create(_ context.Context, u *domain.User) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, e := range r.s.users {
		if strings.EqualFold(e.Handle, u.Handle) || e.Email == u.Email {
			return domain.ErrConflict
		}
	}
	r.s.nextUser++
	u.ID = r.s.nextUser
	u.CreatedAt = time.Now()
	c := *u
	r.s.users[u.ID] = &c
	return nil
}

func (r *UserRepo) find(match func(*domain.User) bool) (*domain.User, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, u := range r.s.users {
		if match(u) {
			c := *u
			return &c, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *UserRepo) FindByID(_ context.Context, id int64) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return u.ID == id })
}

func (r *UserRepo) FindByHandle(_ context.Context, h string) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return strings.EqualFold(u.Handle, h) })
}

func (r *UserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	email = strings.ToLower(email)
	return r.find(func(u *domain.User) bool { return u.Email == email })
}

func (r *UserRepo) Profile(ctx context.Context, handle string, viewerID int64) (*domain.UserProfile, error) {
	u, err := r.FindByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	p := &domain.UserProfile{User: *u}
	for k := range r.s.follows {
		if k[1] == u.ID {
			p.FollowerCount++
		}
		if k[0] == u.ID {
			p.FollowingCount++
		}
	}
	for id, post := range r.s.posts {
		if post.UserID == u.ID && !r.s.deleted[id] {
			p.PostCount++
		}
	}
	p.IsFollowing = r.s.follows[[2]int64{viewerID, u.ID}]
	return p, nil
}

// ---- sessions ----

type SessionRepo struct{ s *Store }

func (r *SessionRepo) Create(_ context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.sessions[tokenHash] = session{userID, expiresAt}
	return nil
}

func (r *SessionRepo) FindUserID(_ context.Context, tokenHash string, now time.Time) (int64, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	se, ok := r.s.sessions[tokenHash]
	if !ok || !se.expiresAt.After(now) {
		return 0, domain.ErrNotFound
	}
	return se.userID, nil
}

func (r *SessionRepo) Delete(_ context.Context, tokenHash string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	delete(r.s.sessions, tokenHash)
	return nil
}

// ---- posts ----

type PostRepo struct{ s *Store }

func (r *PostRepo) Create(_ context.Context, p *domain.Post) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.nextPost++
	p.ID = r.s.nextPost
	p.CreatedAt = time.Now()
	c := *p
	r.s.posts[p.ID] = &c
	return nil
}

// view は呼び出し側で mu を保持していること。
func (s *Store) view(p *domain.Post, viewerID int64) domain.PostView {
	v := domain.PostView{Post: *p}
	if u := s.users[p.UserID]; u != nil {
		v.AuthorHandle, v.AuthorDisplayName = u.Handle, u.DisplayName
	}
	for k := range s.likes {
		if k[1] == p.ID {
			v.LikeCount++
		}
	}
	v.LikedByMe = s.likes[[2]int64{viewerID, p.ID}]
	return v
}

func (r *PostRepo) Get(_ context.Context, id, viewerID int64) (*domain.PostView, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	p := r.s.posts[id]
	if p == nil || r.s.deleted[id] {
		return nil, domain.ErrNotFound
	}
	v := r.s.view(p, viewerID)
	return &v, nil
}

func (r *PostRepo) Delete(_ context.Context, id int64) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if r.s.posts[id] == nil || r.s.deleted[id] {
		return domain.ErrNotFound
	}
	r.s.deleted[id] = true
	return nil
}

func (r *PostRepo) list(viewerID, before int64, limit int, include func(*domain.Post) bool) []domain.PostView {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	var posts []*domain.Post
	for id, p := range r.s.posts {
		if r.s.deleted[id] || (before != 0 && id >= before) || !include(p) {
			continue
		}
		posts = append(posts, p)
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].ID > posts[j].ID })
	if len(posts) > limit {
		posts = posts[:limit]
	}
	out := make([]domain.PostView, 0, len(posts))
	for _, p := range posts {
		out = append(out, r.s.view(p, viewerID))
	}
	return out
}

func (r *PostRepo) Home(_ context.Context, viewerID, before int64, limit int) ([]domain.PostView, error) {
	return r.list(viewerID, before, limit, func(p *domain.Post) bool {
		return p.UserID == viewerID || r.s.follows[[2]int64{viewerID, p.UserID}]
	}), nil
}

func (r *PostRepo) ByUser(_ context.Context, authorID, viewerID, before int64, limit int) ([]domain.PostView, error) {
	return r.list(viewerID, before, limit, func(p *domain.Post) bool { return p.UserID == authorID }), nil
}

// ---- follows / likes ----

type FollowRepo struct{ s *Store }

func (r *FollowRepo) Follow(_ context.Context, a, b int64) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.follows[[2]int64{a, b}] = true
	return nil
}

func (r *FollowRepo) Unfollow(_ context.Context, a, b int64) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	delete(r.s.follows, [2]int64{a, b})
	return nil
}

type LikeRepo struct{ s *Store }

func (r *LikeRepo) Like(_ context.Context, userID, postID int64) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.likes[[2]int64{userID, postID}] = true
	return nil
}

func (r *LikeRepo) Unlike(_ context.Context, userID, postID int64) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	delete(r.s.likes, [2]int64{userID, postID})
	return nil
}
