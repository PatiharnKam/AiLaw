package feedback

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func NewStorage(db *pgxpool.Pool) *Storage {
	return &Storage{db: db}
}

func (s *Storage) FeedbackStorage(ctx context.Context, req FeedbackRequest) error {
	query := `
		UPDATE chat_messages
		SET feedback = $1
		WHERE message_id = $2 AND role = 'model'
	`
	rows, err := s.db.Exec(ctx, query, req.Feedback, req.MessageID)
	if err != nil {
		return err
	}

	if rows.RowsAffected() != 1 {
		return fmt.Errorf("no rows were updated")
	}

	return nil
}
