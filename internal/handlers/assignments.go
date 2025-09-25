package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/templates/components/assignment"
	"net/http"
	"strconv"
	"strings"
)

func HandleAssignmentDetail(store *database.Store, w http.ResponseWriter, r *http.Request, professor bool) {
	idStr := r.URL.Query().Get("id")

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	classId := parts[0]

	classIdInt, err := strconv.Atoi(classId)
	if err != nil {
		fmt.Println("  ! Invalid class Id:", classId)
		http.Error(w, "Invalid class Id", http.StatusBadRequest)
		return
	}

	fmt.Println("📥 [HandleAssignmentDetail] Request received")
	fmt.Printf("  > Class: %d | Assignment: %s | Professor: %v\n", classIdInt, idStr, professor)

	assignmentModel, err := database.GetWithPrefix[models.Assignment](store, []byte("Assignments"), classId, idStr)
	if err != nil || assignmentModel == nil {
		fmt.Println("  ! Assignment not found in DB")
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}
	fmt.Printf("  ✓ Assignment loaded: %+v\n", assignmentModel)

	a := dto.AssignmentFromModel(*assignmentModel)

	if professor {
		submissionModels, err := database.ListByPrefix[models.Submission](store, []byte("Submissions"), classId, idStr)
		if err != nil {
			fmt.Println("  ! Error loading submissions:", err)
			submissionModels = []models.Submission{}
		}
		fmt.Printf("  ✓ Submissions loaded: %d\n", len(submissionModels))

		subsDTO := dto.SubmissionFromModels(submissionModels)
		fmt.Println("  → Rendering professor detail")
		assignment.AssignmentDetailProfessor(classIdInt, a, subsDTO).Render(r.Context(), w)
		fmt.Println("  ✔ Render complete")
		return
	}

	fmt.Println("  → Rendering student detail")
	assignment.AssignmentDetail(a).Render(r.Context(), w)
	fmt.Println("  ✔ Render complete")
}
