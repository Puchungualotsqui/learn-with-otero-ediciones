package internal

import (
	"frontend/auth"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/templates/body"
	"frontend/templates/components/assignment"
	"frontend/templates/components/home"
	"log"
	"net/http"
	"strconv"
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

	switch {
	case parts[0] == "login":
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

	case parts[0] == "":
		classes, err := database.ListClassesForUser(store, cookie.Value)
		if err != nil {
			log.Printf("⚠️ fallback: user %s classes not loaded: %v", cookie.Value, err)
			classes = []models.Class{}
		}
		slotsInfo := dto.ClassSlotFromModels(classes)

		RenderWithLayout(w, r, home.Home(slotsInfo), body.Home)

	case isClassValid(store, cookie.Value, parts[0]):
		if len(parts) > 1 {
			switch parts[1] {
			case "asignaciones":
				subject := parts[0]
				classId, _ := strconv.Atoi(parts[0])
				assignments := database.ListAssignmentsOfClass(store, classId)
				RenderWithLayout(w, r, assignment.AssignmentContent(subject, assignments), body.Home)
				return
			}
		}
		http.NotFound(w, r)

	default:
		http.NotFound(w, r)
	}
}

func isClassValid(store *database.Store, username, classId string) bool {
	classes, err := database.ListClassesForUser(store, username)
	if err != nil {
		return false
	}
	classIds, err := strconv.Atoi(classId)
	if err != nil {
		return false
	}

	for _, class := range classes {
		if class.Id == classIds {
			return true
		}
	}
	return false
}
