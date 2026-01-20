package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type storage struct {
	db *pgxpool.Pool
}

func NewStorage(db *pgxpool.Pool) *storage {
	return &storage{db: db}
}

func (s *storage) CreateSession(ctx context.Context, req CreateChatSessionRequest) (*CreateChatSessionResponse, error) {
	query := `INSERT INTO chat_sessions
				(session_id, user_id, title, created_at, last_message_at)
				VALUES ($1, $2, $3, $4, $5)`
	now := time.Now()
	sessionId := uuid.NewString()
	_, err := s.db.Exec(ctx, query,
		sessionId,
		req.UserId,
		req.Title,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("error when insert data: %v", err)
	}
	data := CreateChatSessionResponse{
		SessionId: sessionId,
	}
	return &data, nil
}

func (s *storage) UpdateLastMessageAt(ctx context.Context, userId string, sessionId string) error {
	query := `
		UPDATE chat_sessions
		SET last_message_at = $3
		WHERE user_id = $1 AND session_id = $2
	`

	now := time.Now()

	cmdTag, err := s.db.Exec(ctx, query, userId, sessionId, now)
	if err != nil {
		return fmt.Errorf("error when updating data: %v", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("failed to update: row not found")
	}

	return nil
}

func (s *storage) SaveUserMessage(ctx context.Context, sessionId, userMessage string) error {
	query := `INSERT INTO chat_messages 
				(message_id, session_id, role ,content, created_at)
			VALUES ($1, $2, $3, $4, $5)`
	_, err := s.db.Exec(ctx, query,
		uuid.NewString(),
		sessionId,
		"user",
		userMessage,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("error when insert data: %v", err)
	}
	return nil
}

func (s *storage) SaveModelMessage(ctx context.Context, sessionId string, modelDetail ModelMessageDetail) error {
	query := `INSERT INTO chat_messages 
			(message_id, session_id, role ,content, created_at , feedback,
			prompt_tokens, completion_tokens, response_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.db.Exec(ctx, query,
		uuid.NewString(),
		sessionId,
		"model",
		modelDetail.Content,
		time.Now(),
		modelDetail.Feedback,
		modelDetail.PromptTokens,
		modelDetail.CompletionTokens,
		modelDetail.ResponseTime,
	)
	if err != nil {
		return fmt.Errorf("error when insert data: %v", err)
	}
	return nil
}
