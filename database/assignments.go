package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"

	"go.etcd.io/bbolt"
)

func CreateAssignment(s *Store, classId int, title, description, dueDate string) (*models.Assignment, error) {
	var a *models.Assignment

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["assignments"])
		if err != nil {
			return err
		}

		// Generate a unique ID
		id64, _ := b.NextSequence()
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

		// 🔎 Debug logs
		fmt.Printf("🆕 [CreateAssignment] classId=%d assignmentId=%d\n", classId, a.Id)
		fmt.Printf("🆕 [CreateAssignment] key=%q\n", key)
		fmt.Printf("🆕 [CreateAssignment] value=%s\n", string(data))

		// Save to DB
		return b.Put([]byte(key), data)
	})

	if err != nil {
		fmt.Printf("❌ [CreateAssignment] failed: %v\n", err)
		return nil, err
	}

	fmt.Printf("✅ [CreateAssignment] Stored assignment %+v\n", a)
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
		return []*models.Assignment{}
	}

	return assignments
}
