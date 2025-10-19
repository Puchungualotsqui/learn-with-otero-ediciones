package admin

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminClassCreate"
	"frontend/templates/components/admin/adminMessage"
	"net/http"
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
	adminMessage.AdminMessage(message).Render(r.Context(), w)
	fmt.Printf("✅ [AdminClassCreatePost] Class created — ID=%d | Name=%s\n", class.Id, name)
}
