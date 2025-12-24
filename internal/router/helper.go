package router

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"slices"
	"strconv"
)

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

func isClassValid(store *database.Store, username, class string) bool {
	user, err := database.Get[models.User](store, database.Buckets["users"], username)
	if err != nil {
		fmt.Println("Error getting user")
		return false
	}

	classId, err := strconv.Atoi(class)
	if err != nil {
		fmt.Println("Invalid class Id")
		return false
	}

	return slices.Contains(user.Classes, classId)
}
