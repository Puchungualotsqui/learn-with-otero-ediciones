package handlers

import (
	"fmt"
	"frontend/database"
	"frontend/dto"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminCreateUser"
	"frontend/templates/components/admin/adminHome"
	"frontend/templates/components/admin/adminMessage"
	"net/http"
)

var options []*dto.AdminOption = []*dto.AdminOption{
	&dto.AdminOption{
		Title:       "Gestión de Usuarios",
		Description: "Ver y modificar información de los usuarios",
		SubUrl:      "user",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Crear",
				Url:   "create",
			},
			&dto.AdminSubOptionSlot{
				Title: "Modificar",
				Url:   "modify",
			},
			&dto.AdminSubOptionSlot{
				Title: "Buscar",
				Url:   "search",
			},
		},
	},
	&dto.AdminOption{
		Title:       "Clases",
		Description: "Crear, buscar y administrar clases",
		SubUrl:      "class",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Crear",
				Url:   "create",
			},
			&dto.AdminSubOptionSlot{
				Title: "Buscar",
				Url:   "search",
			},
		},
	},
	&dto.AdminOption{
		Title:       "Gestión de Materias",
		Description: "Crear, editar y administrar información de materias",
		SubUrl:      "user",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Administrar",
				Url:   "manage",
			},
		},
	},
	&dto.AdminOption{
		Title:       "Recursos",
		Description: "Agregar y eliminar materiales y archivos",
		SubUrl:      "class",
		SubOptions: []*dto.AdminSubOptionSlot{
			&dto.AdminSubOptionSlot{
				Title: "Administrar",
				Url:   "manage",
			},
		},
	},
}

func HandleAdminDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminDefault] Request received")

	render.RenderWithLayout(w, r, adminHome.AdminHome(options), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserCreateDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserCreate] Request received")

	render.RenderWithLayout(w, r, adminCreateUser.AdminCreateUser(), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserCreatePost(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserCreatePost] Request received")

	// --- Parse form
	if err := r.ParseForm(); err != nil {
		fmt.Printf("❌ [AdminUserCreatePost] Failed to parse form: %v\n", err)
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	role := r.FormValue("role")
	school := r.FormValue("school")
	grade := r.FormValue("grade")

	// --- Input validation
	if firstName == "" || lastName == "" || role == "" || school == "" || grade == "" {
		fmt.Printf("⚠️ [AdminUserCreatePost] Missing required fields — firstName=%q, lastName=%q, role=%q, school=%q, grade=%q\n",
			firstName, lastName, role, school, grade)
		http.Error(w, "Nombre, apellido, rol, colegio y grado son obligatorios", http.StatusBadRequest)
		return
	}

	fmt.Printf("➡️ [AdminUserCreatePost] Creating user: %s %s | Role=%s | School=%s | Grade=%s\n",
		firstName, lastName, role, school, grade)

	// --- Attempt creation
	user, err := database.CreateUser(store, "", "", firstName, lastName, role, school, grade)
	if err != nil {
		fmt.Printf("❌ [AdminUserCreatePost] Error creating user: %v\n", err)
		http.Error(w, fmt.Sprintf("Error creando usuario: %v", err), http.StatusInternalServerError)
		return
	}

	message := fmt.Sprintf("Nombre de usuario: %s Contraseña: %s",
		user.Username, user.PasswordNotHashed)

	adminMessage.AdminMessage(message).Render(r.Context(), w)

	fmt.Printf("✅ [AdminUserCreatePost] User created successfully — Username=%s | Role=%s\n", user.Username, role)
	fmt.Println("  ✔ Render complete")
}
