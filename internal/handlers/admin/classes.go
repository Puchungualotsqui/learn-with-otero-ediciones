package admin

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"frontend/database/sqlite"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminClassCreate"
	"frontend/templates/components/admin/adminClassModify"
	"frontend/templates/components/admin/adminClassModifyForm"
	"frontend/templates/components/admin/adminClassSearch"
	"frontend/templates/components/admin/adminClassSearchResults"
	"frontend/templates/components/admin/adminMessage"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func HandleAdminClassCreateDefault(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassCreate] Request received")

	subjects, err := store.ListSubjects()
	if err != nil {
		fmt.Printf("Materias no encontradas: %v\n", err)
		http.Error(w, "Materias no encontradas", http.StatusNotFound)
		return
	}

	subjectsArray := make([]string, len(subjects))
	for i, subject := range subjects {
		subjectsArray[i] = subject.Name
	}

	render.RenderWithLayout(w, r, adminClassCreate.AdminCreateClass(subjectsArray), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminClassCreatePost(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassCreatePost] Request received")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	grade := strings.TrimSpace(r.FormValue("grade"))

	if name == "" || subject == "" || description == "" || grade == "" {
		fmt.Printf("Todos los campos son obligatorios\n")
		http.Error(w, "Todos los campos son obligatorios", http.StatusBadRequest)
		return
	}

	class, err := store.CreateClass(name, description, grade, subject)
	if err != nil {
		fmt.Printf("Error creating the class %v\n", err)
		http.Error(w, "Error creating the class", http.StatusBadRequest)
		return
	}

	message := fmt.Sprintf("Clase creada con éxito (ID: %d) — %s (%s)", class.Id, name, subject)
	modifyURL := fmt.Sprintf("/admin/class/modify?class_id=%d", class.Id)

	adminMessage.AdminMessage(
		"✅ Clase creada correctamente",
		message,
		modifyURL,
		"Ir a modificar esta clase",
	).Render(r.Context(), w)

	fmt.Printf("✅ [AdminClassCreatePost] Class created — ID=%d | Name=%s\n", class.Id, name)
}

func HandleAdminClassModifyDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassModifyDefault] Request received")

	classId := r.URL.Query().Get("class_id")

	render.RenderWithLayout(w, r, adminClassModify.AdminClassModify(classId), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminClassModifySearch(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassModifySearch] Request received")

	classId, err := strconv.Atoi(r.URL.Query().Get("class_id"))
	if err != nil {
		http.Error(w, "Class inválida", http.StatusBadRequest)
		return
	}

	class, err := store.GetClass(classId)
	if err != nil {
		fmt.Printf("Class no encontrado: %v\n", err)
		http.Error(w, "Class no encontrado", http.StatusNotFound)
		return
	}

	users, err := store.GetUsersByClassID(classId)
	if err != nil {
		fmt.Printf("Error getting users: %v\n", err)
		http.Error(w, "Error getting users", http.StatusNotFound)
		return
	}

	subjects, err := store.ListSubjects()
	if err != nil {
		fmt.Printf("Materias no encontradas: %v\n", err)
		http.Error(w, "Materias no encontradas", http.StatusNotFound)
		return
	}

	subjectsArray := make([]string, len(subjects))
	for i, subject := range subjects {
		subjectsArray[i] = subject.Name
	}

	adminClassModifyForm.AdminClassModifyForm(class, users, subjectsArray).Render(r.Context(), w)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminClassModifyUpdate(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminClassModifyUpdate triggered")

	r.ParseForm()
	fmt.Printf("🧩 PostForm: %#v\n", r.PostForm)
	fmt.Println("🔹 Raw user_data:", r.FormValue("user_data"))

	var payload struct {
		Add  []string `json:"add"`
		Keep []string `json:"keep"`
		Del  []string `json:"del"`
	}

	raw := r.FormValue("user_data")
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			fmt.Println("❌ Error parsing class_data:", err)
		}
	}

	classId := r.FormValue("class_id")
	description := r.FormValue("description")
	grade := r.FormValue("grade")
	name := r.FormValue("name")
	subject := r.FormValue("subject")

	if classId == "" || description == "" || grade == "" || name == "" || subject == "" {
		http.Error(w, "Missing field", http.StatusBadRequest)
		return
	}

	classIdInt, err := strconv.Atoi(classId)
	if err != nil {
		fmt.Printf("Invalid classId: %v\n", err)
		http.Error(w, "Invalid classId", http.StatusNotFound)
		return
	}

	if err := store.UpdateClass(classIdInt, name, description, grade, subject); err != nil {
		fmt.Printf("❌ Error updating class: %v\n", err)
		http.Error(w, "Error updating class", http.StatusInternalServerError)
		return
	}

	for _, username := range payload.Add {
		if err := store.AddUserToClass(classIdInt, username); err != nil {
			fmt.Printf("❌ Error adding user %s to class %d: %v\n", username, classIdInt, err)
			http.Error(w, "Error updating class users", http.StatusInternalServerError)
			return
		}
	}

	for _, username := range payload.Keep {
		if err := store.AddUserToClass(classIdInt, username); err != nil {
			fmt.Printf("❌ Error keeping user %s in class %d: %v\n", username, classIdInt, err)
			http.Error(w, "Error updating class users", http.StatusInternalServerError)
			return
		}
	}

	for _, username := range payload.Del {
		if err := store.RemoveUserFromClass(classIdInt, username); err != nil {
			fmt.Printf("❌ Error removing user %s from class %d: %v\n", username, classIdInt, err)
			http.Error(w, "Error updating class users", http.StatusInternalServerError)
			return
		}
	}

	message := fmt.Sprintf("Class: %s . Was modified", classId)
	adminMessage.AdminMessage("Clase actualizada", message, "", "").Render(r.Context(), w)
}

