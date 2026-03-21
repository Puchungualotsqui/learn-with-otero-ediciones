package commonUsers

import (
	"fmt"
	"frontend/database/models"
	"frontend/database/sqlite"
	"frontend/helper"
	"frontend/internal/render"
	"frontend/storage"
	"frontend/templates/body"
	"frontend/templates/components/assignment/assignmentDetail"
	"frontend/templates/components/assignment/assignmentDetailProfessor"
	"frontend/templates/components/assignment/assignmentList"
	"frontend/templates/components/assignment/studentSubmissionSlot"
	"frontend/templates/components/assignment/submissionDetail"
	"frontend/templates/components/assignment/submissionEditor"
	"frontend/templates/components/panelsContent"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/a-h/templ"
)

func HandleSubmissionDefault(
	store *sqlite.Store,
	w http.ResponseWriter,
	r *http.Request,
	classId int,
	professor bool,
	username string,
) {
	fmt.Println("📥 [HandleSubmissionDefault] Request received")

	if !professor {
		fmt.Println("Acces denied")
		http.Error(w, "Acces denied", http.StatusBadRequest)
		return
	}

	assignments, err := store.ListAssignmentsOfClass(classId)
	if err != nil {
		fmt.Println("Error listing assignments:", err)
		http.Error(w, "Error listing assignments", http.StatusInternalServerError)
		return
	}

	assignments, err = helper.OrderAssignments(assignments)
	if err != nil {
		fmt.Println("Error ordering assignments:", err)
	}

	render.RenderWithLayout(
		w, r,
		panelsContent.PanelsContent(
			assignmentList.AssignmentList(
				classId,
				assignments,
				[]string{},
				professor,
				false,
				username,
			),
			assignmentDetailProfessor.AssignmentDetailProfessor(
				classId,
				nil,
				nil,
				false,
				[]string{},
			),
			submissionDetail.SubmissionDetail(
				nil,
				"",
				"",
				professor,
				true,
			),
		),
		body.Home,
	)
}

func HandleAssignmentSubmissions(store *sqlite.Store, w http.ResponseWriter, r *http.Request, professor bool) {
	fmt.Println("📥 [HandleAssignmentSubmissions] Request received")

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	helper.PrintArray(parts)

	if !professor {
		http.Error(w, "Not allowed", http.StatusBadRequest)
		return
	}

	classIdInt, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("! Invalid class Id:", parts[0])
		http.Error(w, "Invalid class Id", http.StatusBadRequest)
		return
	}

	assignmentId, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid assignment id", http.StatusBadRequest)
		return
	}

	assignment, err := store.GetAssignment(classIdInt, assignmentId)
	if err != nil {
		fmt.Printf("Error fetching assignment: %v\n", err)
		http.Error(w, "Server database error", http.StatusInternalServerError)
		return
	}

	dateStatus, err := helper.GetDateStatus(assignment.DueDate)
	if err != nil {
		fmt.Printf("Error calculating date status of assignment: %v\n", err)
		http.Error(w, "Error calculating date status of assignment", http.StatusInternalServerError)
		return
	}

	var submissions []*models.Submission
	if !dateStatus.Past {
		assignment = nil
	} else {
		submissions, err = store.ListSubmissionsByAssignment(classIdInt, assignmentId)
		if err != nil {
			fmt.Printf("Error fetching submissions: %v\n", err)
			http.Error(w, "Server database error", http.StatusInternalServerError)
			return
		}
	}

	fullNameStudents := make([]string, len(submissions))
	for i, submission := range submissions {
		user, err := store.GetUser(submission.Username)
		if err != nil {
			fmt.Printf("Error retrieving user %s: %v\n", submission.Username, err)
			fullNameStudents[i] = submission.Username
			continue
		}
		fullNameStudents[i] = strings.TrimRightFunc(user.FirstName, unicode.IsSpace) + " " + user.LastName
	}

	fmt.Println("→ Rendering professor submissions list")
	assignmentDetailProfessor.AssignmentDetailProfessor(
		classIdInt,
		assignment,
		submissions,
		dateStatus.Past,
		fullNameStudents,
	).Render(r.Context(), w)
	submissionDetail.SubmissionDetail(nil, "", "", false, false).Render(r.Context(), w)
	fmt.Println("✔ Render complete")
}

