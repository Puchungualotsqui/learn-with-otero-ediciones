package admin

import (
	"encoding/json"
	"fmt"
	"frontend/database/sqlite"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminMessage"
	"frontend/templates/components/admin/adminSubjectManager"
	"frontend/templates/components/admin/adminSubjectRename"
	"net/http"
	"strings"
)

func HandleAdminSubjectsManagerDefault(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminSubjectsDefault] Request received")

	subjects, err := store.ListSubjects()
	if err != nil {
		http.Error(w, "Error fetching subjects", http.StatusInternalServerError)
		return
	}

	names := make([]string, len(subjects))
	for i, s := range subjects {
		names[i] = s.Name
	}

	render.RenderWithLayout(w, r, adminSubjectManager.AdminSubjectManager(names, true), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminSubjectManagerUpdate(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminSubjectUpdate] Request received")

	raw := r.FormValue("subject_data")
	if raw == "" {
		http.Error(w, "No subject data", http.StatusBadRequest)
		return
	}

	var payload struct {
		Add []string `json:"add"`
		Del []string `json:"del"`
	}

	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	addNames := make([]string, 0, len(payload.Add))
	for _, name := range payload.Add {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		addNames = append(addNames, name)
	}

	delNames := make([]string, 0, len(payload.Del))
	for _, name := range payload.Del {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		delNames = append(delNames, name)
	}

	if len(addNames) > 0 {
		if err := store.CreateSubjects(addNames); err != nil {
			http.Error(w, "Error saving subjects", http.StatusInternalServerError)
			return
		}
	}

	if len(delNames) > 0 {
		if err := store.DeleteSubjects(delNames); err != nil {
			http.Error(w, "Error deleting subjects", http.StatusInternalServerError)
			return
		}
	}

	subjects, err := store.ListSubjects()
	if err != nil {
		http.Error(w, "Error fetching subjects", http.StatusInternalServerError)
		return
	}

	names := make([]string, len(subjects))
	for i, s := range subjects {
		names[i] = s.Name
	}

	adminMessage.AdminMessage("✅ Materias actualizadas", "La materia fue actualizada", "", "").Render(r.Context(), w)
	adminSubjectManager.AdminSubjectManager(names, false).Render(r.Context(), w)
}

func HandleAdminSubjectRenameDefault(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminSubjectRenameDefault] Request received")

	subjects, err := store.ListSubjects()
	if err != nil {
		http.Error(w, "Error fetching subjects", http.StatusInternalServerError)
		return
	}

	names := make([]string, len(subjects))
	for i, s := range subjects {
		names[i] = s.Name
	}

	render.RenderWithLayout(w, r, adminSubjectRename.AdminSubjectRename(names), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminSubjectRenameUpdate(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminSubjectRename] Request received")

	raw := r.FormValue("subject_data")
	if raw == "" {
		http.Error(w, "No subject data", http.StatusBadRequest)
		return
	}

	var payload struct {
		Rename []struct {
			Old string `json:"old"`
			New string `json:"new"`
		} `json:"rename"`
	}

	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	renames := make([]sqlite.SubjectRename, 0, len(payload.Rename))
	for _, p := range payload.Rename {
		oldName := strings.TrimSpace(p.Old)
		newName := strings.TrimSpace(p.New)
		if oldName == "" || newName == "" {
			continue
		}
		renames = append(renames, sqlite.SubjectRename{
			Old: oldName,
			New: newName,
		})
	}

	if len(renames) > 0 {
		if err := store.RenameSubjects(renames); err != nil {
			http.Error(w, "Error renaming subjects", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusNoContent)
}
