package sqlite

import (
	"database/sql"
	"fmt"
	"frontend/database/models"
)

func (s *Store) CreateAssignment(classId int, title, description, dueDate string) (*models.Assignment, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO assignments (class_id, title, description, due_date)
		VALUES (?, ?, ?, ?)
	`, classId, title, description, dueDate)
	if err != nil {
		fmt.Printf("❌ [CreateAssignment] failed inserting assignment: %v\n", err)
		return nil, err
	}

	id64, err := res.LastInsertId()
	if err != nil {
		fmt.Printf("❌ [CreateAssignment] failed getting last insert id: %v\n", err)
		return nil, err
	}

	a := &models.Assignment{
		Id:          int(id64),
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		Content:     []string{},
	}

	fmt.Printf("✅ [CreateAssignment] Stored assignment %+v\n", a)

	rows, err := tx.Query(`
		SELECT u.username
		FROM class_users cu
		JOIN users u ON u.username = cu.username
		WHERE cu.class_id = ? AND u.role = 'student'
	`, classId)
	if err != nil {
		fmt.Printf("X Error getting class users: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			fmt.Printf("X Error scanning class user: %v\n", err)
			return nil, err
		}
		usernames = append(usernames, username)
	}
	if err := rows.Err(); err != nil {
		fmt.Printf("X Error iterating class users: %v\n", err)
		return nil, err
	}

	for _, username := range usernames {
		_, err := tx.Exec(`
			INSERT INTO submissions (
				class_id, assignment_id, username, description, submitted_at, grade
			) VALUES (?, ?, ?, ?, ?, ?)
		`, classId, int(id64), username, "", "", "")
		if err != nil {
			fmt.Printf("X Error creating user submission for %s: %v\n", username, err)
			// match old behavior: continue on per-user submission failure
			err = nil
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		fmt.Printf("❌ [CreateAssignment] failed committing: %v\n", err)
		return nil, err
	}

	return a, nil
}

func (s *Store) ListAssignmentsOfClass(classID int) ([]*models.Assignment, error) {
	rows, err := s.DB.Query(`
		SELECT id, title, description, due_date
		FROM assignments
		WHERE class_id = ?
		ORDER BY id ASC
	`, classID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	assignments := make([]*models.Assignment, 0)

	for rows.Next() {
		var a models.Assignment
		if err := rows.Scan(&a.Id, &a.Title, &a.Description, &a.DueDate); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}

		// Load content
		contentRows, err := s.DB.Query(`
			SELECT value
			FROM assignment_content
			WHERE assignment_id = ?
			ORDER BY idx ASC
		`, a.Id)
		if err != nil {
			return nil, fmt.Errorf("query assignment content: %w", err)
		}

		a.Content = []string{}
		for contentRows.Next() {
			var value string
			if err := contentRows.Scan(&value); err != nil {
				contentRows.Close()
				return nil, fmt.Errorf("scan assignment content: %w", err)
			}
			a.Content = append(a.Content, value)
		}
		contentRows.Close()

		assignments = append(assignments, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignments: %w", err)
	}

	return assignments, nil
}

func (s *Store) GetAssignment(classID, assignmentID int) (*models.Assignment, error) {
	row := s.DB.QueryRow(`
		SELECT id, title, description, due_date
		FROM assignments
		WHERE class_id = ? AND id = ?
	`, classID, assignmentID)

	var a models.Assignment
	if err := row.Scan(&a.Id, &a.Title, &a.Description, &a.DueDate); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("assignment not found")
		}
		return nil, fmt.Errorf("get assignment: %w", err)
	}

	contentRows, err := s.DB.Query(`
		SELECT value
		FROM assignment_content
		WHERE assignment_id = ?
		ORDER BY idx
	`, a.Id)
	if err != nil {
		return nil, fmt.Errorf("get assignment content: %w", err)
	}
	defer contentRows.Close()

	for contentRows.Next() {
		var v string
		if err := contentRows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan assignment content: %w", err)
		}
		a.Content = append(a.Content, v)
	}

	if err := contentRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignment content: %w", err)
	}

	return &a, nil
}

func (s *Store) UpdateAssignment(classID int, a *models.Assignment) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE assignments
		SET title = ?, description = ?, due_date = ?
		WHERE class_id = ? AND id = ?
	`, a.Title, a.Description, a.DueDate, classID, a.Id)
	if err != nil {
		return fmt.Errorf("update assignment: %w", err)
	}

	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return fmt.Errorf("assignment not found")
	}

	if _, err := tx.Exec(`
		DELETE FROM assignment_content
		WHERE assignment_id = ?
	`, a.Id); err != nil {
		return fmt.Errorf("clear assignment content: %w", err)
	}

	for i, item := range a.Content {
		if _, err := tx.Exec(`
			INSERT INTO assignment_content (assignment_id, idx, value)
			VALUES (?, ?, ?)
		`, a.Id, i, item); err != nil {
			return fmt.Errorf("insert assignment content: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Store) DeleteAssignment(classID, assignmentID int) error {
	res, err := s.DB.Exec(`
		DELETE FROM assignments
		WHERE class_id = ? AND id = ?
	`, classID, assignmentID)
	if err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}

	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}
