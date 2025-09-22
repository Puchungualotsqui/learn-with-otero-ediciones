package database

import "frontend/database/models"

func CreateSubject(s *Store, name string) error {
	return Save(s, Buckets["ubjects"], name, models.Subject{Name: name})
}
