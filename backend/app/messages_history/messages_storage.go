package messageshistory

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func NewStorage(db *pgxpool.Pool) *Storage {
	return &Storage{db: db}
}

func (s *Storage) GetMessageHistoryStorage(ctx context.Context, sessionId string) ([]MessageHistoryData, error) {
	query := `
		SELECT 
			session_id,
			message_id,
			role,
			content,
			created_at,
			feedback
		FROM chat_messages
		WHERE session_id = $1
		ORDER BY created_at ASC;
	`

	rows, err := s.db.Query(ctx, query, sessionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dataList := []MessageHistoryData{}

	for rows.Next() {
		var data MessageHistoryData
		err := rows.Scan(
			&data.SessionId,
			&data.MessageId,
			&data.Role,
			&data.Content,
			&data.CreatedAt,
			&data.Feedback,
		)
		if err != nil {
			return nil, err
		}
		dataList = append(dataList, data)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dataList, nil
}
