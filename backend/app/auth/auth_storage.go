package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type authStorage struct {
	db *pgxpool.Pool
}

func NewStorage(db *pgxpool.Pool) *authStorage {
	return &authStorage{db: db}
}

func (s *authStorage) CheckUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT user_id, email FROM users WHERE email = $1`

	user := User{}
	err := s.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	return &user, nil
}

func (s *authStorage) CreateUser(ctx context.Context, user User) error {
	query := `INSERT INTO users
				(user_id, email, username, created_at, picture, updated_at)
				Values($1, $2, $3, $4, $5, $6)`

	_, err := s.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.CreatedAt,
		user.Picture,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error when insert user data: %v", err)
	}

	return nil
}

func (s *authStorage) UpdateUser(ctx context.Context, user User) error {
	query := `UPDATE users
			  SET username = $3, picture = $4 , updated_at = $5
			  WHERE user_id = $1 AND email = $2`
	cmdTag, err := s.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.Picture,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error when updating user data: %v", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("failed to update: row not found")
	}

	return nil
}

func (s *authStorage) StoreRefreshToken(ctx context.Context, userID, token string, expiredAt time.Time) error {

	query := `INSERT INTO refresh_tokens
            (user_id, token, expires_at, created_at)
          VALUES ($1, $2, $3, $4)`

	_, err := s.db.Exec(ctx, query, userID, token, expiredAt, time.Now())
	if err != nil {
		return fmt.Errorf("error when insert token: %v", err)
	}

	return nil
}
func (s *authStorage) ValidateRefreshToken(ctx context.Context, userID, token string) (bool, error) {
	query := `SELECT COUNT(1) 
              FROM refresh_tokens 
              WHERE user_id = $1 AND token = $2 AND expires_at > $3`

	var count int
	err := s.db.QueryRow(ctx, query, userID, token, time.Now()).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking token validity: %v", err)
	}

	if count > 0 {
		return true, nil
	}

	return false, nil
}

func (s *authStorage) DeleteRefreshTokens(ctx context.Context, refreshToken string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`

	cmdTag, err := s.db.Exec(ctx, query, refreshToken)
	if err != nil {
		return fmt.Errorf("error deleting refresh tokens: %v", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("error deleting refresh tokens: no row found")
	}

	return nil
}
