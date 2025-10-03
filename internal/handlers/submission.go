package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/helper"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/assignment/assignmentDetailProfessor"
	"frontend/templates/components/assignment/assignmentList"
	"frontend/templates/components/assignment/panelsContent"
	"frontend/templates/components/assignment/studentSubmissionSlot"
	"frontend/templates/components/assignment/submissionDetail"
	"frontend/templates/components/assignment/submissionEditor"
	"net/http"
	"strconv"
	"strings"
)

func HandleSubmissionDefault(
	store *database.Store,
	w http.ResponseWriter,
	r *http.Request,
	classId int,
	professor bool,
	username string) {

	if !professor {
		fmt.Println("Acces denied")
		http.Error(w, "Acces denied", http.StatusBadRequest)
		return
	}

	assignments := database.ListAssignmentsOfClass(store, classId)

	var assignment *dto.Assignment = nil
	var submissions []*models.Submission = []*models.Submission{}
	if len(assignments) != 0 {
		var err error
		submissions, err = database.GetSubmissionsByAssignment(store, classId, assignments[0].Id)
		if err != nil {
			fmt.Println("Error getting submissions")
		}
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

func HandleAssignmentSubmissions(store *database.Store, w http.ResponseWriter, r *http.Request, professor bool) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	helper.PrintArray(parts)

	submissions, err := database.ListByPrefix[models.Submission](store, database.Buckets["submissions"], parts[0], parts[2])
	if err != nil {
		fmt.Println("Error fetching submissions: %w", err)
		http.Error(w, "Server database error", http.StatusInternalServerError)
		return
	}

	if professor {
		submissionsDto := dto.SubmissionFromModels(submissions)

		classIdInt, err := strconv.Atoi(parts[0])
		if err != nil {
			fmt.Println("! Invalid class Id:", parts[0])
			http.Error(w, "Invalid class Id", http.StatusBadRequest)
			return
		}

		assignment, err := database.GetWithPrefix[models.Assignment](store, database.Buckets["assignments"], parts[2], parts[0])
		if err != nil {
			fmt.Println("Error fetching assignment: %w", err)
			http.Error(w, "Server database error", http.StatusInternalServerError)
		}
		assignmentDto := dto.AssignmentFromModel(assignment)

		fmt.Println("→ Rendering professor submissions list")
		assignmentDetailProfessor.AssignmentDetailProfessor(classIdInt, assignmentDto, submissionsDto).Render(r.Context(), w)
		fmt.Println("✔ Render complete")
		return
	}
}

func HandleAssignmentSubmission(store *database.Store, w http.ResponseWriter, r *http.Request, username string, professor bool) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	classIdInt, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("! Invalid class Id:", parts[0])
		http.Error(w, "Invalid class Id", http.StatusBadRequest)
		return
	}

	fmt.Println("📥 [HandleAssignmentDetail] Request received")
	fmt.Printf("  > Class: %d | Assignment: %s | Professor: %v\n", classIdInt, parts[0], professor)

	submission, err := database.GetWithPrefix[models.Submission](store, database.Buckets["submissions"], parts[4], parts[0], parts[2])
	if err != nil {
		fmt.Println("Error fetching submission: %w", err)
		http.Error(w, "Server database error", http.StatusInternalServerError)
		return
	}
	fmt.Printf("  ✓ Assignment loaded: %+v\n", submission)

	s := dto.SubmissionFromModel(submission)

	if professor {
		fmt.Println("  → Rendering professor detail")
		submissionDetail.SubmissionDetail(s, parts[0], parts[2]).Render(r.Context(), w)
		fmt.Println("  ✔ Render complete")
		return
	}

	if username == parts[4] {
		fmt.Println("  → Rendering student detail")
		assignment, err := database.GetWithPrefix[models.Assignment](store, database.Buckets["assignments"], parts[2], parts[0])
		if err != nil {
			fmt.Println("Error fetching assignment info: %w", err)
			http.Error(w, "Server database error", http.StatusInternalServerError)
			return
		}

		arguments, err := helper.StringsToInts(parts[0], parts[2])
		if err != nil {
			fmt.Println("Invalid arguments: %w", err)
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}

		submissionEditor.SubmissionEditor(s, arguments[0], arguments[1], assignment.Title).Render(r.Context(), w)
		fmt.Println("  ✔ Render complete")
		return
	}
}

func HandleSubmissionGrade(store *database.Store, w http.ResponseWriter, r *http.Request, classId int, username string, professor bool) {
	if !professor {
		fmt.Println("Not allowed")
		http.Error(w, "Not allowed", http.StatusBadRequest)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	assignmentId, err := strconv.Atoi(parts[2])
	if err != nil {
		fmt.Printf("Invalid assignment Id: %v\n", err)
		http.Error(w, "Invalid assignment Id", http.StatusBadRequest)
		return
	}

	grade := r.FormValue("grade")

	submission, err := database.GradeSubmission(store, classId, assignmentId, username, grade)
	if err != nil {
		fmt.Println("Database error grading: %w", err)
		http.Error(w, "Database error grading", http.StatusBadRequest)
		return
	}

	fmt.Println("→ Rendering Student Submission Slot")
	studentSubmissionSlot.StudentSubmissionSlot(classId, assignmentId, dto.SubmissionFromModel(submission)).Render(r.Context(), w)
	fmt.Println("✔ Render complete")
}
