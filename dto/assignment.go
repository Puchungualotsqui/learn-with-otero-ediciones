package dto

import "frontend/database/models"

type Assignment struct {
	Id          int
	Title       string
	Description string
	Content     []string
	DueDate     string // Formatted as "2025-09-30"
}

func AssignmentFromModel(a models.Assignment) Assignment {
	return Assignment{
		Id:          a.Id,
		Title:       a.Title,
		Description: a.Description,
		Content:     a.Content,
		DueDate:     a.DueDate,
	}
}

func AssignmentFromModels(list []models.Assignment) []Assignment {
	result := make([]Assignment, len(list))
	for i, a := range list {
		result[i] = AssignmentFromModel(a)
	}
	return result
}
