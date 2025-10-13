package database

import (
	"fmt"
	"frontend/auth"
	"frontend/database/models"
	"frontend/helper"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// CreateUser stores a new user with hashing + encryption
func CreateUser(s *Store, username, plainPassword, firstName, lastName, role string) error {
	err := godotenv.Load(".venv")
	if err != nil {
		return fmt.Errorf("Error loading .env file")
	}

	encKey := os.Getenv("ENC_KEY")

	hashed, err := auth.HashPassword(plainPassword)
	if err != nil {
		return err
	}
	encrypted, err := auth.Encrypt([]byte(encKey), plainPassword)
	if err != nil {
		return err
	}

	u := models.User{
		Username:          username,
		PasswordHashed:    hashed,
		PasswordNotHashed: encrypted,
		FirstName:         firstName,
		LastName:          lastName,
		Role:              role,
	}

	return Save(s, Buckets["users"], u.Username, u)
}

func DeleteUser(s *Store, username string) error {
	user, err := Get[models.User](s, Buckets["users"], username)
	if err != nil {
		return fmt.Errorf("Error getting user to remove: %w", err)
	}

	for _, class := range user.Classes {
		if err := UpdateWithPrefix(s, Buckets["classes"], func(t *models.Class) error {
			t.Users = helper.Remove(t.Users, username)
			return nil
		}, strconv.Itoa(class)); err != nil {
			return fmt.Errorf("Error updating class: %w", err)
		}
	}

	return Delete(s, Buckets["users"], username)
}