func HandleAssignmentSubmission(store *sqlite.Store, w http.ResponseWriter, r *http.Request, username string, professor bool) {
	fmt.Println("📥 [HandleAssignmentSubmission] Request received")

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	classIdInt, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("! Invalid class Id:", parts[0])
		http.Error(w, "Invalid class Id", http.StatusBadRequest)
		return
	}

	assignmentIdInt, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid assignment id", http.StatusBadRequest)
		return
	}

	targetUsername := parts[4]

	fmt.Printf("  > Class: %d | Assignment: %d | Professor: %v\n", classIdInt, assignmentIdInt, professor)

	submission, err := store.GetSubmission(classIdInt, assignmentIdInt, targetUsername)
	if err != nil {
		fmt.Printf("Error fetching submission: %v\n", err)
		http.Error(w, "Server database error", http.StatusInternalServerError)
		return
	}
	fmt.Printf("  ✓ Submission loaded: %+v\n", submission)

	if professor {
		fmt.Println("  → Rendering professor detail")
		submissionDetail.SubmissionDetail(
			submission,
			strconv.Itoa(classIdInt),
			strconv.Itoa(assignmentIdInt),
			professor,
			false,
		).Render(r.Context(), w)
		fmt.Println("  ✔ Render complete")
		return
	}

	if username == targetUsername {
		fmt.Println("  → Rendering student detail")

		assignment, err := store.GetAssignment(classIdInt, assignmentIdInt)
		if err != nil {
			fmt.Printf("Error fetching assignment info: %v\n", err)
			http.Error(w, "Server database error", http.StatusInternalServerError)
			return
		}

		status, err := helper.GetDateStatus(assignment.DueDate)
		if err != nil {
			fmt.Printf("Invalid due date: %v\n", err)
			http.Error(w, "Invalid due date", http.StatusBadRequest)
			return
		}

		var detailWindow templ.Component
		if !status.Past {
			detailWindow = submissionEditor.SubmissionEditor(submission, classIdInt, assignmentIdInt, assignment.Title)
		} else {
			detailWindow = submissionDetail.SubmissionDetail(
				submission,
				strconv.Itoa(classIdInt),
				strconv.Itoa(assignmentIdInt),
				false,
				false,
			)
		}
		assignmentDetailWindow := assignmentDetail.AssignmentDetail(assignment, false)

		detailWindow.Render(r.Context(), w)
		assignmentDetailWindow.Render(r.Context(), w)
		fmt.Println("  ✔ Render complete")
		return
	}
}

func HandleSubmissionGrade(store *sqlite.Store, w http.ResponseWriter, r *http.Request, classId int, username string, professor bool) {
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

	submission, err := store.GradeSubmission(classId, assignmentId, username, grade)
	if err != nil {
		fmt.Printf("Database error grading: %v\n", err)
		http.Error(w, "Database error grading", http.StatusBadRequest)
		return
	}

	user, err := store.GetUser(submission.Username)
	if err != nil {
		fmt.Printf("Error retrieving user: %v\n", err)
		return
	}

	fullName := strings.TrimRightFunc(user.FirstName, unicode.IsSpace) + " " + user.LastName

	fmt.Println("→ Rendering Student Submission Slot")
	studentSubmissionSlot.StudentSubmissionSlot(classId, assignmentId, submission, fullName).Render(r.Context(), w)
	fmt.Println("✔ Render complete")
}

