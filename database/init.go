package database

import (
	"frontend/database/models"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.etcd.io/bbolt"
)

var Buckets = map[string][]byte{
	"users":       []byte("Users"),
	"subjects":    []byte("Subjects"),
	"classes":     []byte("Classes"),
	"assignments": []byte("Assignments"),
	"submissions": []byte("Submissions"),
}

// Init opens (or creates) the DB and seeds test data if new
func Init(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	newDB := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		newDB = true
	}

	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	// Create buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range Buckets {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	store := &Store{db: db}

	if newDB {
		log.Println("🌱 Seeding database with test data...")

		// Create sample users
		_ = CreateUser(store, "prof1", "password", "Alice", "Smith", "professor", []byte("my-secret-key-1234567890123456"))
		_ = CreateUser(store, "student1", "password", "Bob", "Perez", "student", []byte("my-secret-key-1234567890123456"))

		// Create a subject
		subject := models.Subject{
			Name: "literatura",
		}
		_ = Save(store, Buckets["subjects"], subject.Name, subject)

		// Create a class
		class := models.Class{
			Id:         1,
			Name:       "Literatura 4to A",
			Subject:    "literatura",
			TeacherIds: []int{1}, // professor ID
			StudentIds: []int{2}, // student ID
		}
		_ = Save(store, Buckets["classes"], strconv.Itoa(class.Id), class)

		// Create an assignment
		a, _ := CreateAssignment(store, class.Id, "Álgebra I", "Resolver los ejercicios de la página 42", time.Now().AddDate(0, 0, 7).Format("2006-01-02"))

		// Create a submission
		_, _ = CreateSubmission(store, a.Id, 2, "solucion.pdf", time.Now().Format(time.RFC3339), "")
	}

	log.Println("✅ Database ready at", path)
	return store, nil
}
