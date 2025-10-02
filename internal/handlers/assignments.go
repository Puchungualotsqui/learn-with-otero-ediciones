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
	"frontend/templates/components/assignment/assignmentEditor"
	"frontend/templates/components/assignment/assignmentList"
	"frontend/templates/components/assignment/assignmentSlotProfessor"
	"frontend/templates/components/assignment/panelsContent"
	"frontend/templates/components/assignment/submissionDetail"
	"frontend/templates/components/assignment/submissionEditor"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
)

func HandleAssignmentDefault(
	w http.ResponseWriter,
	r *http.Request,
	assignments []*models.Assignment,
	classId int,
	professor bool,
	username string,
) {
	// Convert models to DTO once
	assignmentsDTO := dto.AssignmentFromModels(assignments)

	// Grab the “first assignment” bits once
	var (
		firstDTO   *dto.Assignment
		firstID    int
		firstTitle string
	)
	if len(assignments) > 0 {
		firstDTO = dto.AssignmentFromModel(assignments[0])
		firstID = assignments[0].Id // keep using model ID if that’s your source of truth
		firstTitle = firstDTO.Title // title from DTO is fine
	}

	// Right panel differs by role
	var right templ.Component
	if professor {
		fmt.Println("📌 Routed to AssignmentContentProfessor (assignment management)")
		right = assignmentEditor.AssignmentEditor(firstDTO, classId)
	} else {
		fmt.Println("📌 Routed to AssignmentContent (assignment management)")
		// submissionDto is nil for fresh view (same behavior as before)
		right = submissionEditor.SubmissionEditor(nil, classId, firstID, firstTitle)
	}

	// Render once
	render.RenderWithLayout(
		w, r,
		panelsContent.PanelsContent(
			assignmentList.AssignmentList(
				classId,
				assignmentsDTO,
				professor,
				professor,
				username,
			),
			right,
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

func HandleAssignmentDetail(w http.ResponseWriter, r *http.Request, assignments []*models.Assignment, professor bool) {
	fmt.Println("📥 [HandleAssignmentDetail] Request received")
	var assignment *models.Assignment
	if len(assignments) > 0 {
		assignment = assignments[0]
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if !professor {
		fmt.Println("Not allowed")
		http.Error(w, "Not allowed", http.StatusBadRequest)
		return
	}

	classId, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("Invalid class")
		http.Error(w, "Invalid class", http.StatusBadRequest)
		return
	}

	assignmentEditor.AssignmentEditor(dto.AssignmentFromModel(assignment), classId).Render(r.Context(), w)
	fmt.Println("  ✔ Render complete")
}

// HandleAssignmentNew creates a blank assignment for a class and renders the edit form
func HandleAssignmentNew(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, professor bool) {
	fmt.Println("📥 [HandleAssignmentNew] Request received")

	if !professor {
		fmt.Println("Access denied")
		http.Error(w, "Access denied", http.StatusNotAcceptable)
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create empty assignment with placeholder values
	newAssignment, err := database.CreateAssignment(
		store,
		classId,
		"Nuevo título",
		"Agrega la descripción aquí...",
		time.Now().Format("02/01/2006"),
	)
	if err != nil {
		http.Error(w, "Failed to create assignment", http.StatusInternalServerError)
		return
	}
	fmt.Printf("✅ Created new assignment: %+v\n", newAssignment)

	// 2. Convert to DTO
	a := dto.AssignmentFromModel(newAssignment)

	// 3. Render updated slot list
	fmt.Fprintf(w, `<div hx-swap-oob="beforeend:#assignments-list">`)
	assignmentSlotProfessor.AssignmentSlotProfessor(classId, a, true).Render(r.Context(), w)
	fmt.Fprint(w, `</div>`)

	// 4. Render editor into #assignment-detail
	assignmentEditor.AssignmentEditor(a, classId).Render(r.Context(), w)

	fmt.Println("✔ New assignment created and rendered")
}

// HandleAssignmentUpdate updates an assignment based on form data (HTMX-friendly)
func HandleAssignmentUpdate(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, professor bool) {
	fmt.Println("📥 [HandleAssignmentUpdate] Request received")

	if !professor {
		fmt.Println("Access denied")
		http.Error(w, "Access denied", http.StatusNotAcceptable)
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing assignment id", http.StatusBadRequest)
		return
	}
	fmt.Printf("👉 Assignment ID: %s | Class ID: %d\n", idStr, classId)

	// Need to parse multipart form because of file uploads
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB max memory
		fmt.Printf("❌ Failed to parse multipart form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	fmt.Println("✅ Multipart form parsed successfully")

	// Parse values
	title := r.FormValue("title")
	description := r.FormValue("description")
	dueDateGross := r.FormValue("due_date")

	var dueDate string

	// Try parse as yyyy-mm-dd
	if t, err := time.Parse("2006-01-02", dueDateGross); err == nil {
		dueDate = t.Format("02/01/2006")
	} else if t, err := time.Parse("02/01/2006", dueDateGross); err == nil {
		dueDate = t.Format("02/01/2006")
	} else {
		// fallback → store raw string if cannot parse
		dueDate = dueDateGross
	}

	keep := r.Form["keep[]"]                   // already uploaded files to keep
	uploads := r.MultipartForm.File["uploads"] // newly uploaded files

	fmt.Println("👉 Parsed form values:")
	fmt.Printf("   - Title: %q\n", title)
	fmt.Printf("   - Description: %q\n", description)
	fmt.Printf("   - DueDate: %q\n", dueDate)
	fmt.Printf("   - Keep[]: %+v\n", keep)
	fmt.Printf("   - Uploads count: %d\n", len(uploads))
	for i, f := range uploads {
		fmt.Printf("     [%d] Filename=%q Size=%d Header=%+v\n", i, f.Filename, f.Size, f.Header)
	}

	// 1. Load assignment
	assignmentModel, err := database.GetWithPrefix[models.Assignment](
		store,
		database.Buckets["assignments"],
		idStr,
		strconv.Itoa(classId),
	)
	if err != nil || assignmentModel == nil {
		fmt.Printf("❌ Assignment not found for key classId=%d id=%s: %v\n", classId, idStr, err)
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}
	fmt.Printf("✅ Loaded assignment: %+v\n", assignmentModel)

	// 2. Build new Content
	var newContent []string
	newContent = append(newContent, keep...)
	fmt.Printf("📂 Initial newContent (kept): %+v\n", newContent)

	// 2.A track files to keep
	keepSet := make(map[string]struct{})
	for _, k := range keep {
		keepSet[k] = struct{}{}
	}

	// 2b. Delete files not in keep[]
	for _, oldUrl := range assignmentModel.Content {
		if _, ok := keepSet[oldUrl]; !ok {
			if err := storage.DeleteFile(r.Context(), oldUrl); err != nil {
				fmt.Printf("⚠️ failed to delete old file %s: %v\n", oldUrl, err)
			} else {
				fmt.Printf("🗑 deleted old file %s\n", oldUrl)
			}
		}
	}

	// Upload new files to B2
	for _, f := range uploads {
		fmt.Printf("⬆️ Uploading file: %s\n", f.Filename)
		file, err := f.Open()
		if err != nil {
			fmt.Printf("❌ Failed to open uploaded file %s: %v\n", f.Filename, err)
			http.Error(w, "Failed to open uploaded file", http.StatusInternalServerError)
			return
		}

		safeName := helper.NormalizeFilename(f.Filename)
		key := fmt.Sprintf("assignments/%d/%s", assignmentModel.Id, safeName)

		// delete old version if it exists
		err = storage.DeleteFile(r.Context(), key)
		if err == nil {
			fmt.Printf("🗑 Replaced old version of %s\n", key)
		} else {
			fmt.Printf("ℹ️ No old version to delete for %s (or delete failed: %v)\n", key, err)
		}

		fileURL, err := storage.UploadFile(r.Context(), key, file)
		if err != nil {
			fmt.Printf("❌ Failed to upload file %s: %v\n", f.Filename, err)
			http.Error(w, "Failed to upload file", http.StatusInternalServerError)
			return
		}

		if cerr := file.Close(); cerr != nil {
			fmt.Printf("⚠️ Failed to close file %s: %v\n", f.Filename, cerr)
		}

		fmt.Printf("✅ Uploaded file to %s\n", fileURL)
		newContent = append(newContent, fileURL)
	}

	// 3. Update fields
	assignmentModel.Title = title
	assignmentModel.Description = description
	assignmentModel.DueDate = dueDate
	assignmentModel.Content = newContent
	fmt.Printf("📝 Updated assignment model: %+v\n", assignmentModel)

	// 4. Save back
	key := fmt.Sprintf("%d:%d", classId, assignmentModel.Id)
	if err := database.Save(store, database.Buckets["assignments"], key, assignmentModel); err != nil {
		fmt.Printf("❌ Failed to save assignment: %v\n", err)
		http.Error(w, "Failed to save assignment", http.StatusInternalServerError)
		return
	}
	fmt.Println("✅ Assignment saved successfully")

	// 5. Re-render editor
	a := dto.AssignmentFromModel(assignmentModel)
	fmt.Println("📤 Rendering updated AssignmentEditor...")

	// First: editor in #assignment-detail (normal target)
	assignmentEditor.AssignmentEditor(a, classId).Render(r.Context(), w)

	// Then: slot, but out-of-band (update existing slot in sidebar)
	fmt.Fprintf(w, `<div hx-swap-oob="outerHTML:#assignment-slot-%d">`, a.Id)
	assignmentSlotProfessor.AssignmentSlotProfessor(classId, a, true).Render(r.Context(), w)
	fmt.Fprint(w, `</div>`)

	fmt.Println("✔ Render complete")
}

func HandleAssignmentDelete(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, professor bool) {
	fmt.Println("📥 [HandleAssignmentDelete] Request received")

	if !professor {
		fmt.Println("Access denied")
		http.Error(w, "Access denied", http.StatusNotAcceptable)
	}

	if r.Method != http.MethodDelete {
		fmt.Printf("Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		fmt.Printf("Missing argument")
		http.Error(w, "Missing assignment id", http.StatusBadRequest)
		return
	}

	// 1. Load assignment
	assignmentModel, err := database.GetWithPrefix[models.Assignment](
		store,
		database.Buckets["assignments"],
		idStr,
		strconv.Itoa(classId),
	)
	if err != nil || assignmentModel == nil {
		fmt.Printf("Assignment not found")
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	// 2. Delete attached files from B2
	for _, url := range assignmentModel.Content {
		key := strings.TrimPrefix(url, fmt.Sprintf("%s/file/%s/", storage.BaseUrl, storage.Bucket.Name()))
		if key == url {
			// fallback: if TrimPrefix didn’t match, assume full key is stored
			key = url
		}
		if err := storage.DeleteFile(r.Context(), key); err != nil {
			fmt.Printf("⚠️ Failed to delete file %s: %v\n", url, err)
		} else {
			fmt.Printf("🗑 Deleted file %s\n", url)
		}
	}

	// 3. Delete assignment from DB
	key := fmt.Sprintf("%d:%s", classId, idStr)
	if err := database.Delete(store, database.Buckets["assignments"], key); err != nil {
		http.Error(w, "Failed to delete assignment", http.StatusInternalServerError)
		return
	}
	fmt.Printf("🗑 Assignment %s deleted from DB\n", key)

	// 4. Return response → HTMX removes <li> AND clears editor
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<div hx-swap-oob="innerHTML:#assignment-detail"></div>`)
}
