package usecase

import (
	"context"

	"github.com/sora804/sns/backend/internal/domain"
)

type SocialUsecase struct {
	users   domain.UserRepository
	posts   domain.PostRepository
	follows domain.FollowRepository
	likes   domain.LikeRepository
}

func NewSocialUsecase(u domain.UserRepository, p domain.PostRepository, f domain.FollowRepository, l domain.LikeRepository) *SocialUsecase {
	return &SocialUsecase{users: u, posts: p, follows: f, likes: l}
}

func (s *SocialUsecase) Profile(ctx context.Context, handle string, viewerID int64) (*domain.UserProfile, error) {
	return s.users.Profile(ctx, handle, viewerID)
}

func (s *SocialUsecase) Follow(ctx context.Context, me int64, handle string) error {
	target, err := s.users.FindByHandle(ctx, handle)
	if err != nil {
		return err
	}
	if target.ID == me {
		return &domain.ValidationError{Field: "handle", Message: "自分自身はフォローできません"}
	}
	return s.follows.Follow(ctx, me, target.ID)
}

func (s *SocialUsecase) Unfollow(ctx context.Context, me int64, handle string) error {
	target, err := s.users.FindByHandle(ctx, handle)
	if err != nil {
		return err
	}
	return s.follows.Unfollow(ctx, me, target.ID)
}

func (s *SocialUsecase) Like(ctx context.Context, me, postID int64) error {
	if _, err := s.posts.Get(ctx, postID, me); err != nil {
		return err
	}
	return s.likes.Like(ctx, me, postID)
}

func (s *SocialUsecase) Unlike(ctx context.Context, me, postID int64) error {
	if _, err := s.posts.Get(ctx, postID, me); err != nil {
		return err
	}
	return s.likes.Unlike(ctx, me, postID)
}
