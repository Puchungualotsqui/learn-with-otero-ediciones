package database

import (
	"fmt"
	"frontend/auth"
	"frontend/database/models"
	"os"

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