// HandleSubmissionUpdate updates a submission based on form data (HTMX-friendly)
func HandleSubmissionUpdate(store *sqlite.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, assignmentId, username string, professor bool) {
	fmt.Println("📥 [HandleSubmissionUpdate] Request received")

	if professor {
		http.Error(w, "Not allowed", http.StatusBadRequest)
		return
	}

	assignmentIdInt, err := strconv.Atoi(assignmentId)
	if err != nil {
		http.Error(w, "Invalid assignment Id", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if username == "" {
		http.Error(w, "Missing submission username", http.StatusBadRequest)
		return
	}
	fmt.Printf("👉 Submission Username: %s | Class ID: %d | Assignment ID: %d\n", username, classId, assignmentIdInt)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		fmt.Printf("❌ Failed to parse multipart form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	fmt.Println("✅ Multipart form parsed successfully")

	description := r.FormValue("description")
	keep := r.Form["keep[]"]
	uploads := r.MultipartForm.File["uploads"]

	fmt.Println("👉 Parsed form values:")
	fmt.Printf("   - Description: %q\n", description)
	fmt.Printf("   - Keep[]: %+v\n", keep)
	fmt.Printf("   - Uploads count: %d\n", len(uploads))

	submissionModel, err := store.GetSubmission(classId, assignmentIdInt, username)
	if err != nil || submissionModel == nil {
		fmt.Printf("❌ Submission not found for username=%s: %v\n", username, err)
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}
	fmt.Printf("✅ Loaded submission: %+v\n", submissionModel)

	var newContent []string
	newContent = append(newContent, keep...)

	keepSet := make(map[string]struct{})
	for _, k := range keep {
		keepSet[k] = struct{}{}
	}

	for _, oldURL := range submissionModel.Content {
		if _, ok := keepSet[oldURL]; !ok {
			if err := storage.DeleteFile(r.Context(), oldURL); err != nil {
				fmt.Printf("⚠️ failed to delete old file %s: %v\n", oldURL, err)
			} else {
				fmt.Printf("🗑 deleted old file %s\n", oldURL)
			}
		}
	}

	for _, f := range uploads {
		fmt.Printf("⬆️ Uploading file: %s\n", f.Filename)
		file, err := f.Open()
		if err != nil {
			fmt.Printf("❌ Failed to open uploaded file %s: %v\n", f.Filename, err)
			http.Error(w, "Failed to open uploaded file", http.StatusInternalServerError)
			return
		}

		safeName := helper.NormalizeFilename(f.Filename)
		key := fmt.Sprintf("submissions/%d/%d/%s", classId, assignmentIdInt, safeName)

		_ = storage.DeleteFile(r.Context(), key)

		fileURL, err := storage.UploadFile(r.Context(), key, file)
		if err != nil {
			_ = file.Close()
			fmt.Printf("❌ Failed to upload file %s: %v\n", f.Filename, err)
			http.Error(w, "Failed to upload file", http.StatusInternalServerError)
			return
		}
		_ = file.Close()

		fmt.Printf("✅ Uploaded file to %s\n", fileURL)
		newContent = append(newContent, fileURL)
	}

	submissionModel.Description = description
	submissionModel.Content = newContent
	fmt.Printf("📝 Updated submission model: %+v\n", submissionModel)

	if err := store.UpsertSubmission(classId, assignmentIdInt, submissionModel); err != nil {
		fmt.Printf("❌ Failed to save submission: %v\n", err)
		http.Error(w, "Failed to save submission", http.StatusInternalServerError)
		return
	}

	assignmentModel, err := store.GetAssignment(classId, assignmentIdInt)
	if err != nil || assignmentModel == nil {
		fmt.Printf("❌ Failed to load assignment after saving submission: %v\n", err)
		http.Error(w, "Failed to load assignment", http.StatusInternalServerError)
		return
	}

	submissionEditor.SubmissionEditor(submissionModel, classId, assignmentIdInt, assignmentModel.Title).Render(r.Context(), w)
	fmt.Println("✅ Submission saved successfully")
}
