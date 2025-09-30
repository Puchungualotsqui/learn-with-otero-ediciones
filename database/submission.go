package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"
	"strconv"
	"strings"

	"go.etcd.io/bbolt"
)

// CreateSubmission stores a submission with unique (classId, assignmentId, username).
func CreateSubmission(
	s *Store,
	classId, assignmentId int,
	username, description string,
	content []string,
	submittedAt, grade string,
) (*models.Submission, error) {
	var sub *models.Submission

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["submissions"])
		if err != nil {
			return err
		}

		// Composite key: schoolId:classId:assignmentId:username
		key := fmt.Sprintf("%d:%d:%s", classId, assignmentId, username)

		// Ensure uniqueness: one submission per assignment/user
		if b.Get([]byte(key)) != nil {
			return fmt.Errorf("submission already exists for class %d, assignment %d and user %s",
				classId, assignmentId, username)
		}

		// Auto ID (unique across all submissions)
		id64, _ := b.NextSequence()

		sub = &models.Submission{
			Id:          int(id64),
			Username:    username,
			Description: description,
			Content:     content,
			SubmittedAt: submittedAt,
			Grade:       grade,
		}

		data, err := json.Marshal(sub)
		if err != nil {
			return err
		}

		return b.Put([]byte(key), data)
	})

	if err != nil {
		return nil, err
	}
	return sub, nil
}

// GetSubmission retrieves submission by assignmentId + studentId
func GetSubmission(s *Store, assignmentId int, username string) (*models.Submission, error) {
	key := fmt.Sprintf("%d:%s", assignmentId, username)
	return Get[models.Submission](s, Buckets["submissions"], key)
}

// ListSubmissionsByStudent → scans all submissions, filters by StudentId
// (optional: add a second index "studentId:assignmentId" if needed)
func ListSubmissionsByStudent(s *Store, username string) ([]models.Submission, error) {
	all, err := ListByPrefix[models.Submission](s, Buckets["submissions"], "") // "" → all keys
	if err != nil {
		return nil, err
	}

	var results []models.Submission
	for _, sub := range all {
		if sub.Username == username {
			results = append(results, sub)
		}
	}
	return results, nil
}

// GradeSubmission → updates the Grade field
func GradeSubmission(s *Store, classId, assignmentId int, username, grade string) error {
	key := fmt.Sprintf("%d:%d:%s", classId, assignmentId, username)

	sub, err := Get[models.Submission](s, Buckets["submissions"], key)
	if err != nil {
		return err
	}

	sub.Grade = grade

	return Save(s, Buckets["submissions"], key, sub)
}

func GetSubmissionByAssignmentAndUser(s *Store, classId, assignmentId int, username string) (*models.Submission, error) {
	keyParts := []string{
		strconv.Itoa(classId),
		strconv.Itoa(assignmentId),
		username,
	}
	key := strings.Join(keyParts, ":")

	return Get[models.Submission](s, []byte("Submissions"), key)
}
