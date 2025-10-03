package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"

	"go.etcd.io/bbolt"
)

func CreateAssignment(s *Store, classId int, title, description, dueDate string) (*models.Assignment, error) {
	var a *models.Assignment
	var id64 uint64

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["assignments"])
		if err != nil {
			return err
		}

		// Generate a unique ID
		id64, _ = b.NextSequence()
		a = &models.Assignment{
			Id:          int(id64),
			Title:       title,
			Description: description,
			DueDate:     dueDate,
		}

		// Marshal assignment
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}

		// Key format: classId:assignmentId
		key := fmt.Sprintf("%d:%d", classId, a.Id)

		// Save to DB
		return b.Put([]byte(key), data)
	})

	if err != nil {
		fmt.Printf("❌ [CreateAssignment] failed: %v\n", err)
		return nil, err
	}

	fmt.Printf("✅ [CreateAssignment] Stored assignment %+v\n", a)

	class, err := GetWithPrefix[models.Class](s, Buckets["classes"], fmt.Sprintf("%d", classId))
	if err != nil {
		fmt.Printf("X Error getting class: %v\n", err)
		return nil, err
	}

	for _, username := range class.Users {
		user, err := Get[models.User](s, Buckets["users"], username)
		if err != nil {
			fmt.Printf("X Error getting user: %v\n", err)
			err = nil
			continue
		}

		if user.Role != "student" {
			continue
		}

		_, err = CreateSubmission(s, classId, int(id64), username, "", []string{}, "", "")
		if err != nil {
			fmt.Printf("X Error creating user submission: %v\n", err)
			err = nil
			continue
		}
	}

	return a, nil
}

func ListAssignmentsOfClass(store *Store, classID int) []*models.Assignment {
	key := fmt.Sprintf("%d", classID)

	assignments, err := ListByPrefix[models.Assignment](
		store,
		Buckets["assignments"],
		key,
	)

	if err != nil {
		fmt.Printf("Error listing assignments of class")
		return []*models.Assignment{}
	}

	return assignments
}
