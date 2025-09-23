package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"
	"slices"

	"go.etcd.io/bbolt"
)

func CreateClass(s *Store, subject string) (*models.Class, error) {
	var c *models.Class
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["classes"])
		if err != nil {
			return err
		}

		id64, _ := b.NextSequence()
		c = &models.Class{
			Id:      int(id64),
			Subject: subject,
			Users:   []string{},
		}

		data, err := json.Marshal(c)
		if err != nil {
			return err
		}

		key := fmt.Appendf(nil, "%d", c.Id)
		return b.Put(key, data)
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func updateClass(s *Store, classId int, updater func(*models.Class) error) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(Buckets["classes"])
		if b == nil {
			return fmt.Errorf("bucket %s not found", Buckets["classes"])
		}

		key := fmt.Appendf(nil, "%d", classId)
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("class %d not found", classId)
		}

		var c models.Class
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		// Apply caller's logic
		if err := updater(&c); err != nil {
			return err
		}

		data, _ := json.Marshal(c)
		return b.Put(key, data)
	})
}

func ListClassesForUser(s *Store, username string) ([]models.Class, error) {
	var results []models.Class

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(Buckets["classes"])
		if b == nil {
			return fmt.Errorf("bucket %s not found", Buckets["classes"])
		}

		return b.ForEach(func(k, v []byte) error {
			var c models.Class
			if err := json.Unmarshal(v, &c); err != nil {
				return err
			}

			if slices.Contains(c.Users, username) {
				results = append(results, c)
			}
			return nil
		})
	})

	return results, err
}
