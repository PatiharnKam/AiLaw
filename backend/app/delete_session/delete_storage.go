package deletesession

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

func (s *Storage) DeleteChatSessionStorage(ctx context.Context, req DeleteChatSessionRequest) error {
	query := `
		DELETE FROM chat_sessions
		WHERE user_id = $1 AND session_id = $2;
	`
	rows, err := s.db.Exec(ctx, query, req.UserID, req.SessionID)
	if err != nil {
		return err
	}

	if rows.RowsAffected() != 1 {
		return fmt.Errorf("no rows were deleted")
	}

	return nil
}
