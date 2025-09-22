package database

import (
	"frontend/auth"
	"frontend/database/models"
	"frontend/helper"
)

// CreateUser stores a new user with hashing + encryption
func CreateUser(s *Store, username, plainPassword, firstName, lastName, role string, encKey []byte) error {
	hashed, err := auth.HashPassword(plainPassword)
	if err != nil {
		return err
	}
	encrypted, err := helper.Encrypt(encKey, plainPassword)
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
