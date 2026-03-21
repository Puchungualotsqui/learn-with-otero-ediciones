package sqlite

import (
	"database/sql"
	"fmt"
	"frontend/database/models"
	"strings"
)

var Grades []string = []string{
	"1primaria", "2primaria", "3primaria", "4primaria", "5primaria", "6primaria",
	"1secundaria", "2secundaria", "3secundaria", "4secundaria", "5secundaria", "6secundaria",
}

func (s *Store) CreateClass(name, description, grade, subject string) (*models.Class, error) {
	res, err := s.DB.Exec(`
		INSERT INTO classes (name, description, grade, subject)
		VALUES (?, ?, ?, ?)
	`, name, description, grade, subject)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	class := &models.Class{
		Id:          int(id),
		Name:        name,
		Description: description,
		Grade:       grade,
		Subject:     subject,
		Users:       []string{}, // empty by default
	}

	return class, nil
}

func (s *Store) GetClass(id int) (*models.Class, error) {
	row := s.DB.QueryRow(`
		SELECT id, name, description, grade, subject
		FROM classes
		WHERE id = ?
	`, id)

	var c models.Class
	if err := row.Scan(&c.Id, &c.Name, &c.Description, &c.Grade, &c.Subject); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("class not found")
		}
		return nil, err
	}

	userRows, err := s.DB.Query(`
		SELECT username
		FROM class_users
		WHERE class_id = ?
		ORDER BY username
	`, id)
	if err != nil {
		return nil, err
	}
	defer userRows.Close()

	for userRows.Next() {
		var username string
		if err := userRows.Scan(&username); err != nil {
			return nil, err
		}
		c.Users = append(c.Users, username)
	}

	return &c, userRows.Err()
}

func (s *Store) AddUserToClass(classID int, username string) error {
	_, err := s.DB.Exec(`
		INSERT OR IGNORE INTO class_users (class_id, username)
		VALUES (?, ?)
	`, classID, username)
	return err
}

func (s *Store) RemoveUserFromClass(classID int, username string) error {
	_, err := s.DB.Exec(`
		DELETE FROM class_users
		WHERE class_id = ? AND username = ?
	`, classID, username)
	return err
}

func (s *Store) DeleteClass(classID int) error {
	_, err := s.DB.Exec(`DELETE FROM classes WHERE id = ?`, classID)
	return err
}

func (s *Store) UpdateClass(id int, name, description, grade, subject string) error {
	_, err := s.DB.Exec(`
		UPDATE classes
		SET name = ?, description = ?, grade = ?, subject = ?
		WHERE id = ?
	`, name, description, grade, subject, id)
	return err
}

func (s *Store) SearchClasses(idQuery, nameQuery, descQuery, grade, subject string) ([]*models.Class, error) {
	query := `
		SELECT id, name, description, grade, subject
		FROM classes
		WHERE 1=1
	`
	args := make([]any, 0)

	if idQuery != "" {
		query += ` AND CAST(id AS TEXT) LIKE ?`
		args = append(args, "%"+idQuery+"%")
	}

	if nameQuery != "" {
		query += ` AND LOWER(name) LIKE ?`
		args = append(args, "%"+strings.ToLower(nameQuery)+"%")
	}

	if descQuery != "" {
		query += ` AND LOWER(description) LIKE ?`
		args = append(args, "%"+strings.ToLower(descQuery)+"%")
	}

	if grade != "" {
		query += ` AND grade = ?`
		args = append(args, grade)
	}

	if subject != "" {
		query += ` AND LOWER(subject) LIKE ?`
		args = append(args, "%"+strings.ToLower(subject)+"%")
	}

	query += ` ORDER BY id DESC`

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query classes: %w", err)
	}
	defer rows.Close()

	results := make([]*models.Class, 0)
	for rows.Next() {
		var c models.Class
		if err := rows.Scan(
			&c.Id,
			&c.Name,
			&c.Description,
			&c.Grade,
			&c.Subject,
		); err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}

		results = append(results, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classes: %w", err)
	}

	return results, nil
}

func (s *Store) GetClassesByUsername(username string) ([]*models.Class, error) {
	rows, err := s.DB.Query(`
		SELECT
			c.id,
			c.name,
			c.description,
			c.grade,
			c.subject
		FROM class_users cu
		JOIN classes c ON c.id = cu.class_id
		WHERE cu.username = ?
		ORDER BY c.id ASC
	`, username)
	if err != nil {
		return nil, fmt.Errorf("query classes by username: %w", err)
	}
	defer rows.Close()

	classes := make([]*models.Class, 0)
	for rows.Next() {
		var c models.Class
		if err := rows.Scan(
			&c.Id,
			&c.Name,
			&c.Description,
			&c.Grade,
			&c.Subject,
		); err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}
		classes = append(classes, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classes by username: %w", err)
	}

	return classes, nil
}
