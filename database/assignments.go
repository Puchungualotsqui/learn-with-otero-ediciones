package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"

	"go.etcd.io/bbolt"
)

func CreateAssignment(s *Store, classId int, title, description, dueDate string) (*models.Assignment, error) {
	var a *models.Assignment

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["assignments"])
		if err != nil {
			return err
		}

		// Unique per bucket (but global). We'll scope it per class by prefix.
		id64, _ := b.NextSequence()
		a = &models.Assignment{
			Id:          int(id64),
			ClassId:     classId,
			Title:       title,
			Description: description,
			DueDate:     dueDate,
		}

		data, err := json.Marshal(a)
		if err != nil {
			return err
		}

		// Key pattern: "classId:assignmentId"
		key := fmt.Sprintf("%d:%d", classId, a.Id)

		return b.Put([]byte(key), data)
	})

	if err != nil {
		return nil, err
	}
	return a, nil
}
