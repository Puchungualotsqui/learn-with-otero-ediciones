package database

import (
	"fmt"
	"frontend/database/models"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

var Buckets = map[string][]byte{
	"users":       []byte("Users"),
	"subjects":    []byte("Subjects"),
	"classes":     []byte("Classes"),
	"assignments": []byte("Assignments"),
	"submissions": []byte("Submissions"),
	"schools":     []byte("Schools"),
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
		_ = CreateUser(store, "prof1", "password", "Alice", "Smith", "professor", []byte("my-secret-key-12"))
		if err := CreateUser(store, "student1", "password", "Bob", "Perez", "student", []byte("my-secret-key-12")); err != nil {
			fmt.Printf("Error creating User: %v\n", err)
		}

		// Create a subject
		subject := models.Subject{
			InternalName: "matematicas",
			Name:         "Matemáticas",
		}
		_ = CreateSubject(store, subject.InternalName, subject.Name)

		class, _ := CreateClass(store, "Matemáticas", "Clase con el profe Hugo", "matematicas")

		AddUserToClass(store, class.Id, "prof1")
		AddUserToClass(store, class.Id, "student1")

		// Create an assignment
		a, _ := CreateAssignment(store, class.Id, "Álgebra I", "Resolver los ejercicios de la página 42", time.Now().AddDate(0, 0, 7).Format("02/01/2006"))

		// Create a submission
		_, _ = CreateSubmission(store, class.Id, a.Id, "student1", "that's my submission content", []string{"solucion.pdf"}, time.Now().Format(time.RFC3339), "")
	}

	log.Println("✅ Database ready at", path)
	return store, nil
}
