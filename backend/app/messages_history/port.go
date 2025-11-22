package messageshistory

import (
	"context"
	"time"
)

type MessageService interface {
	GetMessageHistoryService(ctx context.Context, req MessageHistoryRequest) ([]MessageHistoryResponse, error)
}
type MessageStorage interface {
	GetMessageHistoryStorage(ctx context.Context, sessionId string) ([]MessageHistoryData, error)
}

type MessageHistoryRequest struct {
	SessionId string `json:"sessionId" validate:"required,uuid4"`
}

type MessageHistoryResponse struct {
	SessionId string    `json:"sessionId"`
	MessageId string    `json:"messageId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	Feedback  *string   `json:"feedback"`
}

type MessageHistoryData struct {
	SessionId string    `db:"session_id"`
	MessageId string    `db:"message_id"`
	Role      string    `db:"role"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
	Feedback  *string   `db:"feedback"`
}
