package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/whyttea/lenslocked/rand"
)

type Session struct {
	ID     int
	UserID int
	// Token is only set when creating a new session. when looking up a session this will be left empty
	// as we only store the hash of a session token in our db and we cannot reverse it.
	Token     string
	TokenHash string
}

type SessionService struct {
	DB *sql.DB
	// The min number of bytes to be used for each session token.
	// If this value is not set or is less than the MinBytesPerToken const,
	// it will be ignored and MinBytesPerToken will be used instread.
	BytesPerToken int
}

func (ss *SessionService) Create(userID int) (*Session, error) {
	// Create session token
	bytesPerToken := ss.BytesPerToken
	if bytesPerToken < rand.MinBytesPerToken {
		bytesPerToken = rand.MinBytesPerToken
	}
	token, err := rand.String(bytesPerToken)
	if err != nil {
		return nil, fmt.Errorf("create failed at token generation step with: %w", err)
	}
	// hash the token
	session := Session{
		UserID:    userID,
		Token:     token,
		TokenHash: ss.hash(token),
		// Set the token hash
	}
	// we attempt to create a user, if there is a conflict on the user_id field, they must have an active session already.
	// in that case, let's just update the session token_hash
	row := ss.DB.QueryRow(`
	INSERT INTO sessions (user_id, token_hash)
	VALUES ($1, $2) ON CONFLICT (user_id) DO
	UPDATE
	SET token_hash = $2
	RETURNING id;
	`, session.UserID, session.TokenHash)
	err = row.Scan(&session.ID)
	if err != nil {
		return nil, fmt.Errorf("create failed on inserting/updating the db with: %w", err)
	}
	// store the session in our DB
	return &session, nil
}

func (ss *SessionService) User(token string) (*User, error) {
	// Hash the session token

	tokenHash := ss.hash(token)
	var user User

	row := ss.DB.QueryRow(`
	SELECT users.id, users.email, users.password_hash
	FROM sessions 
	JOIN users ON users.id = sessions.user_id
	WHERE sessions.token_hash = $1;
	`, tokenHash)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user failed at fetching from DB due to absent row with it: %w", err)
		}
		return nil, fmt.Errorf("user failed at fetchign from DB: %w", err)
	}

	// // Query the DB for the session with that hash
	// row := ss.DB.QueryRow(`
	// 	SELECT user_id FROM sessions
	// 	WHERE token_hash = $1;
	// 	`, tokenHash)
	// err := row.Scan(&user.ID)
	// if err != nil {
	// 	return nil, fmt.Errorf("user failed at selecting from sessions DB: %w", err)
	// }
	// // Using the user_id fro mthe session, we need to query for the user
	// row = ss.DB.QueryRow(`
	// SELECT email, password_hash
	// FROM users
	// WHERE id = $1;
	// `, user.ID)
	// err = row.Scan(&user.Email, &user.PasswordHash)
	// if err != nil {
	// 	return nil, fmt.Errorf("user failed at selecting from users DB: %w", err)
	// }

	return &user, nil
}

func (ss SessionService) Delete(token string) error {
	tokenHash := ss.hash(token)
	// we do not care about any data being returned from DB so we better use Exec rather than QueryRow or Query
	_, err := ss.DB.Exec(`
	DELETE FROM sessions
	WHERE token_hash = $1;
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete of the user session failed from query to the db with: %w", err)
	}
	return nil
}

func (ss *SessionService) hash(token string) string {
	tokenHash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(tokenHash[:]) // nifty trick to turn Go's byte ARRAY into byte SLICE
}
