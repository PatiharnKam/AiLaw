package updatesessionname

import (
	"context"
	"fmt"
)

type Service struct {
	storage UpdateSessionNameStorage
}

func NewService(storage UpdateSessionNameStorage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) UpdateSessionNameService(ctx context.Context, req UpdateSessionNameRequest) error {
	err := s.storage.UpdateSessionNameStorage(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update session name in storage error : %w", err)
	}

	return nil
}
