package deletechatsession

import (
	"context"
	"fmt"
)

type Service struct {
	storage DeleteChatSessionStorage
}

func NewService(storage DeleteChatSessionStorage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) DeleteChatSessionService(ctx context.Context, req DeleteChatSessionRequest) error {
	err := s.storage.DeleteChatSessionStorage(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete chat session in storage error : %w", err)
	}

	return nil
}
