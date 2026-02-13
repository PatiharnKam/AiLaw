package updatesessionname

import "context"

type UpdateSessionNameService interface {
	UpdateSessionNameService(ctx context.Context, req UpdateSessionNameRequest) error
}

type UpdateSessionNameStorage interface {
	UpdateSessionNameStorage(ctx context.Context, req UpdateSessionNameRequest) error
}

type UpdateSessionNameRequest struct {
	UserID    string `json:"userId" validate:"required"`
	SessionID string `json:"sessionID" validate:"required"`
	NewName   string `json:"newName" validate:"required"`
}
