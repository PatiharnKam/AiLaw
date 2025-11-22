package sessionshistory

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

func (s *Storage) GetSessionsHistoryStorage(ctx context.Context, userId string) ([]SessionsHistoryData, error) {
	query := `
		SELECT 
			user_id,
			session_id,
			title,
			created_at,
			last_message_at
		FROM chat_sessions
		WHERE user_id = $1
		ORDER BY last_message_at DESC;
	`

	rows, err := s.db.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dataList := []SessionsHistoryData{}

	for rows.Next() {
		var data SessionsHistoryData
		err := rows.Scan(
			&data.UserId,
			&data.SessionId,
			&data.Title,
			&data.CreatedAt,
			&data.LastMessageAt,
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
