package updatesessionname

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type storage struct {
	db *pgxpool.Pool
}

func NewStorage(db *pgxpool.Pool) *storage {
	return &storage{db: db}
}

func (s *storage) UpdateSessionNameStorage(ctx context.Context, req UpdateSessionNameRequest) error {
	query := `UPDATE chat_sessions 
			  SET title = $1, updated_at = $2
			  WHERE session_id = $3 AND user_id = $4`

	rowAffected, err := s.db.Exec(ctx, query, req.NewName, time.Now(), req.SessionID, req.UserID)

	if err != nil {
		return err
	}

	if rowAffected.RowsAffected() == 0 {
		return fmt.Errorf("session not found or unauthorized")
	}

	return nil
}
