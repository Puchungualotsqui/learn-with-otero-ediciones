package database

import (
	"frontend/database/models"
)

func CreateSubject(s *Store, internalName, name string) error {
	return Save(s, Buckets["subjects"], internalName, models.Subject{
		InternalName: internalName,
		Name:         name})
}
