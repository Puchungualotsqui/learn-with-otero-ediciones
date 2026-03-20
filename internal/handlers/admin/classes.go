package admin

import (
	"encoding/json"
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminClassCreate"
	"frontend/templates/components/admin/adminClassModify"
	"frontend/templates/components/admin/adminClassModifyForm"
	"frontend/templates/components/admin/adminClassSearch"
	"frontend/templates/components/admin/adminClassSearchResults"
	"frontend/templates/components/admin/adminMessage"
	"net/http"
	"strconv"
	"strings"
)

func HandleAdminClassCreateDefault(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassCreate] Request received")

	subjects, err := database.List[models.Subject](store, database.Buckets["subjects"], 150)
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

func HandleAdminClassCreatePost(store *database.Store, w http.ResponseWriter, r *http.Request) {
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

	class, err := database.CreateClass(store, name, description, subject, grade)
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

func HandleAdminClassModifySearch(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassModifySearch] Request received")

	classId := r.URL.Query().Get("class_id")
	class, err := database.Get[models.Class](store, database.Buckets["classes"], classId)
	if err != nil {
		fmt.Printf("Class no encontrado: %v\n", err)
		http.Error(w, "Class no encontrado", http.StatusNotFound)
		return
	}

	users, err := database.GetManyWithPrefix[models.User](store, database.Buckets["users"], class.Users)
	if err != nil {
		fmt.Printf("Error getting users: %v\n", err)
		http.Error(w, "Error getting users", http.StatusNotFound)
		return
	}

	subjects, err := database.List[models.Subject](store, database.Buckets["subjects"], 150)
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

func HandleAdminClassModifyUpdate(store *database.Store, w http.ResponseWriter, r *http.Request) {
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

	database.UpdateWithPrefix(store, database.Buckets["classes"], func(t *models.Class) error {
		t.Id = classIdInt
		t.Description = description
		t.Grade = grade
		t.Name = name
		t.Subject = subject
		return nil
	}, classId)

	for _, id := range payload.Add {
		database.AddUserToClass(store, classIdInt, id)
	}

	for _, id := range payload.Keep {
		database.AddUserToClass(store, classIdInt, id)
	}

	for _, id := range payload.Del {
		database.RemoveUserFromClass(store, classIdInt, id)
	}

	message := fmt.Sprintf("Class: %s . Was modified", classId)

	adminMessage.AdminMessage("Clase actualizada", message, "", "").Render(r.Context(), w)
}

func HandleAdminClassSearchDefault(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassSearchDefault] Request received")

	subjects, err := database.List[models.Subject](store, database.Buckets["subjects"], 150)
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

func HandleAdminClassSearchLookUp(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminClassSearchLookUp] Request received")

	idQuery := strings.TrimSpace(r.URL.Query().Get("id"))
	nameQuery := strings.TrimSpace(r.URL.Query().Get("name"))
	descQuery := strings.TrimSpace(r.URL.Query().Get("description"))
	grade := r.URL.Query().Get("grade")
	subject := strings.TrimSpace(r.URL.Query().Get("subject"))

	classes, err := database.List[models.Class](store, database.Buckets["classes"], -1)
	if err != nil {
		http.Error(w, "Error fetching classes", http.StatusInternalServerError)
		return
	}

	fmt.Println("classes found were: ", len(classes))

	var results []*models.Class
	for _, c := range classes {
		// ID fuzzy search
		if idQuery != "" {
			idStr := strconv.Itoa(c.Id)
			if !strings.Contains(idStr, idQuery) {
				continue
			}
		}

		// Name fuzzy search
		if nameQuery != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(nameQuery)) {
			continue
		}

		// Description fuzzy search
		if descQuery != "" && !strings.Contains(strings.ToLower(c.Description), strings.ToLower(descQuery)) {
			continue
		}

		// Grade exact
		if grade != "" && c.Grade != grade {
			continue
		}

		// Subject fuzzy
		if subject != "" && !strings.Contains(strings.ToLower(c.Subject), strings.ToLower(subject)) {
			continue
		}

		results = append(results, c)
	}

	adminClassSearchResults.AdminClassSearchResults(results).Render(r.Context(), w)
}
