package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/PatiharnKam/AiLaw/config"
	"github.com/pkoukk/tiktoken-go"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	redis           *redis.Client
	dailyLimit      int64
	maxPromptTokens int
}

func NewQuotaService(redisClient *redis.Client, cfg *config.Quota) *Service {
	return &Service{
		redis:           redisClient,
		dailyLimit:      cfg.DailyLimit,
		maxPromptTokens: cfg.MaxPromptTokens,
	}
}

func (s *Service) GetUserQuotaKey(userID string) string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf("quota:user:%s:date:%s", userID, today)
}

func (s *Service) CheckQuota(ctx context.Context, userID string) (*QuotaStatus, error) {
	key := s.GetUserQuotaKey(userID)

	used, err := s.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		// ยังไม่เคยใช้วันนี้
		used = 0
	} else if err != nil {
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}

	isExceeded := false
	remaining := s.dailyLimit - used
	if remaining <= 0 {
		isExceeded = true
		remaining = 0
	}

	return &QuotaStatus{
		UserID:     userID,
		TokensUsed: used,
		Remaining:  remaining,
		IsExceeded: isExceeded,
	}, nil
}

func (s *Service) ConsumeTokens(ctx context.Context, userID string, totalTokens int64) error {
	key := s.GetUserQuotaKey(userID)

	pipe := s.redis.Pipeline()

	pipe.IncrBy(ctx, key, totalTokens)

	// Set TTL
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	ttl := time.Until(midnight)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to consume tokens: %w", err)
	}

	return nil
}

func (s *Service) CheckPromptLength(text string) (int, error) {
	var encoding *tiktoken.Tiktoken
	encoding, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		return 0, err
	}
	tokens := encoding.Encode(text, nil, nil)
	tokenCount := len(tokens)

	isWithinLimit := tokenCount <= s.maxPromptTokens

	if !isWithinLimit {
		return 0, fmt.Errorf("Prompt exceeds maximum token limit")
	}

	return tokenCount, nil
}
