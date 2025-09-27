package router

import (
	"frontend/database"
	"frontend/database/models"
	"strconv"
)

func isClassValid(store *database.Store, username, classId string) bool {
	classes, err := database.ListClassesForUser(store, username)
	if err != nil {
		return false
	}
	classIds, err := strconv.Atoi(classId)
	if err != nil {
		return false
	}

	for _, class := range classes {
		if class.Id == classIds {
			return true
		}
	}
	return false
}

func isProfessor(store *database.Store, username string) (bool, error) {
	user, err := database.Get[models.User](store, []byte("Users"), username)
	if err != nil {
		return false, err
	}
	if user.Role == "professor" {
		return true, nil
	}
	return false, nil
}
