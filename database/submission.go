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

		// Composite key: classId:assignmentId:username
		key := fmt.Sprintf("%d:%d:%s", classId, assignmentId, username)

		sub = &models.Submission{
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
func GetSubmission(s *Store, classId, assignmentId int, username string) (*models.Submission, error) {
	key := fmt.Sprintf("%d:%d:%s", classId, assignmentId, username)
	return Get[models.Submission](s, Buckets["submissions"], key)
}

// GradeSubmission → updates the Grade field
func GradeSubmission(s *Store, classId, assignmentId int, username, grade string) (*models.Submission, error) {
	_, err := strconv.Atoi(grade)
	if err != nil {
		fmt.Println("Invalid grade: %w", err)
		return nil, err
	}

	key := fmt.Sprintf("%d:%d:%s", classId, assignmentId, username)

	sub, err := Get[models.Submission](s, Buckets["submissions"], key)
	if err != nil {
		return nil, err
	}

	sub.Grade = grade

	err = Save(s, Buckets["submissions"], key, sub)

	return sub, err
}

func GetSubmissionsByAssignment(s *Store, classId, assignmentId int) ([]*models.Submission, error) {
	keyParts := []string{
		strconv.Itoa(classId),
		strconv.Itoa(assignmentId),
	}
	key := strings.Join(keyParts, ":")

	return ListByPrefix[models.Submission](s, Buckets["submissions"], key)
}
