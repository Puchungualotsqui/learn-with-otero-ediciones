package admin

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"frontend/auth"
	"frontend/database/sqlite"
	"frontend/internal/mail"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/admin/adminMessage"
	"frontend/templates/components/admin/adminUserCreate"
	"frontend/templates/components/admin/adminUserModify"
	"frontend/templates/components/admin/adminUserModifyForm"
	"frontend/templates/components/admin/adminUserSearch"
	"frontend/templates/components/admin/adminUserSearchResults"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"
)

func HandleAdminUserCreateDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserCreate] Request received")

	render.RenderWithLayout(w, r, adminUserCreate.AdminCreateUser(), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserCreatePost(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserCreatePost] Request received")

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
	email := r.FormValue("email")
	phoneNumber := r.FormValue("phone")

	if firstName == "" || lastName == "" || role == "" || school == "" || grade == "" || email == "" || phoneNumber == "" {
		fmt.Printf("⚠️ [AdminUserCreatePost] Missing required fields — firstName=%q, lastName=%q, role=%q, school=%q, grade=%q\n",
			firstName, lastName, role, school, grade)
		http.Error(w, "Nombre, apellido, rol, colegio y grado son obligatorios", http.StatusBadRequest)
		return
	}

	fmt.Printf("➡️ [AdminUserCreatePost] Creating user: %s %s | Role=%s | School=%s | Grade=%s\n",
		firstName, lastName, role, school, grade)

	user, err := store.CreateUser("", "", firstName, lastName, role, school, grade, email, phoneNumber)
	if err != nil {
		fmt.Printf("❌ [AdminUserCreatePost] Error creating user: %v\n", err)
		http.Error(w, fmt.Sprintf("Error creando usuario: %v", err), http.StatusInternalServerError)
		return
	}

	fullName := strings.TrimRightFunc(user.FirstName, unicode.IsSpace) + " " + user.LastName

	go func() {
		err := mail.SendWelcomeEmail(user.Email, user.Username, fullName, user.PasswordNotHashed)
		if err != nil {
			fmt.Printf("❌ Error enviando email de bienvenida a %s: %v\n", user.Email, err)
		}
	}()

	message := fmt.Sprintf("Nombre de usuario: %s | Contraseña: %s",
		user.Username, user.PasswordNotHashed)

	modifyURL := fmt.Sprintf("/admin/user/modify?username=%s", user.Username)

	adminMessage.AdminMessage(
		"✅ Usuario creado correctamente",
		message,
		modifyURL,
		"Ir a modificar este usuario",
	).Render(r.Context(), w)

	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserModifyDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserModifyDefault] Request received")

	username := r.URL.Query().Get("username")

	render.RenderWithLayout(w, r, adminUserModify.AdminUserModify(username), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserModifySearch(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserModifySearch] Request received")

	username := r.URL.Query().Get("username")

	user, err := store.GetUser(username)
	if err != nil {
		fmt.Printf("Usuario no encontrado: %v\n", err)
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	classes, err := store.GetClassesByUsername(username)
	if err != nil {
		fmt.Printf("Error getting classes: %v\n", err)
		http.Error(w, "Error getting classes", http.StatusNotFound)
		return
	}

	adminUserModifyForm.AdminUserModifyForm(user, classes).Render(r.Context(), w)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserModifyUpdate(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminUserUpdate triggered")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	fmt.Printf("🧩 PostForm: %#v\n", r.PostForm)
	fmt.Println("🔹 Raw class_data:", r.FormValue("class_data"))

	var payload struct {
		Add  []int `json:"add"`
		Keep []int `json:"keep"`
		Del  []int `json:"del"`
	}

	raw := r.FormValue("class_data")
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			fmt.Println("❌ Error parsing class_data:", err)
			http.Error(w, "Invalid class data", http.StatusBadRequest)
			return
		}
	}

	username := r.FormValue("username")
	first := r.FormValue("first_name")
	last := r.FormValue("last_name")
	role := r.FormValue("role")
	school := r.FormValue("school")
	grade := r.FormValue("grade")
	phone := r.FormValue("phone")
	email := r.FormValue("email")

	if username == "" || first == "" || last == "" || role == "" || school == "" || grade == "" || phone == "" || email == "" {
		http.Error(w, "Missing field", http.StatusBadRequest)
		return
	}

	if err := store.UpdateUser(username, first, last, role, school, grade, phone, email); err != nil {
		fmt.Printf("❌ Error updating user: %v\n", err)
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	for _, id := range payload.Add {
		if err := store.AddUserToClass(id, username); err != nil {
			fmt.Printf("❌ Error adding user %s to class %d: %v\n", username, id, err)
			http.Error(w, "Error updating user classes", http.StatusInternalServerError)
			return
		}
	}

	for _, id := range payload.Keep {
		if err := store.AddUserToClass(id, username); err != nil {
			fmt.Printf("❌ Error keeping user %s in class %d: %v\n", username, id, err)
			http.Error(w, "Error updating user classes", http.StatusInternalServerError)
			return
		}
	}

	for _, id := range payload.Del {
		if err := store.RemoveUserFromClass(id, username); err != nil {
			fmt.Printf("❌ Error removing user %s from class %d: %v\n", username, id, err)
			http.Error(w, "Error updating user classes", http.StatusInternalServerError)
			return
		}
	}

	message := fmt.Sprintf("User: %s . Was modified", username)

	adminMessage.AdminMessage(
		"✅ Usuario actualizado",
		message,
		"",
		"",
	).Render(r.Context(), w)
}

func HandleAdminUserRevealPassword(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminUserRevealPassword triggered")

	username := r.URL.Query().Get("username")
	key := r.Header.Get("HX-Prompt")

	user, err := store.GetUser(username)
	if err != nil {
		fmt.Println("Usuario no encontrado")
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	plain, err := auth.Decrypt([]byte(key), user.PasswordNotHashed)
	if err != nil {
		fmt.Println("Error al descifrar la contrasena:", err)
		http.Error(w, "Error al descifrar contraseña", http.StatusInternalServerError)
		return
	}

	safePassword := html.EscapeString(plain)
	fmt.Println("safe password:", safePassword)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<input type="text" id="password-field"
			value="%s"
			readonly
			class="input input-bordered bg-white text-gray-800 text-sm w-40 cursor-not-allowed" />`, safePassword)
}

func HandleAdminUserRememberPassword(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminUserRememberPassword triggered")

	username := r.URL.Query().Get("username")
	key := os.Getenv("ENC_KEY")

	user, err := store.GetUser(username)
	if err != nil {
		fmt.Println("Usuario no encontrado")
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	plain, err := auth.Decrypt([]byte(key), user.PasswordNotHashed)
	if err != nil {
		fmt.Println("Error al descifrar la contrasena:", err)
		http.Error(w, "Error al descifrar contraseña", http.StatusInternalServerError)
		return
	}

	fullName := strings.TrimRightFunc(user.FirstName, unicode.IsSpace) + " " + user.LastName
	go func() {
		err := mail.SendRememberPasswordEmail(user.Email, user.Username, fullName, plain)
		if err != nil {
			fmt.Printf("❌ Error enviando recordatorio a %s: %v\n", user.Email, err)
		}
	}()

	adminMessage.AdminMessage(
		"✅ Recordatorio enviado",
		"Se ha enviado un correo con la contraseña del usuario.",
		"",
		"",
	).Render(r.Context(), w)
}

func HandleAdminUserDelete(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminUserDelete triggered")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		username = strings.TrimSpace(r.URL.Query().Get("username"))
	}
	if username == "" {
		http.Error(w, "Usuario no especificado", http.StatusBadRequest)
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

	if username == "admin" {
		http.Error(w, "No se puede eliminar el usuario admin", http.StatusForbidden)
		return
	}

	if err := store.DeleteUser(username); err != nil {
		fmt.Printf("❌ Error deleting user %s: %v\n", username, err)
		http.Error(w, "Error eliminando usuario", http.StatusInternalServerError)
		return
	}

	fmt.Printf("✅ User deleted: %s\n", username)

	w.Header().Set("HX-Redirect", "/admin/user/search")
	w.WriteHeader(http.StatusOK)
}

func HandleAdminUserSearchDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserSearchDefault] Request received")

	render.RenderWithLayout(w, r, adminUserSearch.AdminUserSearch(), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserSearchLookUp(store *sqlite.Store, w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	role := strings.TrimSpace(r.URL.Query().Get("role"))
	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	school := strings.TrimSpace(r.URL.Query().Get("school"))
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	classIDRaw := strings.TrimSpace(r.URL.Query().Get("class"))

	var classID *int
	if classIDRaw != "" {
		id, err := strconv.Atoi(classIDRaw)
		if err != nil {
			http.Error(w, "Invalid class id", http.StatusBadRequest)
			return
		}
		classID = &id
	}

	users, err := store.SearchUsers(query, role, grade, school, email, phone, classID)
	if err != nil {
		fmt.Printf("❌ Error fetching users: %v\n", err)
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	fmt.Println("users found were:", len(users))
	adminUserSearchResults.AdminUserSearchResults(users).Render(r.Context(), w)
}
