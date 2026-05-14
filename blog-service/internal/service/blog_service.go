package service

import (
	"context"
	"fmt"
	"log"

	"github.com/sharanch/inkwell/blog-service/internal/model"
	"github.com/sharanch/inkwell/blog-service/internal/repository"
)

type BlogService struct {
	repo   *repository.PostRepository
	feed   *FeedClient
}

func NewBlogService(repo *repository.PostRepository, feed *FeedClient) *BlogService {
	return &BlogService{repo: repo, feed: feed}
}

func (s *BlogService) CreatePost(ctx context.Context, authorID string, req *model.CreatePostRequest) (*model.Post, error) {
	post, err := s.repo.Create(ctx, authorID, req)
	if err != nil {
		return nil, err
	}
	if post.Visibility == model.VisibilityPublic {
		go s.indexPost(post)
	}
	return post, nil
}

func (s *BlogService) GetPost(ctx context.Context, id, requestingUserID string) (*model.Post, error) {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post.Visibility == model.VisibilityPrivate && post.AuthorID != requestingUserID {
		return nil, fmt.Errorf("not found")
	}
	go s.repo.IncrementView(context.Background(), id)
	return post, nil
}

func (s *BlogService) UpdatePost(ctx context.Context, id, authorID string, req *model.UpdatePostRequest) (*model.Post, error) {
	post, err := s.repo.Update(ctx, id, authorID, req)
	if err != nil {
		return nil, err
	}
	if post.Visibility == model.VisibilityPublic {
		go s.indexPost(post)
	}
	return post, nil
}

func (s *BlogService) DeletePost(ctx context.Context, id, authorID string) error {
	return s.repo.Delete(ctx, id, authorID)
}

func (s *BlogService) ListPosts(ctx context.Context, q *model.ListPostsQuery) (*model.PostsPage, error) {
	return s.repo.List(ctx, q)
}

func (s *BlogService) ToggleLike(ctx context.Context, postID, userID string) (bool, error) {
	return s.repo.ToggleLike(ctx, postID, userID)
}

func (s *BlogService) GetByIDs(ctx context.Context, ids []string) ([]*model.Post, error) {
	return s.repo.GetByIDs(ctx, ids)
}

func (s *BlogService) indexPost(post *model.Post) {
	if err := s.feed.IndexPost(context.Background(), post.ID, post.AuthorID, post.Tags); err != nil {
		log.Printf("WARN: failed to index post %s in feed: %v", post.ID, err)
	}
}
