package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA busy_timeout = 5000;`,
		`PRAGMA synchronous = NORMAL;`,
	}

	for _, q := range pragmas {
		if _, err := db.Exec(q); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("pragma failed %q: %w", q, err)
		}
	}

	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func Init(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	newDB := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		newDB = true
	}

	store, err := Open(path)
	if err != nil {
		return nil, err
	}

	if err := store.Migrate(); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	if newDB {
		log.Println("🌱 Seeding database with test data...")

		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			_ = store.Close()
			return nil, fmt.Errorf("missing ADMIN_PASSWORD env var")
		}

		if _, err := store.CreateUser(
			"admin",
			adminPassword,
			"",
			"",
			"admin",
			"",
			"",
			"otero_ediciones@gmail.com",
			"70832284",
		); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("create admin user: %w", err)
		}

		if _, err := store.CreateUser(
			"prof1",
			"password",
			"Alice",
			"Smith",
			"professor",
			"Otero Ediciones",
			"1primaria",
			"example@email.com",
			"0000000",
		); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("create professor user: %w", err)
		}

		if _, err := store.CreateUser(
			"student1",
			"password",
			"Bob",
			"Perez",
			"student",
			"Otero Ediciones",
			"1primaria",
			"example1@email.com",
			"00000",
		); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("create student user: %w", err)
		}

		if err := store.CreateInitialSubjects(); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("create initial subjects: %w", err)
		}

		class, err := store.CreateClass(
			"Literatura",
			"Clase con el profe Hugo",
			"1primaria",
			"literatura",
		)
		if err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("create sample class: %w", err)
		}

		if err := store.AddUserToClass(class.Id, "prof1"); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("add professor to class: %w", err)
		}

		if err := store.AddUserToClass(class.Id, "student1"); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("add student to class: %w", err)
		}

		if _, err := store.CreateAssignment(
			class.Id,
			"Álgebra I",
			"Resolver los ejercicios de la página 42",
			time.Now().AddDate(0, 0, 7).Format("02/01/2006"),
		); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("create sample assignment: %w", err)
		}
	}

	log.Println("✅ Database ready at", path)
	return store, nil
}

func (s *Store) CreateInitialSubjects() error {
	return s.CreateSubjects(SubjectsNames)
}

func (s *Store) CountTable(table string) (int, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count)
	return count, err
}

func (s *Store) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password_hashed TEXT NOT NULL,
			password_not_hashed TEXT NOT NULL,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			school TEXT NOT NULL DEFAULT '',
			grade TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL,
			phone_number TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT ''
		);`,

		`CREATE TABLE IF NOT EXISTS subjects (
			name TEXT PRIMARY KEY
		);`,

		`CREATE TABLE IF NOT EXISTS classes (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			grade TEXT NOT NULL,
			subject TEXT NOT NULL,
			FOREIGN KEY (subject) REFERENCES subjects(name)
		);`,

		`CREATE TABLE IF NOT EXISTS class_users (
			class_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			PRIMARY KEY (class_id, username),
			FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE,
			FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS assignments (
			id INTEGER PRIMARY KEY,
			class_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			due_date TEXT NOT NULL,
			FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS assignment_content (
			assignment_id INTEGER NOT NULL,
			idx INTEGER NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (assignment_id, idx),
			FOREIGN KEY (assignment_id) REFERENCES assignments(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS submissions (
			class_id INTEGER NOT NULL,
			assignment_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			submitted_at TEXT NOT NULL DEFAULT '',
			grade TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (class_id, assignment_id, username),
			FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE,
			FOREIGN KEY (assignment_id) REFERENCES assignments(id) ON DELETE CASCADE,
			FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS submission_content (
			class_id INTEGER NOT NULL,
			assignment_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			idx INTEGER NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (class_id, assignment_id, username, idx),
			FOREIGN KEY (class_id, assignment_id, username)
				REFERENCES submissions(class_id, assignment_id, username)
				ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS assets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			subject TEXT NOT NULL,
			grade TEXT NOT NULL,
			name TEXT NOT NULL,
			original_name TEXT NOT NULL,
			url TEXT NOT NULL,
			student_visibility INTEGER NOT NULL DEFAULT 0,
			professor_visibility INTEGER NOT NULL DEFAULT 0,
			UNIQUE(subject, grade, name)
		);`,

		`CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
		);`,

		`CREATE INDEX IF NOT EXISTS idx_classes_subject_grade
			ON classes(subject, grade);`,

		`CREATE INDEX IF NOT EXISTS idx_assignments_class_id
			ON assignments(class_id);`,

		`CREATE INDEX IF NOT EXISTS idx_submissions_assignment
			ON submissions(class_id, assignment_id);`,

		`CREATE INDEX IF NOT EXISTS idx_assets_subject_grade
			ON assets(subject, grade);`,

		`CREATE INDEX IF NOT EXISTS idx_sessions_username
			ON sessions(username);`,
	}

	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
