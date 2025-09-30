package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/templates/components/assignment/studentSubmissionSlot"
	"frontend/templates/components/assignment/submissionDetail"
	"net/http"
	"strconv"
)

func HandleSubmissionDetail(store *database.Store, w http.ResponseWriter, r *http.Request, classId, assignmentId int) {
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

	s := dto.SubmissionFromModel(*submissionModel)
	fmt.Println("  ✓ Submission loaded")
	submissionDetail.SubmissionDetail(s, strconv.Itoa(classId), strconv.Itoa(assignmentId)).Render(r.Context(), w)
}

func HandleSubmissionGrade(store *database.Store, w http.ResponseWriter, r *http.Request, classId int, assignmentId int, username string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	gradeStr := r.FormValue("grade")
	grade, err := strconv.Atoi(gradeStr)
	if err != nil {
		http.Error(w, "Invalid grade", http.StatusBadRequest)
		return
	}
	if grade > 100 || grade < 0 {
		http.Error(w, "Grade out of bounds", http.StatusBadRequest)
		return
	}

	// Update in DB
	err = database.GradeSubmission(store, classId, assignmentId, username, gradeStr)
	if err != nil {
		http.Error(w, "Error saving grade", http.StatusInternalServerError)
		return
	}

	// Get the fresh submission to pass to template
	sub, err := database.GetSubmissionByAssignmentAndUser(store, classId, assignmentId, username)
	if err != nil || sub == nil {
		http.Error(w, "Submission not found after update", http.StatusInternalServerError)
		return
	}

	// Convert model → dto
	subDto := dto.SubmissionFromModel(*sub)

	// Write OOB wrapper manually, render slot inside
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	studentSubmissionSlot.StudentSubmissionSlot(classId, assignmentId, subDto).Render(r.Context(), w)
}
