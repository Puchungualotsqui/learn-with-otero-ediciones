package dto

import "frontend/database/models"

type Submission struct {
	Id          int
	Username    string
	Description string
	Content     []string
	SubmittedAt string
	Grade       string
}

func SubmissionFromModel(submission *models.Submission) *Submission {
	return &Submission{
		Username:    submission.Username,
		Description: submission.Description,
		Content:     submission.Content,
		SubmittedAt: submission.SubmittedAt,
		Grade:       submission.Grade,
	}
}

func SubmissionFromModels(submissions []*models.Submission) []*Submission {
	result := make([]*Submission, len(submissions))
	for i, submission := range submissions {
		result[i] = SubmissionFromModel(submission)
	}

	return result
}
