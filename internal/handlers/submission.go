package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/templates/components/assignment/submissionDetail"
	"net/http"
	"strconv"
)

func HandleSubmissionDetail(store *database.Store, w http.ResponseWriter, r *http.Request, classId, assignmentId int, professor bool) {
	idStr := r.URL.Query().Get("id")

	fmt.Println("📥 [HandleSubmissionDetail] Request received")
	fmt.Printf("  > Class: %d | Assignment: %d | Submission: %s\n", classId, assignmentId, idStr)

	submissionModel, err := database.GetWithPrefix[models.Submission](
		store,
		[]byte("Submissions"),
		idStr, // username
		strconv.Itoa(classId),
		strconv.Itoa(assignmentId),
	)

	if err != nil || submissionModel == nil {
		fmt.Println("Submission not found")
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	if professor {
		s := dto.SubmissionFromModel(submissionModel)
		fmt.Println("  ✓ Submission loaded")
		submissionDetail.SubmissionDetail(s, strconv.Itoa(classId), strconv.Itoa(assignmentId)).Render(r.Context(), w)
	}

}
