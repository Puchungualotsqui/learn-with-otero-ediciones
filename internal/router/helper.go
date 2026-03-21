package router

import (
	"fmt"
	"frontend/database/sqlite"
	"slices"
	"strconv"
)

func isProfessor(store *sqlite.Store, username string) (bool, error) {
	user, err := store.GetUser(username)
	if err != nil {
		return false, err
	}
	return user.Role == "professor", nil
}

func isClassValid(store *sqlite.Store, username, class string) bool {
	user, err := store.GetUser(username)
	if err != nil {
		fmt.Println("Error getting user:", err)
		return false
	}

	classId, err := strconv.Atoi(class)
	if err != nil {
		fmt.Println("Invalid class Id:", err)
		return false
	}

	return slices.Contains(user.Classes, classId)
}
