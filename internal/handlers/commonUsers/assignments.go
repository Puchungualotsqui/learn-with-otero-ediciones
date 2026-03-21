package commonUsers

import (
	"context"
	"fmt"
	"frontend/database/models"
	"frontend/database/sqlite"
	"frontend/helper"
	"frontend/internal/render"
	"frontend/storage"
	"frontend/templates/body"
	"frontend/templates/components/assignment/assignmentDetail"
	"frontend/templates/components/assignment/assignmentEditor"
	"frontend/templates/components/assignment/assignmentList"
	"frontend/templates/components/assignment/assignmentSlotProfessor"
	"frontend/templates/components/assignment/submissionEditor"
	"frontend/templates/components/panelsContent"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
)

func HandleAssignmentDefault(
	store *sqlite.Store,
	w http.ResponseWriter,
	r *http.Request,
	classId int,
	professor bool,
	username string,
) {
	fmt.Println("📥 [HandleAssignmentDefault] Request received")

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

	var panels []templ.Component
	grades := []string{}

	if professor {
		panels = make([]templ.Component, 2)
		panels[0] = assignmentList.AssignmentList(classId, assignments, grades, professor, professor, username)
		panels[1] = assignmentEditor.AssignmentEditor(nil, classId, true)
	} else {
		panels = make([]templ.Component, 3)
		grades = make([]string, len(assignments))

		for i, assignment := range assignments {
			tempSubmission, err := store.GetSubmission(classId, assignment.Id, username)
			if err != nil {
				fmt.Println("Error getting grade:", err)
				grades[i] = ""
			} else {
				grades[i] = tempSubmission.Grade
			}
			fmt.Println("Grade:", grades[i])
		}

		panels[0] = assignmentList.AssignmentList(classId, assignments, grades, professor, professor, username)
		panels[1] = assignmentDetail.AssignmentDetail(nil, true)
		panels[2] = submissionEditor.SubmissionEditor(nil, classId, 0, "")
	}

	render.RenderWithLayout(
		w, r,
		panelsContent.PanelsContent(panels...),
		body.Home,
	)
}

func HandleAssignmentDetail(store *sqlite.Store, w http.ResponseWriter, r *http.Request, classId int, professor bool) {
	fmt.Println("📥 [HandleAssignmentDetail] Request received")

	if !professor {
		fmt.Println("Not allowed")
		http.Error(w, "Not allowed", http.StatusBadRequest)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	classId, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("Invalid class")
		http.Error(w, "Invalid class", http.StatusBadRequest)
		return
	}

	var assignmentModel *models.Assignment

	if len(parts) >= 3 {
		assignmentId, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "Invalid assignment id", http.StatusBadRequest)
			return
		}

		assignmentModel, err = store.GetAssignment(classId, assignmentId)
		if err != nil || assignmentModel == nil {
			fmt.Printf("❌ Assignment not found for id=%d: %v\n", assignmentId, err)
			http.Error(w, "Assignment not found", http.StatusNotFound)
			return
		}
	} else {
		assignments, err := store.ListAssignmentsOfClass(classId)
		if err != nil {
			http.Error(w, "Error listing assignments", http.StatusInternalServerError)
			return
		}

		if len(assignments) > 0 {
			assignmentModel = assignments[0]
		} else {
			assignmentEditor.AssignmentEditor(nil, classId, true).Render(r.Context(), w)
			fmt.Println("✔ No assignments, rendered empty editor")
			return
		}
	}

	assignmentEditor.AssignmentEditor(assignmentModel, classId, true).Render(r.Context(), w)
	fmt.Println("✔ Render complete")
}

