package sqlite

import (
	"database/sql"
	"fmt"
	"frontend/auth"
	"frontend/database/models"
	"frontend/helper"
	"os"
	"strings"
)

func (s *Store) CreateUser(username, plainPassword, firstName, lastName, role, school, grade, email, phoneNumber string) (*models.User, error) {
	encKey := os.Getenv("ENC_KEY")
	if encKey == "" {
		return nil, fmt.Errorf("ENC_KEY not found in environment")
	}

	if strings.TrimSpace(plainPassword) == "" {
		plainPassword = auth.GenerateRandomPassword(10)
	}

	hashed, err := auth.HashPassword(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	encrypted, err := auth.Encrypt([]byte(encKey), plainPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}

	if strings.TrimSpace(username) == "" {
		username, err = s.generateUsername(firstName, lastName, school, grade)
		if err != nil {
			return nil, fmt.Errorf("generate username: %w", err)
		}
	}

	u := &models.User{
		Username:          username,
		PasswordHashed:    hashed,
		PasswordNotHashed: encrypted,
		FirstName:         firstName,
		LastName:          lastName,
		School:            school,
		Grade:             grade,
		PhoneNumber:       phoneNumber,
		Email:             email,
		Role:              role,
		Classes:           []int{},
	}

	_, err = s.DB.Exec(`
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
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	u.PasswordNotHashed = plainPassword
	return u, nil
}

func (s *Store) GetUser(username string) (*models.User, error) {
	row := s.DB.QueryRow(`
		SELECT username, password_hashed, password_not_hashed,
		       first_name, last_name, school, grade, role,
		       phone_number, email
		FROM users
		WHERE username = ?
	`, username)

	var u models.User
	err := row.Scan(
		&u.Username, &u.PasswordHashed, &u.PasswordNotHashed,
		&u.FirstName, &u.LastName, &u.School, &u.Grade, &u.Role,
		&u.PhoneNumber, &u.Email,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	classRows, err := s.DB.Query(`
		SELECT class_id
		FROM class_users
		WHERE username = ?
		ORDER BY class_id
	`, username)
	if err != nil {
		return nil, err
	}
	defer classRows.Close()

	for classRows.Next() {
		var classID int
		if err := classRows.Scan(&classID); err != nil {
			return nil, err
		}
		u.Classes = append(u.Classes, classID)
	}

	return &u, classRows.Err()
}

func (s *Store) DeleteUser(username string) error {
	res, err := s.DB.Exec(`
		DELETE FROM users
		WHERE username = ?
	`, username)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (s *Store) UserExists(username string) (bool, error) {
	var count int
	err := s.DB.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE username = ?
	`, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return count > 0, nil
}

func (s *Store) GetUsersByClassID(classID int) ([]*models.User, error) {
	rows, err := s.DB.Query(`
		SELECT
			u.username,
			u.password_hashed,
			u.password_not_hashed,
			u.first_name,
			u.last_name,
			u.school,
			u.grade,
			u.role,
			u.phone_number,
			u.email
		FROM class_users cu
		JOIN users u ON u.username = cu.username
		WHERE cu.class_id = ?
		ORDER BY u.first_name, u.last_name, u.username
	`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.Username,
			&u.PasswordHashed,
			&u.PasswordNotHashed,
			&u.FirstName,
			&u.LastName,
			&u.School,
			&u.Grade,
			&u.Role,
			&u.PhoneNumber,
			&u.Email,
		); err != nil {
			return nil, err
		}

		users = append(users, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *Store) generateUsername(firstName, lastName, school, grade string) (string, error) {
	clean := func(v string) string {
		parts := strings.Fields(v)
		for i := range parts {
			parts[i] = strings.ToLower(parts[i])
		}
		parts = helper.Remove(parts, "colegio")
		if len(parts) == 0 {
			return ""
		}
		return parts[0]
	}

	f := clean(firstName)
	l := clean(lastName)
	schoolPart := clean(school)

	if f == "admin" {
		return "admin", nil
	}
	if f == "" && l == "" {
		return "user", nil
	}
	if l == "" {
		l = f
	}
	if schoolPart == "" {
		schoolPart = "school"
	}

	base := fmt.Sprintf("%s%s-%s", f, string(l[0]), schoolPart)

	username := base
	counter := 1

	for {
		exists, err := s.UserExists(username)
		if err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}

		username = fmt.Sprintf("%s%s-%s", f, string(l[0]), grade)
		exists, err = s.UserExists(username)
		if err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}

		username = fmt.Sprintf("%s%d", base, counter)
		counter++
	}
}

func (s *Store) UpdateUser(username, firstName, lastName, role, school, grade, phoneNumber, email string) error {
	res, err := s.DB.Exec(`
		UPDATE users
		SET
			first_name = ?,
			last_name = ?,
			role = ?,
			school = ?,
			grade = ?,
			phone_number = ?,
			email = ?
		WHERE username = ?
	`,
		firstName,
		lastName,
		role,
		school,
		grade,
		phoneNumber,
		email,
		username,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (s *Store) SearchUsers(query, role, grade, school, email, phone string, classID *int) ([]*models.User, error) {
	baseQuery := `
		SELECT DISTINCT
			u.username,
			u.password_hashed,
			u.password_not_hashed,
			u.first_name,
			u.last_name,
			u.school,
			u.grade,
			u.role,
			u.phone_number,
			u.email
		FROM users u
	`
	args := make([]any, 0)
	conditions := make([]string, 0)

	if classID != nil {
		baseQuery += ` JOIN class_users cu ON cu.username = u.username `
		conditions = append(conditions, `cu.class_id = ?`)
		args = append(args, *classID)
	}

	if query != "" {
		q := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
		conditions = append(conditions, `
			(
				LOWER(u.username) LIKE ?
				OR LOWER(u.first_name) LIKE ?
				OR LOWER(u.last_name) LIKE ?
				OR LOWER(TRIM(u.first_name || ' ' || u.last_name)) LIKE ?
			)
		`)
		args = append(args, q, q, q, q)
	}

	if school != "" {
		conditions = append(conditions, `LOWER(u.school) LIKE ?`)
		args = append(args, "%"+strings.ToLower(school)+"%")
	}

	if role != "" {
		conditions = append(conditions, `u.role = ?`)
		args = append(args, role)
	}

	if grade != "" {
		conditions = append(conditions, `u.grade = ?`)
		args = append(args, grade)
	}

	if email != "" {
		conditions = append(conditions, `LOWER(u.email) = LOWER(?)`)
		args = append(args, email)
	}

	if phone != "" {
		conditions = append(conditions, `u.phone_number = ?`)
		args = append(args, phone)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += ` ORDER BY u.first_name, u.last_name, u.username `

	rows, err := s.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	results := make([]*models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.Username,
			&u.PasswordHashed,
			&u.PasswordNotHashed,
			&u.FirstName,
			&u.LastName,
			&u.School,
			&u.Grade,
			&u.Role,
			&u.PhoneNumber,
			&u.Email,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.Classes = []int{}
		results = append(results, &u)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	rows.Close()

	for _, u := range results {
		classRows, err := s.DB.Query(`
			SELECT class_id
			FROM class_users
			WHERE username = ?
			ORDER BY class_id
		`, u.Username)
		if err != nil {
			return nil, fmt.Errorf("load user classes: %w", err)
		}

		for classRows.Next() {
			var id int
			if err := classRows.Scan(&id); err != nil {
				classRows.Close()
				return nil, fmt.Errorf("scan user class: %w", err)
			}
			u.Classes = append(u.Classes, id)
		}
		if err := classRows.Err(); err != nil {
			classRows.Close()
			return nil, fmt.Errorf("iterate user classes: %w", err)
		}
		classRows.Close()
	}

	return results, nil
}
