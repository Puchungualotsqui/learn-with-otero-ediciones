package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/templates/components/assignment/assignmentDetailProfessor"
	"frontend/templates/components/assignment/assignmentDetailStudent"
	"frontend/templates/components/assignment/assignmentEditor"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		assignmentDetailProfessor.AssignmentDetailProfessor(classIdInt, a, subsDTO).Render(r.Context(), w)
		fmt.Println("  ✔ Render complete")
		return
	}

	fmt.Println("  → Rendering student detail")
	assignmentDetailStudent.AssignmentDetailStudent(a).Render(r.Context(), w)
	fmt.Println("  ✔ Render complete")
}

// HandleAssignmentNew creates a blank assignment for a class and renders the edit form
func HandleAssignmentNew(store *database.Store, w http.ResponseWriter, r *http.Request, classId int) {
	fmt.Println("📥 [HandleAssignmentNew] Request received")

	// Create empty assignment with placeholder values
	newAssignment, err := database.CreateAssignment(
		store,
		classId,
		"Nuevo título",
		"Agrega la descripción aquí...",
		time.Now().AddDate(0, 0, 7).Format("02/01/2006"),
	)
	if err != nil {
		http.Error(w, "Failed to create assignment", http.StatusInternalServerError)
		return
	}

	a := dto.AssignmentFromModel(*newAssignment)
	assignmentEditor.AssignmentEditor(a, classId).Render(r.Context(), w)
}

// HandleAssignmentUpdate updates an assignment based on form data (HTMX-friendly)
func HandleAssignmentUpdate(store *database.Store, w http.ResponseWriter, r *http.Request, classId int) {
	fmt.Println("📥 [HandleAssignmentUpdate] Request received")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing assignment id", http.StatusBadRequest)
		return
	}

	// Parse form values (HTMX sends application/x-www-form-urlencoded)
	title := r.FormValue("title")
	description := r.FormValue("description")
	dueDate := r.FormValue("due_date")

	// Handle multiple "content" inputs (files/URLs)
	content := r.Form["content"]

	// Load existing assignment
	assignmentModel, err := database.GetWithPrefix[models.Assignment](
		store,
		[]byte("Assignments"),
		idStr,
		strconv.Itoa(classId),
	)
	if err != nil || assignmentModel == nil {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	// Update fields
	assignmentModel.Title = title
	assignmentModel.Description = description
	assignmentModel.Content = content
	assignmentModel.DueDate = dueDate

	// Save back
	key := fmt.Sprintf("%d:%d", classId, assignmentModel.Id)
	if err := database.Save(store, database.Buckets["assignments"], key, assignmentModel); err != nil {
		http.Error(w, "Failed to save assignment", http.StatusInternalServerError)
		return
	}

	// Render updated detail back to UI
	a := dto.AssignmentFromModel(*assignmentModel)
	assignmentEditor.AssignmentEditor(a, classId).Render(r.Context(), w)
}
