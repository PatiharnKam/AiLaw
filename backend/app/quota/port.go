package quota

import "context"

type QuotaStatus struct {
	UserID     string `json:"user_id"`
	TokensUsed int64  `json:"tokens_used"`
	Remaining  int64  `json:"remaining"`
	IsExceeded bool   `json:"is_exceeded"`
}

type QuotaService interface {
	CheckQuota(ctx context.Context, userID string) (*QuotaStatus, error)
	ConsumeTokens(ctx context.Context, userID string, totalTokens int64) error
}
