package database

import (
	"frontend/database/models"
	"frontend/helper"
	"slices"
)

func AddUserToClass(s *Store, classId int, username string) error {
	return updateClass(s, classId, func(c *models.Class) error {
		if !slices.Contains(c.Users, username) {
			c.Users = append(c.Users, username)
		}
		return nil
	})
}

func RemoveUserFromClass(s *Store, classId int, username string) error {
	return updateClass(s, classId, func(c *models.Class) error {
		c.Users = helper.Remove(c.Users, username)
		return nil
	})
}
