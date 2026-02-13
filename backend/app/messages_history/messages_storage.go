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
			'user' AS role,
			content,
			created_at,
			NULL AS feedback
		FROM user_messages
		WHERE session_id = $1

		UNION ALL

		SELECT 
			session_id,
			message_id,
			'model' AS role,
			content,
			created_at,
			feedback
		FROM model_messages
		WHERE session_id = $1

		ORDER BY created_at ASC;
	`

	rows, err := s.db.Query(ctx, query, sessionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataList []MessageHistoryData

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
