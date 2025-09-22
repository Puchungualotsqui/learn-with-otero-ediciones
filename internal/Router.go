package internal

import (
	"fmt"
	"frontend/auth"
	"frontend/database"
	"frontend/database/models"
	"frontend/templates/body"
	"frontend/templates/components/assignment"
	"frontend/templates/components/class"
	"frontend/templates/components/home"
	"net/http"
	"strings"
)

func Router(store *database.Store, w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	cookie, err := r.Cookie("session_user")
	if parts[0] != "login" { // protect everything except /login
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if !database.Exists(store, database.Buckets["users"], cookie.Value) {
			// invalid cookie, force re-login
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
	}

	switch parts[0] {
	case "login":
		switch r.Method {
		case http.MethodGet:
			RenderWithLayout(w, r, body.Auth())

		case http.MethodPost:
			username := r.FormValue("username")
			password := r.FormValue("password")

			user, err := database.Get[models.User](store, database.Buckets["users"], username)
			if err != nil {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Usuario no encontrado"))
				return
			}

			if !auth.CheckPassword(user.PasswordHashed, password) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Contraseña incorrecta"))
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "session_user",
				Value:    username,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // set true in production with HTTPS
				SameSite: http.SameSiteLaxMode,
			})
			w.Header().Set("HX-Redirect", "/")

		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}

	case "":
		slotsInfo := []class.SlotInfo{
			{Title: "Matemáticas", SubTitle: "Nivel: Secundaria"},
			{Title: "Historia", SubTitle: "Nivel: Secundaria"},
			{Title: "Ciencias", SubTitle: "Nivel: Primaria"},
		}

		RenderWithLayout(w, r, home.Home(slotsInfo), body.Home)

	case "matematicas", "historia", "ciencias":
		if len(parts) > 1 {
			switch parts[1] {
			case "asignaciones":
				subject := parts[0]
				assignments := loadAssignments(subject)
				RenderWithLayout(w, r, assignment.AssignmentContent(subject, assignments), body.Home)
				return
			}
		}
		http.NotFound(w, r)

	default:
		http.NotFound(w, r)
	}
}

func loadAssignments(store *database.Store, classID int) []assignment.Assignment {
	var results []assignment.Assignment

	prefix := fmt.Sprintf("%d:", classID) // keys look like "classID:assignmentID"
	assignments, err := database.GetWithPrefix[models.Assignment](store, database.Buckets["assignments"], prefix)
	if err == nil {
		results = append(results, assignments...)
	}

	return results
}
