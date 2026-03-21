package main

import (
	"fmt"
	"log"
	"os"

	olddb "frontend/database"
	"frontend/database/sqlite"
)

func main() {
	if _, err := os.Stat("data/school.db"); os.IsNotExist(err) {
		log.Fatal("❌ old school.db not found")
	}

	if _, err := os.Stat("data/school.sqlite"); err == nil {
		log.Fatal("⚠️ SQLite already exists — aborting migration")
	}

	oldStore, err := olddb.OpenExisting("data/school.db")
	if err != nil {
		log.Fatalf("open old bbolt: %v", err)
	}
	defer oldStore.Close()

	newStore, err := sqlite.Init("data/school.sqlite")
	if err != nil {
		log.Fatalf("open new sqlite: %v", err)
	}
	defer newStore.Close()

	if err := migrateSubjects(oldStore, newStore); err != nil {
		log.Fatalf("migrate subjects: %v", err)
	}
	if err := migrateUsers(oldStore, newStore); err != nil {
		log.Fatalf("migrate users: %v", err)
	}
	if err := migrateClasses(oldStore, newStore); err != nil {
		log.Fatalf("migrate classes: %v", err)
	}
	if err := migrateClassUsers(oldStore, newStore); err != nil {
		log.Fatalf("migrate class_users: %v", err)
	}
	if err := migrateAssignments(oldStore, newStore); err != nil {
		log.Fatalf("migrate assignments: %v", err)
	}
	if err := migrateSubmissions(oldStore, newStore); err != nil {
		log.Fatalf("migrate submissions: %v", err)
	}
	if err := migrateAssets(oldStore, newStore); err != nil {
		log.Fatalf("migrate assets: %v", err)
	}
	if err := migrateSessions(oldStore, newStore); err != nil {
		log.Fatalf("migrate sessions: %v", err)
	}

	if err := verifyMigration(oldStore, newStore); err != nil {
		log.Fatalf("error in verification: %v", err)
	}
	fmt.Println("migration complete")
}