func HandleAdminClassModifyDelete(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminClassModifyDelete triggered")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	classID := strings.TrimSpace(r.FormValue("class_id"))
	if classID == "" {
		classID = strings.TrimSpace(r.URL.Query().Get("class_id"))
	}
	if classID == "" {
		http.Error(w, "Clase no especificada", http.StatusBadRequest)
		return
	}

	enteredPassword := r.Header.Get("HX-Prompt")
	expectedPassword := os.Getenv("ADMIN_PASSWORD")

	if expectedPassword == "" {
		http.Error(w, "ADMIN_PASSWORD no configurado", http.StatusInternalServerError)
		return
	}

	if subtle.ConstantTimeCompare([]byte(enteredPassword), []byte(expectedPassword)) != 1 {
		http.Error(w, "Contraseña de administrador incorrecta", http.StatusForbidden)
		return
	}

	classIDInt, err := strconv.Atoi(classID)
	if err != nil {
		http.Error(w, "ID de clase inválido", http.StatusBadRequest)
		return
	}

	if err := store.DeleteClass(classIDInt); err != nil {
		fmt.Printf("❌ Error deleting class %s: %v\n", classID, err)
		http.Error(w, "Error eliminando clase", http.StatusInternalServerError)
		return
	}

	fmt.Printf("✅ Class deleted: %s\n", classID)

	w.Header().Set("HX-Redirect", "/admin/class/search")
	w.WriteHeader(http.StatusOK)
}

func HandleAdminClassSearchDefault(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassSearchDefault] Request received")

	subjects, err := store.ListSubjects()
	if err != nil {
		fmt.Printf("Materias no encontradas: %v\n", err)
		http.Error(w, "Materias no encontradas", http.StatusNotFound)
		return
	}

	subjectsArray := make([]string, len(subjects))
	for i, subject := range subjects {
		subjectsArray[i] = subject.Name
	}

	render.RenderWithLayout(w, r, adminClassSearch.AdminClassSearch(subjectsArray), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminClassSearchLookUp(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassSearchLookUp] Request received")

	idQuery := strings.TrimSpace(r.URL.Query().Get("id"))
	nameQuery := strings.TrimSpace(r.URL.Query().Get("name"))
	descQuery := strings.TrimSpace(r.URL.Query().Get("description"))
	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	subject := strings.TrimSpace(r.URL.Query().Get("subject"))

	results, err := store.SearchClasses(idQuery, nameQuery, descQuery, grade, subject)
	if err != nil {
		fmt.Printf("❌ Error searching classes: %v\n", err)
		http.Error(w, "Error fetching classes", http.StatusInternalServerError)
		return
	}

	fmt.Println("classes found were:", len(results))
	adminClassSearchResults.AdminClassSearchResults(results).Render(r.Context(), w)
}
