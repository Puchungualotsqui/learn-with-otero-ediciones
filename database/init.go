package database

import (
	"fmt"
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
	"sessions":    []byte("Sessions"),
	"assets":      []byte("Assets"),
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
		_ = CreateUser(store, "prof1", "password", "Alice", "Smith", "professor")
		if err := CreateUser(store, "student1", "password", "Bob", "Perez", "student"); err != nil {
			fmt.Printf("Error creating User: %v\n", err)
		}

		if err := CreateInitialSubjects(store); err != nil {
			fmt.Printf("Error creating Subjects: %v\n", err)
		}

		class, _ := CreateClass(store, "Literatura", "Clase con el profe Hugo", "literatura", "1primaria")

		AddUserToClass(store, class.Id, "prof1")
		AddUserToClass(store, class.Id, "student1")

		// Create an assignment
		CreateAssignment(store, class.Id, "Álgebra I", "Resolver los ejercicios de la página 42", time.Now().AddDate(0, 0, 7).Format("02/01/2006"))
	}

	log.Println("✅ Database ready at", path)
	return store, nil
}
