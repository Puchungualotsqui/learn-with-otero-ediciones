package database

import (
	"fmt"
	"frontend/database/models"
	"frontend/helper"
	"strconv"
)

func AddUserToClass(s *Store, classId int, username string) error {
	if err := UpdateWithPrefix(s, Buckets["classes"], func(t *models.Class) error {
		t.Users = append(t.Users, username)
		return nil
	}, strconv.Itoa(classId)); err != nil {
		return fmt.Errorf("Error updating Class: %w", err)
	}

	if err := UpdateWithPrefix(s, Buckets["users"], func(t *models.User) error {
		t.Classes = append(t.Classes, classId)
		return nil
	}, username); err != nil {
		return fmt.Errorf("Error updating user: %w", err)
	}
	return nil
}

func RemoveUserFromClass(s *Store, classId int, username string) error {
	if err := UpdateWithPrefix(s, Buckets["classes"], func(t *models.Class) error {
		t.Users = helper.RemoveElement(t.Users, username)
		return nil
	}, strconv.Itoa(classId)); err != nil {
		return fmt.Errorf("Error updating Class: %w", err)
	}

	if err := UpdateWithPrefix(s, Buckets["users"], func(t *models.User) error {
		t.Classes = helper.RemoveElement(t.Classes, classId)
		return nil
	}, username); err != nil {
		return fmt.Errorf("Error updating user: %w", err)
	}
	return nil
}
