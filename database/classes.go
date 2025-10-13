package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"
	"frontend/helper"
	"strconv"

	"go.etcd.io/bbolt"
)

var Grades []string = []string{
	"1primaria", "2primaria", "3primaria", "4primaria", "5primaria", "6primaria",
	"1secundaria", "2secundaria", "3secundaria", "4secundaria", "5secundaria", "6secundaria",
}

func CreateClass(s *Store, name, description, subject, grade string) (*models.Class, error) {
	var c *models.Class
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["classes"])
		if err != nil {
			return err
		}

		id64, err := b.NextSequence()
		if err != nil {
			return err
		}

		c = &models.Class{
			Id:          int(id64),
			Name:        name,
			Description: description,
			Grade:       grade,
			Subject:     subject,
			Users:       []string{},
		}

		data, err := json.Marshal(c)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("%d", c.Id)
		return b.Put([]byte(key), data)
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func DeleteClass(s *Store, classId int) error {
	classIdString := strconv.Itoa(classId)

	class, err := Get[models.Class](s, Buckets["classes"], classIdString)
	if err != nil {
		return fmt.Errorf("Error getting class to remove: %w", err)
	}

	for _, user := range class.Users {
		if err := UpdateWithPrefix(s, Buckets["users"], func(t *models.User) error {
			t.Classes = helper.Remove(t.Classes, classId)
			return nil
		}, user); err != nil {
			return fmt.Errorf("Error updating class: %w", err)
		}
	}

	return Delete(s, Buckets["classes"], classIdString)
}
