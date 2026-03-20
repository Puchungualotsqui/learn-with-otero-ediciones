package admin

import (
	"encoding/json"
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminMessage"
	"frontend/templates/components/admin/adminSubjectManager"
	"frontend/templates/components/admin/adminSubjectRename"
	"net/http"
	"strconv"
	"strings"
)

func HandleAdminSubjectsManagerDefault(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminSubjectsDefault] Request received")

	subjects, err := database.List[models.Subject](store, database.Buckets["subjects"], 200)
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

func HandleAdminSubjectManagerUpdate(store *database.Store, w http.ResponseWriter, r *http.Request) {
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

	// --- Create new subjects
	addMap := make(map[string]models.Subject)
	for _, name := range payload.Add {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		addMap[name] = models.Subject{Name: name}
	}

	if len(addMap) > 0 {
		if err := database.SaveMany(store, database.Buckets["subjects"], addMap); err != nil {
			http.Error(w, "Error saving subjects", http.StatusInternalServerError)
			return
		}
	}

	// --- Delete subjects
	if len(payload.Del) > 0 {
		if err := database.DeleteMany(store, database.Buckets["subjects"], payload.Del); err != nil {
			http.Error(w, "Error deleting subjects", http.StatusInternalServerError)
			return
		}
	}

	subjects, err := database.List[models.Subject](store, database.Buckets["subjects"], 200)
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

func HandleAdminSubjectRenameDefault(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminSubjectRenameDefault] Request received")

	subjects, err := database.List[models.Subject](store, database.Buckets["subjects"], 200)
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

func HandleAdminSubjectRenameUpdate(store *database.Store, w http.ResponseWriter, r *http.Request) {
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

	oldSubjects := make([]string, 0, len(payload.Rename))
	newSubjects := make([]string, 0, len(payload.Rename))
	for _, p := range payload.Rename {
		if strings.TrimSpace(p.Old) == "" || strings.TrimSpace(p.New) == "" {
			continue
		}
		oldSubjects = append(oldSubjects, p.Old)
		newSubjects = append(newSubjects, p.New)
	}

	// 1. Create new subjects
	addMap := make(map[string]models.Subject, len(newSubjects))
	for _, newSub := range newSubjects {
		addMap[newSub] = models.Subject{Name: newSub}
	}

	if len(addMap) > 0 {
		if err := database.SaveMany(store, database.Buckets["subjects"], addMap); err != nil {
			http.Error(w, "Error saving renamed subjects", http.StatusInternalServerError)
			return
		}
	}

	// 2. Delete old subjects
	if len(oldSubjects) > 0 {
		if err := database.DeleteMany(store, database.Buckets["subjects"], oldSubjects); err != nil {
			http.Error(w, "Error deleting old subjects", http.StatusInternalServerError)
			return
		}
	}

	// 3. Update classes referencing renamed subjects
	classList, err := database.List[models.Class](store, database.Buckets["classes"], -1)
	if err != nil {
		http.Error(w, "Error listing classes", http.StatusBadRequest)
		return
	}

	classKeys := make([]string, len(classList))
	for i, c := range classList {
		classKeys[i] = strconv.Itoa(c.Id)
	}

	if err := database.UpdateManyWithPrefix(store, database.Buckets["classes"], func(t *models.Class) error {
		for i, oldName := range oldSubjects {
			if t.Subject == oldName {
				t.Subject = newSubjects[i]
				break
			}
		}
		return nil
	}, classKeys); err != nil {
		http.Error(w, "Error updating class subjects", http.StatusBadRequest)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusNoContent)
}
