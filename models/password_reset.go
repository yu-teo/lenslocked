package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/whyttea/lenslocked/rand"
)

const (
	DefaultResetDuration = 2 * time.Hour
)

type PasswordReset struct {
	ID     int
	UserId int
	// Token is only set when a PasswordReset is being created. We do not store it in db and cannot reverse-engineer it.
	Token     string
	TokenHash string
	ExpiresAt time.Time //alternatively, could use sql.NullTime if we allowed null in Time - check migrations for this table
}

type PasswordResetService struct {
	DB *sql.DB
	// The min number of bytes to be used for each session token.
	// If this value is not set or is less than the MinBytesPerToken const,
	// it will be ignored and MinBytesPerToken will be used instread.
	BytesPerToken int
	// Duration is the amount of time that a PasswordReset is valid for.
	// defaults to DefaultResetDuration
	Duration time.Duration
}

func (prs *PasswordResetService) Create(email string) (*PasswordReset, error) {
	// Verify we have a valid email address for user and get that user's id
	email = strings.ToLower(email)
	var userID int
	row := prs.DB.QueryRow(`
	SELECT id FROM users WHERE email = $1;`, email)
	err := row.Scan(&userID)
	if err != nil {
		// TODO: consider returning specific error when the user does not exist
		return nil, fmt.Errorf("createing password reset service: %w", err)
	}
	// Create password reset token
	bytesPerToken := prs.BytesPerToken
	if bytesPerToken < rand.MinBytesPerToken {
		bytesPerToken = rand.MinBytesPerToken
	}
	token, err := rand.String(bytesPerToken)
	if err != nil {
		return nil, fmt.Errorf("create failed at password reset token generation step with: %w", err)
	}

	duration := prs.Duration
	if duration == 0 {
		duration = DefaultResetDuration
	}

	pwReset := PasswordReset{
		UserId:    userID,
		Token:     token,
		TokenHash: prs.hash(token),
		ExpiresAt: time.Now().Add(duration),
	}

	// Insert the PasswordReset into DB
	row = prs.DB.QueryRow(`
	INSERT INTO password_resets (user_id, token_hash, expires_at)
	VALUES ($1, $2, $3) ON CONFLICT (user_id) DO 
	UPDATE
	SET token_hash = $2, expires_at = $3
	RETURNING id; 
	`, pwReset.UserId, pwReset.TokenHash, pwReset.ExpiresAt)
	err = row.Scan(&pwReset.ID)
	if err != nil {
		return nil, fmt.Errorf("PasswordResetService.Create: %w", err)
	}

	return &pwReset, nil
}

func (prs *PasswordResetService) Consume(token string) (*User, error) {
	tokenHash := prs.hash(token)
	var user User
	var pwReset PasswordReset

	row := prs.DB.QueryRow(`
	SELECT password_resets.id, password_resets.expires_at, users.id, users.email, users.password_hash
	FROM password_resets JOIN users ON users.id = password_resets.user_id
	WHERE password_resets.token_hash = $1;
	`, tokenHash)
	err := row.Scan(&pwReset.ID, &pwReset.ExpiresAt,
		&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("PasswordResetService.Consume failed with: %w", err)
	}

	if time.Now().After(pwReset.ExpiresAt) {
		return nil, fmt.Errorf("token has expired: %v", token)
	}

	err = prs.delete(pwReset.ID)
	if err != nil {
		return nil, fmt.Errorf("PasswordResetService.Consume failed deleting the code reset with: %w", err)
	}

	return &user, nil
}

func (prs *PasswordResetService) hash(token string) string {
	tokenHash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(tokenHash[:]) // nifty trick to turn Go's byte ARRAY into byte SLICE
}

func (prs *PasswordResetService) delete(id int) error {
	_, err := prs.DB.Exec(`
	DELETE FROM password_resets
	WHERE id = $1;`, id)
	if err != nil {
		return fmt.Errorf("deleting password hash failed with: %w", err)
	}
	return nil
}
