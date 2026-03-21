package sqlite

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
)

func (s *Store) GenerateSession(username string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	sessionID := hex.EncodeToString(b)

	_, err := s.DB.Exec(`
		INSERT INTO sessions (session_id, username)
		VALUES (?, ?)
	`, sessionID, username)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *Store) GetUserFromSession(sessionID string) (string, error) {
	row := s.DB.QueryRow(`
		SELECT username
		FROM sessions
		WHERE session_id = ?
	`, sessionID)

	var username string
	if err := row.Scan(&username); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("session not found")
		}
		return "", err
	}

	return username, nil
}

func (s *Store) DeleteSession(sessionID string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE session_id = ?`, sessionID)
	return err
}
