package dto

import (
	"frontend/database/models"
)

type Class struct {
	Id          int
	Name        string
	Description string // e.g. "Primaria", "Secundaria"
}

type ClassSlot struct {
	Id       int
	Title    string
	SubTitle string
}

func ClassFromModel(class models.Class) Class {
	return Class{
		Id:          class.Id,
		Name:        class.Name,
		Description: class.Description,
	}
}

func ClassFromModels(classes []models.Class) []Class {
	result := make([]Class, len(classes))
	for i, class := range classes {
		result[i] = ClassFromModel(class)
	}
	return result
}

func ClassSlotFromModel(class models.Class) ClassSlot {
	return ClassSlot{
		Id:       class.Id,
		Title:    class.Name,
		SubTitle: class.Description,
	}
}

func ClassSlotFromModels(classes []models.Class) []ClassSlot {
	restult := make([]ClassSlot, len(classes))
	for i, class := range classes {
		restult[i] = ClassSlotFromModel(class)
	}

	return restult
}
