package deletesession

import "context"

type DeleteChatSessionService interface {
	DeleteChatSessionService(ctx context.Context, req DeleteChatSessionRequest) error
}

type DeleteChatSessionStorage interface {
	DeleteChatSessionStorage(ctx context.Context, req DeleteChatSessionRequest) error
}

type DeleteChatSessionRequest struct {
	UserID    string `json:"userId" validate:"required"`
	SessionID string `json:"sessionID" validate:"required"`
}
