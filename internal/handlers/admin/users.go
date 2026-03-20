package admin

import (
	"encoding/json"
	"fmt"
	"frontend/auth"
	"frontend/database"
	"frontend/database/models"
	"frontend/helper"
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
	"slices"
	"strconv"
	"strings"
	"unicode"
)

func HandleAdminUserCreateDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserCreate] Request received")

	render.RenderWithLayout(w, r, adminUserCreate.AdminCreateUser(), body.Home)
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
	email := r.FormValue("email")
	phoneNumber := r.FormValue("phone")

	// --- Input validation
	if firstName == "" || lastName == "" || role == "" || school == "" || grade == "" || email == "" || phoneNumber == "" {
		fmt.Printf("⚠️ [AdminUserCreatePost] Missing required fields — firstName=%q, lastName=%q, role=%q, school=%q, grade=%q\n",
			firstName, lastName, role, school, grade)
		http.Error(w, "Nombre, apellido, rol, colegio y grado son obligatorios", http.StatusBadRequest)
		return
	}

	fmt.Printf("➡️ [AdminUserCreatePost] Creating user: %s %s | Role=%s | School=%s | Grade=%s\n",
		firstName, lastName, role, school, grade)

	// --- Attempt creation
	user, err := database.CreateUser(store, "", "", firstName, lastName, role, school, grade, email, phoneNumber)
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
	message := fmt.Sprintf("Nombre de usuario: %s Contraseña: %s",
		user.Username, user.PasswordNotHashed)

	adminMessage.AdminMessage(message).Render(r.Context(), w)

	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserModifyDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserModifyDefault] Request received")

	username := r.URL.Query().Get("username")

	render.RenderWithLayout(w, r, adminUserModify.AdminUserModify(username), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserModifySearch(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserModifySearch] Request received")

	username := r.URL.Query().Get("username")
	user, err := database.Get[models.User](store, database.Buckets["users"], username)
	if err != nil {
		fmt.Printf("Usuario no encontrado: %v\n", err)
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	classes, err := database.GetManyWithPrefix[models.Class](store, database.Buckets["classes"], helper.IntsToStrings(user.Classes...))
	if err != nil {
		fmt.Printf("Error getting classes: %v\n", err)
		http.Error(w, "Error getting classes", http.StatusNotFound)
		return
	}
	adminUserModifyForm.AdminUserModifyForm(user, classes).Render(r.Context(), w)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserModifyUpdate(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminUserUpdate triggered")

	r.ParseForm()
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

	database.UpdateWithPrefix(store, database.Buckets["users"], func(t *models.User) error {
		t.FirstName = first
		t.LastName = last
		t.Role = role
		t.School = school
		t.Grade = grade
		t.PhoneNumber = phone
		t.Email = email

		return nil
	}, username)

	for _, id := range payload.Add {
		database.AddUserToClass(store, id, username)
	}

	for _, id := range payload.Keep {
		database.AddUserToClass(store, id, username)
	}

	for _, id := range payload.Del {
		database.RemoveUserFromClass(store, id, username)
	}

	message := fmt.Sprintf("User: %s . Was modified", username)

	adminMessage.AdminMessage(message).Render(r.Context(), w)
}

func HandleAdminUserRevealPassword(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminUserRevealPassword triggered")
	username := r.URL.Query().Get("username")
	key := r.Header.Get("HX-Prompt") // from hx-prompt

	// Validate admin password here (compare hash, or session role == admin)
	user, err := database.Get[models.User](store, database.Buckets["users"], username)
	if err != nil {
		println("Usuario no encontrado")
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	plain, err := auth.Decrypt([]byte(key), user.PasswordNotHashed)
	if err != nil {
		println("Error al descifrar la contrasena: ", err)
		http.Error(w, "Error al descifrar contraseña", http.StatusInternalServerError)
		return
	}

	safePassword := html.EscapeString(plain)

	println("safe password: ", safePassword)

	// Return a new input field that HTMX swaps in
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<input type="text" id="password-field"
			value="%s"
			readonly
			class="input input-bordered bg-white text-gray-800 text-sm w-40 cursor-not-allowed" />`, safePassword)
}

func HandleAdminUserRememberPassword(store *database.Store, w http.ResponseWriter, r *http.Request) {
	fmt.Println("🧾 HandleAdminUserRememberPassword triggered")
	username := r.URL.Query().Get("username")
	key := os.Getenv("ENC_KEY")

	// Validate admin password here (compare hash, or session role == admin)
	user, err := database.Get[models.User](store, database.Buckets["users"], username)
	if err != nil {
		println("Usuario no encontrado")
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	plain, err := auth.Decrypt([]byte(key), user.PasswordNotHashed)
	if err != nil {
		println("Error al descifrar la contrasena: ", err)
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

	adminMessage.AdminMessage("Se ha enviado un correo con tu contraseña.").Render(r.Context(), w)
}

func HandleAdminUserSearchDefault(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📥 [HandleAdminUserSearchDefault] Request received")

	render.RenderWithLayout(w, r, adminUserSearch.AdminUserSearch(), body.Home)
	fmt.Println("  ✔ Render complete")
}

func HandleAdminUserSearchLookUp(store *database.Store, w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	role := r.URL.Query().Get("role")
	grade := r.URL.Query().Get("grade")
	school := strings.TrimSpace(r.URL.Query().Get("school"))
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	classID := r.URL.Query().Get("class")

	users, err := database.List[models.User](store, database.Buckets["users"], -1)
	if err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	fmt.Println("users found were:", len(users))

	var results []*models.User
	for _, u := range users {
		// Fuzzy match: name or username
		if query != "" {
			q := strings.ToLower(strings.TrimSpace(query))
			fullName := strings.ToLower(strings.TrimSpace(u.FirstName + " " + u.LastName))

			if !strings.Contains(strings.ToLower(u.Username), q) &&
				!strings.Contains(strings.ToLower(u.FirstName), q) &&
				!strings.Contains(strings.ToLower(u.LastName), q) &&
				!strings.Contains(fullName, q) {
				continue
			}
		}

		// School fuzzy
		if school != "" && !strings.Contains(strings.ToLower(u.School), strings.ToLower(school)) {
			continue
		}

		// Role exact
		if role != "" && u.Role != role {
			continue
		}

		// Grade exact
		if grade != "" && u.Grade != grade {
			continue
		}

		// Email exact
		if email != "" && !strings.EqualFold(u.Email, email) {
			continue
		}

		// Phone exact
		if phone != "" {
			phoneNum := u.PhoneNumber
			if phoneNum != phone {
				continue
			}
		}

		// Class ID filter
		if classID != "" {
			id, err := strconv.Atoi(strings.TrimSpace(classID))
			if err != nil {
				http.Error(w, "Invalid class id", http.StatusBadRequest)
				return
			}
			if !slices.Contains(u.Classes, id) {
				continue
			}
		}

		results = append(results, u)
	}

	adminUserSearchResults.AdminUserSearchResults(results).Render(r.Context(), w)
}
