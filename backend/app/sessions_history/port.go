package sessionshistory

import (
	"context"
	"time"
)

type SessionService interface {
	GetSessionsHistoryService(ctx context.Context, req SessionsHistoryRequest) ([]SessionsHistoryResponse, error)
}
type SessionStorage interface {
	GetSessionsHistoryStorage(ctx context.Context, userId string) ([]SessionsHistoryData, error)
}

type SessionsHistoryRequest struct {
	UserId string `json:"userId" validate:"required,uuid4"`
}

type SessionsHistoryResponse struct {
	UserId        string    `json:"userId"`
	SessionId     string    `json:"sessionId"`
	Title         string    `json:"title"`
	CreatedAt     time.Time `json:"createdAt"`
	LastMessageAt time.Time `json:"lastMessageAt"`
}

type SessionsHistoryData struct {
	UserId        string    `db:"user_id"`
	SessionId     string    `db:"session_id"`
	Title         string    `db:"title"`
	CreatedAt     time.Time `db:"created_at"`
	LastMessageAt time.Time `db:"last_message_at"`
}
