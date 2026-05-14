package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/sharanch/inkwell/feed-service/internal/model"
	"github.com/sharanch/inkwell/feed-service/internal/repository"
)

const feedCacheTTL = 5 * time.Minute

type FeedService struct {
	repo           *repository.FeedRepository
	rdb            *redis.Client
	blogServiceURL string
	httpClient     *http.Client
}

func NewFeedService(repo *repository.FeedRepository, rdb *redis.Client, blogServiceURL string) *FeedService {
	return &FeedService{
		repo:           repo,
		rdb:            rdb,
		blogServiceURL: blogServiceURL,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
	}
}

// GetFeed returns a ranked list of public posts filtered by user interests.
// Results are cached in Redis per user.
func (s *FeedService) GetFeed(ctx context.Context, userID string, page int) (*model.FeedPage, error) {
	cacheKey := fmt.Sprintf("feed:%s:%d", userID, page)

	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var fp model.FeedPage
		if json.Unmarshal([]byte(cached), &fp) == nil {
			fp.Cached = true
			return &fp, nil
		}
	}

	interests, err := s.repo.GetInterests(ctx, userID)
	if err != nil || len(interests) == 0 {
		interests = []string{} // empty = show all
	}

	postIDs, err := s.repo.GetFeedPostIDs(ctx, userID, interests, page, 20)
	if err != nil {
		return nil, err
	}

	posts, err := s.fetchPostsFromBlogService(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch posts: %w", err)
	}

	fp := &model.FeedPage{Posts: posts, Page: page, PageSize: 20}
	if b, err := json.Marshal(fp); err == nil {
		s.rdb.Set(ctx, cacheKey, string(b), feedCacheTTL)
	}

	return fp, nil
}

func (s *FeedService) SetInterests(ctx context.Context, userID string, tags []string) error {
	if err := s.repo.SetInterests(ctx, userID, tags); err != nil {
		return err
	}
	// Invalidate feed cache for this user
	keys, _ := s.rdb.Keys(ctx, fmt.Sprintf("feed:%s:*", userID)).Result()
	if len(keys) > 0 {
		s.rdb.Del(ctx, keys...)
	}
	return nil
}

func (s *FeedService) GetInterests(ctx context.Context, userID string) ([]string, error) {
	return s.repo.GetInterests(ctx, userID)
}

func (s *FeedService) fetchPostsFromBlogService(ctx context.Context, ids []string) ([]*model.FeedPost, error) {
	if len(ids) == 0 {
		return []*model.FeedPost{}, nil
	}

	body, _ := json.Marshal(ids)
	req, err := http.NewRequestWithContext(ctx, "POST",
		s.blogServiceURL+"/internal/posts/batch",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var posts []*model.FeedPost
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}
	return posts, nil
}
