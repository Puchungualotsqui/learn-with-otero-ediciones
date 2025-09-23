package database

import (
	"encoding/json"
	"fmt"
	"frontend/database/models"

	"go.etcd.io/bbolt"
)

// CreateSubmission stores a submission with unique (assignmentId, studentId).
func CreateSubmission(s *Store, assignmentId, studentId int, content []string, submittedAt, grade string) (*models.Submission, error) {
	var sub *models.Submission

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Buckets["submissions"])
		if err != nil {
			return err
		}

		// Unique key = "assignmentId:studentId"
		key := fmt.Sprintf("%d:%d", assignmentId, studentId)

		// Generate auto id
		id64, _ := b.NextSequence()

		sub = &models.Submission{
			Id:          int(id64),
			StudentId:   studentId,
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
func GetSubmission(s *Store, assignmentId, studentId int) (*models.Submission, error) {
	key := fmt.Sprintf("%d:%d", assignmentId, studentId)
	return Get[models.Submission](s, Buckets["submissions"], key)
}

// ListSubmissionsByAssignment → uses prefix scan
func ListSubmissionsByAssignment(s *Store, assignmentId int) ([]models.Submission, error) {
	prefix := fmt.Sprintf("%d:", assignmentId)
	return ListByPrefix[models.Submission](s, Buckets["submissions"], prefix)
}

// ListSubmissionsByStudent → scans all submissions, filters by StudentId
// (optional: add a second index "studentId:assignmentId" if needed)
func ListSubmissionsByStudent(s *Store, studentId int) ([]models.Submission, error) {
	all, err := ListByPrefix[models.Submission](s, Buckets["submissions"], "") // "" → all keys
	if err != nil {
		return nil, err
	}

	var results []models.Submission
	for _, sub := range all {
		if sub.StudentId == studentId {
			results = append(results, sub)
		}
	}
	return results, nil
}

// GradeSubmission → updates the Grade field
func GradeSubmission(s *Store, assignmentId, studentId int, grade string) error {
	key := fmt.Sprintf("%d:%d", assignmentId, studentId)

	sub, err := Get[models.Submission](s, Buckets["submissions"], key)
	if err != nil {
		return err
	}

	sub.Grade = grade

	return Save(s, Buckets["submissions"], key, sub)
}
