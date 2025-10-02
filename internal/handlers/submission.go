package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/assignment/assignmentDetailProfessor"
	"frontend/templates/components/assignment/assignmentList"
	"frontend/templates/components/assignment/panelsContent"
	"frontend/templates/components/assignment/submissionDetail"
	"net/http"
	"strconv"
)

func HandleSubmissionDefault(
	store *database.Store,
	w http.ResponseWriter,
	r *http.Request,
	classId, assignmentId int,
	assignments []*models.Assignment,
	professor bool,
	username string) {
	if !professor {
		fmt.Println("Acces denied")
		http.Error(w, "Acces denied", http.StatusBadRequest)
		return
	}
	submissions, err := database.GetSubmissionsByAssignment(store, classId, assignmentId)
	if err != nil {
		fmt.Println("Error getting submissions")
	}

	var assignment *dto.Assignment = nil
	if len(assignments) != 0 {
		assignment = dto.AssignmentFromModel(assignments[0])
	}

	render.RenderWithLayout(
		w, r,
		panelsContent.PanelsContent(
			assignmentList.AssignmentList(
				classId,
				dto.AssignmentFromModels(assignments),
				professor,
				false,
				username,
			),
			assignmentDetailProfessor.AssignmentDetailProfessor(
				classId,
				assignment,
				dto.SubmissionFromModels(submissions),
			),
			submissionDetail.SubmissionDetail(
				nil,
				"",
				""),
		),
		body.Home,
	)
}

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
