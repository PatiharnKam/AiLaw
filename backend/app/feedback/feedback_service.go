package feedback

import (
	"context"
	"fmt"
)

type Service struct {
	storage FeedbackStorage
}

func NewService(storage FeedbackStorage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) FeedbackService(ctx context.Context, req FeedbackRequest) error {
	err := s.storage.FeedbackStorage(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update message feedback error : %w", err)
	}
	return nil
}
