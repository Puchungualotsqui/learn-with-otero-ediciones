package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/helper"
	"frontend/internal/render"
	"frontend/storage"
	"frontend/templates/body"
	"frontend/templates/components/assignment/assignmentDetailProfessor"
	"frontend/templates/components/assignment/assignmentList"
	"frontend/templates/components/assignment/assignmentSlotStudent"
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

// HandleSubmissionUpdate updates a submission based on form data (HTMX-friendly)
func HandleSubmissionUpdate(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, assignmentId, username string) {
	fmt.Println("📥 [HandleSubmissionUpdate] Request received")

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

	// Parse form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		fmt.Printf("❌ Failed to parse multipart form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	fmt.Println("✅ Multipart form parsed successfully")

	// Parse values
	description := r.FormValue("description")
	keep := r.Form["keep[]"]
	uploads := r.MultipartForm.File["uploads"]

	fmt.Println("👉 Parsed form values:")
	fmt.Printf("   - Description: %q\n", description)
	fmt.Printf("   - Keep[]: %+v\n", keep)
	fmt.Printf("   - Uploads count: %d\n", len(uploads))

	// 1. Load submission
	submissionModel, err := database.GetWithPrefix[models.Submission](
		store,
		database.Buckets["submissions"],
		username, // primary key
		fmt.Sprintf("%d:%d", classId, assignmentIdInt), // prefix: class+assignment
	)
	if err != nil || submissionModel == nil {
		fmt.Printf("❌ Submission not found for username=%s: %v\n", username, err)
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}
	fmt.Printf("✅ Loaded submission: %+v\n", submissionModel)

	// 2. Build new Content
	var newContent []string
	newContent = append(newContent, keep...)

	keepSet := make(map[string]struct{})
	for _, k := range keep {
		keepSet[k] = struct{}{}
	}

	// Delete old files not in keep[]
	for _, oldUrl := range submissionModel.Content {
		if _, ok := keepSet[oldUrl]; !ok {
			if err := storage.DeleteFile(r.Context(), oldUrl); err != nil {
				fmt.Printf("⚠️ failed to delete old file %s: %v\n", oldUrl, err)
			} else {
				fmt.Printf("🗑 deleted old file %s\n", oldUrl)
			}
		}
	}

	// Upload new files
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

		// delete old version if it exists
		_ = storage.DeleteFile(r.Context(), key)

		fileURL, err := storage.UploadFile(r.Context(), key, file)
		if err != nil {
			fmt.Printf("❌ Failed to upload file %s: %v\n", f.Filename, err)
			http.Error(w, "Failed to upload file", http.StatusInternalServerError)
			return
		}
		_ = file.Close()

		fmt.Printf("✅ Uploaded file to %s\n", fileURL)
		newContent = append(newContent, fileURL)
	}

	// 3. Update fields
	submissionModel.Description = description
	submissionModel.Content = newContent
	fmt.Printf("📝 Updated submission model: %+v\n", submissionModel)

	// 4. Save back
	key := fmt.Sprintf("%d:%d:%s", classId, assignmentIdInt, username)
	if err := database.Save(store, database.Buckets["submissions"], key, submissionModel); err != nil {
		fmt.Printf("❌ Failed to save submission: %v\n", err)
		http.Error(w, "Failed to save submission", http.StatusInternalServerError)
		return
	}
	fmt.Println("✅ Submission saved successfully")

	fmt.Println("→ Rendering Student Assignment Slot")
	assignmentInfo, err := database.GetWithPrefix[models.Assignment](store, database.Buckets["assignments"], assignmentId, fmt.Sprintf("%d", classId))
	assignmentSlotStudent.AssignmentSlotStudent(classId, dto.AssignmentFromModel(assignmentInfo), username).Render(r.Context(), w)
	fmt.Println("✔ Render complete")
}
