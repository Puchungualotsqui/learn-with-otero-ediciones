package database

import (
	"frontend/database/models"
)

var SubjectsNames []string = []string{"literatura", "educacion_fisica", "psicologia", "artes_plasticas", "musica", "valores"}

func CreateSubject(s *Store, name string) error {
	return Save(s, Buckets["subjects"], name, models.Subject{
		Name: name})
}

func CreateInitialSubjects(s *Store) error {
	for _, name := range SubjectsNames {
		if err := CreateSubject(s, name); err != nil {
			return err
		}
	}
	return nil
}
