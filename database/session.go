package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go.etcd.io/bbolt"
)

// GenerateSession creates a secure random session ID and stores mapping sessionID -> username
func GenerateSession(s *Store, username string) (string, error) {
	// 32 random bytes = 64 hex characters
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	sessionID := hex.EncodeToString(b)

	err = s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(Buckets["sessions"])
		if b == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		return b.Put([]byte(sessionID), []byte(username))
	})
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

// GetUserFromSession resolves username from session ID
func GetUserFromSession(s *Store, sessionID string) (string, error) {
	var user string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(Buckets["sessions"])
		if b == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		v := b.Get([]byte(sessionID))
		if v == nil {
			return fmt.Errorf("session not found")
		}
		user = string(v)
		return nil
	})
	return user, err
}

func DeleteSession(s *Store, sessionID string) error {
	err := Delete(s, Buckets["sessions"], sessionID)
	return err
}
