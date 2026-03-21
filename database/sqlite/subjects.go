package sqlite

import (
	"fmt"
	"frontend/database/models"
	"strings"
)

var SubjectsNames []string = []string{"literatura", "educacion_fisica", "psicologia", "artes_plasticas", "musica", "valores"}

type SubjectRename struct {
	Old string
	New string
}

func (s *Store) CreateSubjects(names []string) error {
	if len(names) == 0 {
		return nil
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO subjects (name)
		VALUES (?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, err := stmt.Exec(name); err != nil {
			return fmt.Errorf("create subject %q: %w", name, err)
		}
	}

	return tx.Commit()
}

func (s *Store) DeleteSubjects(names []string) error {
	if len(names) == 0 {
		return nil
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		var count int
		if err := tx.QueryRow(`
			SELECT
				(SELECT COUNT(*) FROM classes WHERE subject = ?) +
				(SELECT COUNT(*) FROM assets   WHERE subject = ?)
		`, name, name).Scan(&count); err != nil {
			return fmt.Errorf("check subject %q usage: %w", name, err)
		}

		if count > 0 {
			return fmt.Errorf("cannot delete subject %q because it is in use", name)
		}

		if _, err := tx.Exec(`
			DELETE FROM subjects
			WHERE name = ?
		`, name); err != nil {
			return fmt.Errorf("delete subject %q: %w", name, err)
		}
	}

	return tx.Commit()
}

func (s *Store) SubjectInUse(name string) (bool, error) {
	var count int

	err := s.DB.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM classes WHERE subject = ?) +
			(SELECT COUNT(*) FROM assets   WHERE subject = ?)
	`, name, name).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Store) RenameSubjects(renames []SubjectRename) error {
	if len(renames) == 0 {
		return nil
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, pair := range renames {
		oldName := strings.TrimSpace(pair.Old)
		newName := strings.TrimSpace(pair.New)

		if oldName == "" || newName == "" || oldName == newName {
			continue
		}

		// 1. Ensure new subject exists
		if _, err := tx.Exec(`
			INSERT OR IGNORE INTO subjects (name)
			VALUES (?)
		`, newName); err != nil {
			return fmt.Errorf("create renamed subject %q: %w", newName, err)
		}

		// 2. Update classes referencing the old subject
		if _, err := tx.Exec(`
			UPDATE classes
			SET subject = ?
			WHERE subject = ?
		`, newName, oldName); err != nil {
			return fmt.Errorf("update classes subject %q -> %q: %w", oldName, newName, err)
		}

		// 3. Update assets referencing the old subject
		if _, err := tx.Exec(`
			UPDATE assets
			SET subject = ?
			WHERE subject = ?
		`, newName, oldName); err != nil {
			return fmt.Errorf("update assets subject %q -> %q: %w", oldName, newName, err)
		}

		// 4. Delete old subject
		if _, err := tx.Exec(`
			DELETE FROM subjects
			WHERE name = ?
		`, oldName); err != nil {
			return fmt.Errorf("delete old subject %q: %w", oldName, err)
		}
	}

	return tx.Commit()
}

func (s *Store) ListSubjects() ([]*models.Subject, error) {
	rows, err := s.DB.Query(`
		SELECT name
		FROM subjects
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subjects := make([]*models.Subject, 0)
	for rows.Next() {
		var sub models.Subject
		if err := rows.Scan(&sub.Name); err != nil {
			return nil, err
		}
		subjects = append(subjects, &sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subjects, nil
}
