package database

import (
	"fmt"
	"frontend/auth"
	"frontend/database/models"
	"frontend/helper"
	"os"
	"strconv"
	"strings"
)

// CreateUser stores a new user with hashing + encryption
// plainPassword and username are optional
func CreateUser(s *Store, username, plainPassword, firstName, lastName, role, school, grade, email, phoneNumber string) (*models.User, error) {
	// --- Load encryption key
	encKey := os.Getenv("ENC_KEY")
	if encKey == "" {
		return nil, fmt.Errorf("ENC_KEY not found in environment")
	}

	if strings.TrimSpace(plainPassword) == "" {
		plainPassword = auth.GenerateRandomPassword(10)
	}

	// --- Hash and encrypt password
	hashed, err := auth.HashPassword(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	encrypted, err := auth.Encrypt([]byte(encKey), plainPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}

	// --- Generate username
	if strings.TrimSpace(username) == "" {
		username = generateUsername(s, firstName, lastName, school, grade)
	}

	u := models.User{
		Username:          username,
		PasswordHashed:    hashed,
		PasswordNotHashed: encrypted,
		FirstName:         firstName,
		LastName:          lastName,
		School:            school,
		Grade:             grade,
		PhoneNumber:       phoneNumber,
		Email:             email,
		Role:              role,
	}

	// --- Save to database
	if err := Save(s, Buckets["users"], u.Username, u); err != nil {
		return nil, err
	}

	u.PasswordNotHashed = plainPassword

	return &u, nil
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

func generateUsername(s *Store, firstName, lastName, school, grade string) string {
	// Helper: lowercases and takes first word
	clean := func(v string) string {
		parts := strings.Fields(v)
		for i := range parts {
			parts[i] = strings.ToLower(parts[i])
		}
		parts = helper.Remove(parts, "colegio") // removes word "colegio"
		if len(parts) == 0 {
			return ""
		}
		return parts[0]
	}

	f := clean(firstName)
	l := clean(lastName)
	schoolPart := clean(school)

	// ✅ Handle special or minimal cases
	if f == "admin" {
		return "admin"
	}
	if f == "" && l == "" {
		return "user"
	}
	if l == "" {
		l = f
	}
	if schoolPart == "" {
		schoolPart = "school"
	}

	// Base like "juanp-germania"
	base := fmt.Sprintf("%s%s-%s", f, string(l[0]), schoolPart)

	// Check uniqueness
	username := base
	counter := 1
	for {
		exists, _ := Exists(s, Buckets["users"], username)
		if !exists {
			break
		}

		// Try variant with grade
		username = fmt.Sprintf("%s%s-%s", f, string(l[0]), grade)
		exists, _ = Exists(s, Buckets["users"], username)
		if !exists {
			break
		}

		// Add numeric suffix
		username = fmt.Sprintf("%s%d", base, counter)
		counter++
	}

	return username
}
