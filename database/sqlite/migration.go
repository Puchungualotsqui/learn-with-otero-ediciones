package sqlite

import (
	"frontend/database/models"
)

func (s *Store) InsertMigratedUser(u *models.User) error {
	_, err := s.DB.Exec(`
		INSERT INTO users (
			username, password_hashed, password_not_hashed,
			first_name, last_name, school, grade, role,
			phone_number, email
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		u.Username,
		u.PasswordHashed,
		u.PasswordNotHashed,
		u.FirstName,
		u.LastName,
		u.School,
		u.Grade,
		u.Role,
		u.PhoneNumber,
		u.Email,
	)
	return err
}

func (s *Store) InsertMigratedClass(c *models.Class) error {
	_, err := s.DB.Exec(`
		INSERT INTO classes (id, name, description, grade, subject)
		VALUES (?, ?, ?, ?, ?)
	`, c.Id, c.Name, c.Description, c.Grade, c.Subject)
	return err
}

func (s *Store) InsertMigratedAssignment(classID int, a *models.Assignment) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO assignments (id, class_id, title, description, due_date)
		VALUES (?, ?, ?, ?, ?)
	`, a.Id, classID, a.Title, a.Description, a.DueDate)
	if err != nil {
		return err
	}

	for i, item := range a.Content {
		_, err := tx.Exec(`
			INSERT INTO assignment_content (assignment_id, idx, value)
			VALUES (?, ?, ?)
		`, a.Id, i, item)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) InsertMigratedSubmission(classID, assignmentID int, sub *models.Submission) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO submissions (
			class_id, assignment_id, username, description, submitted_at, grade
		) VALUES (?, ?, ?, ?, ?, ?)
	`, classID, assignmentID, sub.Username, sub.Description, sub.SubmittedAt, sub.Grade)
	if err != nil {
		return err
	}

	for i, item := range sub.Content {
		_, err := tx.Exec(`
			INSERT INTO submission_content (
				class_id, assignment_id, username, idx, value
			) VALUES (?, ?, ?, ?, ?)
		`, classID, assignmentID, sub.Username, i, item)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) InsertMigratedAsset(subject, grade string, a *models.Asset) error {
	_, err := s.DB.Exec(`
		INSERT INTO assets (
			subject, grade, name, original_name, url,
			student_visibility, professor_visibility
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, subject, grade, a.Name, a.OriginalName, a.Url, boolToInt(a.StudentVisibility), boolToInt(a.ProfessorVisibility))
	return err
}

func (s *Store) InsertMigratedSession(sessionID, username string) error {
	_, err := s.DB.Exec(`
		INSERT INTO sessions (session_id, username)
		VALUES (?, ?)
	`, sessionID, username)
	return err
}