// HandleAssignmentNew creates a blank assignment for a class and renders the edit form
func HandleAssignmentNew(store *sqlite.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, professor bool) {
	fmt.Println("📥 [HandleAssignmentNew] Request received")

	if !professor {
		fmt.Println("Access denied")
		http.Error(w, "Access denied", http.StatusNotAcceptable)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	newAssignment, err := store.CreateAssignment(
		classId,
		"Nuevo título",
		"",
		time.Now().Format("02/01/2006"),
	)
	if err != nil {
		http.Error(w, "Failed to create assignment", http.StatusInternalServerError)
		return
	}
	fmt.Printf("✅ Created new assignment: %+v\n", newAssignment)

	fmt.Fprintf(w, `<div hx-swap-oob="beforeend:#assignments-list">`)
	assignmentSlotProfessor.AssignmentSlotProfessor(classId, newAssignment, true, "").Render(r.Context(), w)
	fmt.Fprint(w, `</div>`)

	assignmentEditor.AssignmentEditor(newAssignment, classId, true).Render(r.Context(), w)

	fmt.Println("✔ New assignment created and rendered")
}

// HandleAssignmentUpdate updates an assignment based on form data (HTMX-friendly)
func HandleAssignmentUpdate(store *sqlite.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, assignmentId string, professor bool) {
	fmt.Println("📥 [HandleAssignmentUpdate] Request received")

	if !professor {
		fmt.Println("Access denied")
		http.Error(w, "Access denied", http.StatusNotAcceptable)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignmentIdInt, err := strconv.Atoi(assignmentId)
	if err != nil {
		http.Error(w, "Invalid assignment id", http.StatusBadRequest)
		return
	}

	fmt.Printf("👉 Assignment ID: %s | Class ID: %d\n", assignmentId, classId)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		fmt.Printf("❌ Failed to parse multipart form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	fmt.Println("✅ Multipart form parsed successfully")

	title := r.FormValue("title")
	description := r.FormValue("description")
	dueDateGross := r.FormValue("due_date")

	var dueDate string
	if t, err := time.Parse("2006-01-02", dueDateGross); err == nil {
		dueDate = t.Format("02/01/2006")
	} else if t, err := time.Parse("02/01/2006", dueDateGross); err == nil {
		dueDate = t.Format("02/01/2006")
	} else {
		dueDate = dueDateGross
	}

	keep := r.Form["keep[]"]
	uploads := r.MultipartForm.File["uploads"]

	fmt.Println("👉 Parsed form values:")
	fmt.Printf("   - Title: %q\n", title)
	fmt.Printf("   - Description: %q\n", description)
	fmt.Printf("   - DueDate: %q\n", dueDate)
	fmt.Printf("   - Keep[]: %+v\n", keep)
	fmt.Printf("   - Uploads count: %d\n", len(uploads))

	assignmentModel, err := store.GetAssignment(classId, assignmentIdInt)
	if err != nil || assignmentModel == nil {
		fmt.Printf("❌ Assignment not found for key classId=%d id=%s: %v\n", classId, assignmentId, err)
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}
	fmt.Printf("✅ Loaded assignment: %+v\n", assignmentModel)

	var newContent []string
	newContent = append(newContent, keep...)
	fmt.Printf("📂 Initial newContent (kept): %+v\n", newContent)

	keepSet := make(map[string]struct{})
	for _, k := range keep {
		keepSet[k] = struct{}{}
	}

	for _, oldURL := range assignmentModel.Content {
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
		key := fmt.Sprintf("assignments/%d/%s", assignmentModel.Id, safeName)

		err = storage.DeleteFile(r.Context(), key)
		if err == nil {
			fmt.Printf("🗑 Replaced old version of %s\n", key)
		} else {
			fmt.Printf("ℹ️ No old version to delete for %s (or delete failed: %v)\n", key, err)
		}

		fileURL, err := storage.UploadFile(context.Background(), key, file)
		if err != nil {
			_ = file.Close()
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

	assignmentModel.Title = title
	assignmentModel.Description = description
	assignmentModel.DueDate = dueDate
	assignmentModel.Content = newContent
	fmt.Printf("📝 Updated assignment model: %+v\n", assignmentModel)

	if err := store.UpdateAssignment(classId, assignmentModel); err != nil {
		fmt.Printf("❌ Failed to save assignment: %v\n", err)
		http.Error(w, "Failed to save assignment", http.StatusInternalServerError)
		return
	}
	fmt.Println("✅ Assignment saved successfully")

	fmt.Println("📤 Rendering updated slot")
	assignmentSlotProfessor.AssignmentSlotProfessor(classId, assignmentModel, true, "").Render(r.Context(), w)
	assignmentEditor.AssignmentEditor(assignmentModel, classId, false).Render(r.Context(), w)
	fmt.Println("✔ Render complete")
}

func HandleAssignmentDelete(store *sqlite.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request, classId int, professor bool) {
	fmt.Println("📥 [HandleAssignmentDelete] Request received")

	if !professor {
		fmt.Println("Access denied")
		http.Error(w, "Access denied", http.StatusNotAcceptable)
		return
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

	assignmentId, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid assignment id", http.StatusBadRequest)
		return
	}

	assignmentModel, err := store.GetAssignment(classId, assignmentId)
	if err != nil || assignmentModel == nil {
		fmt.Printf("Assignment not found")
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	for _, url := range assignmentModel.Content {
		key := strings.TrimPrefix(url, fmt.Sprintf("%s/file/%s/", storage.BaseUrl, storage.PublicBucket.Name()))
		if key == url {
			key = url
		}
		if err := storage.DeleteFile(r.Context(), key); err != nil {
			fmt.Printf("⚠️ Failed to delete file %s: %v\n", url, err)
		} else {
			fmt.Printf("🗑 Deleted file %s\n", url)
		}
	}

	if err := store.DeleteAssignment(classId, assignmentId); err != nil {
		http.Error(w, "Failed to delete assignment", http.StatusInternalServerError)
		return
	}
	fmt.Printf("🗑 Assignment %d:%d deleted from DB\n", classId, assignmentId)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<div hx-swap-oob="innerHTML:#assignment-detail"></div>`)
}
