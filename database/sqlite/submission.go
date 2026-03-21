package sqlite

import (
	"database/sql"
	"fmt"
	"frontend/database/models"
	"strconv"
)

func (s *Store) GetSubmission(classID, assignmentID int, username string) (*models.Submission, error) {
	row := s.DB.QueryRow(`
		SELECT username, description, submitted_at, grade
		FROM submissions
		WHERE class_id = ? AND assignment_id = ? AND username = ?
	`, classID, assignmentID, username)

	var sub models.Submission
	if err := row.Scan(&sub.Username, &sub.Description, &sub.SubmittedAt, &sub.Grade); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, err
	}

	rows, err := s.DB.Query(`
		SELECT value
		FROM submission_content
		WHERE class_id = ? AND assignment_id = ? AND username = ?
		ORDER BY idx
	`, classID, assignmentID, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		sub.Content = append(sub.Content, v)
	}

	return &sub, rows.Err()
}

func (s *Store) UpsertSubmission(classID, assignmentID int, sub *models.Submission) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO submissions (
			class_id, assignment_id, username, description, submitted_at, grade
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(class_id, assignment_id, username) DO UPDATE SET
			description = excluded.description,
			submitted_at = excluded.submitted_at,
			grade = excluded.grade
	`, classID, assignmentID, sub.Username, sub.Description, sub.SubmittedAt, sub.Grade)
	if err != nil {
		return fmt.Errorf("upsert submission: %w", err)
	}

	if _, err := tx.Exec(`
		DELETE FROM submission_content
		WHERE class_id = ? AND assignment_id = ? AND username = ?
	`, classID, assignmentID, sub.Username); err != nil {
		return fmt.Errorf("clear submission content: %w", err)
	}

	for i, item := range sub.Content {
		if _, err := tx.Exec(`
			INSERT INTO submission_content (
				class_id, assignment_id, username, idx, value
			) VALUES (?, ?, ?, ?, ?)
		`, classID, assignmentID, sub.Username, i, item); err != nil {
			return fmt.Errorf("insert submission content: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Store) GradeSubmission(classID, assignmentID int, username, grade string) (*models.Submission, error) {
	if _, err := strconv.Atoi(grade); err != nil {
		return nil, fmt.Errorf("invalid grade: %w", err)
	}

	res, err := s.DB.Exec(`
		UPDATE submissions
		SET grade = ?
		WHERE class_id = ? AND assignment_id = ? AND username = ?
	`, grade, classID, assignmentID, username)
	if err != nil {
		return nil, fmt.Errorf("grade submission: %w", err)
	}

	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return nil, fmt.Errorf("submission not found")
	}

	return s.GetSubmission(classID, assignmentID, username)
}

func (s *Store) ListSubmissionsByAssignment(classID, assignmentID int) ([]*models.Submission, error) {
	rows, err := s.DB.Query(`
		SELECT username, description, submitted_at, grade
		FROM submissions
		WHERE class_id = ? AND assignment_id = ?
		ORDER BY username ASC
	`, classID, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("list submissions by assignment: %w", err)
	}

	submissions := make([]*models.Submission, 0)
	for rows.Next() {
		var sub models.Submission
		if err := rows.Scan(
			&sub.Username,
			&sub.Description,
			&sub.SubmittedAt,
			&sub.Grade,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan submission: %w", err)
		}
		sub.Content = []string{}
		submissions = append(submissions, &sub)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate submissions: %w", err)
	}
	rows.Close()

	for _, sub := range submissions {
		contentRows, err := s.DB.Query(`
			SELECT value
			FROM submission_content
			WHERE class_id = ? AND assignment_id = ? AND username = ?
			ORDER BY idx ASC
		`, classID, assignmentID, sub.Username)
		if err != nil {
			return nil, fmt.Errorf("query submission content: %w", err)
		}

		for contentRows.Next() {
			var value string
			if err := contentRows.Scan(&value); err != nil {
				contentRows.Close()
				return nil, fmt.Errorf("scan submission content: %w", err)
			}
			sub.Content = append(sub.Content, value)
		}
		if err := contentRows.Err(); err != nil {
			contentRows.Close()
			return nil, fmt.Errorf("iterate submission content: %w", err)
		}
		contentRows.Close()
	}

	return submissions, nil
}
