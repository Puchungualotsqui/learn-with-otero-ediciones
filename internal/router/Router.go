package router

import (
	"fmt"
	"frontend/auth"
	"frontend/database"
	"frontend/database/models"
	"frontend/dto"
	"frontend/helper"
	"frontend/internal/handlers"
	"frontend/internal/render"
	"frontend/storage"
	"frontend/templates/body"
	"frontend/templates/components/home"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func Router(store *database.Store, storage *storage.B2Storage, w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	var username string
	if parts[0] != "login" { // protect everything except /login
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		username, err = database.GetUserFromSession(store, cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
	}

	switch {
	case parts[0] == "login":
		switch r.Method {
		case http.MethodGet:
			render.RenderWithLayout(w, r, body.Auth())

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

			sessionID, err := database.GenerateSession(store, username)
			if err != nil {
				http.Error(w, "Error creando sesión", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    sessionID,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // set true in production with HTTPS
				SameSite: http.SameSiteLaxMode,
			})

			w.Header().Set("HX-Redirect", "/")
			return

		default:
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

	case parts[0] == "logout":
		cookie, err := r.Cookie("session_id")
		if err == nil {
			err = database.DeleteSession(store, cookie.Value)
		}
		// Clear cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})

		// HX-Redirect header makes HTMX go there
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return

	case parts[0] == "":
		classes, err := database.ListClassesForUser(store, username)
		if err != nil {
			log.Printf("fallback: user %s classes not loaded: %v", username, err)
			classes = []*models.Class{}
		}
		slotsInfo := dto.ClassSlotFromModels(classes)

		professor, err := isProfessor(store, username)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		render.RenderWithLayout(w, r, home.Home(slotsInfo, professor), body.Home)
		return

	case isClassValid(store, username, parts[0]):
		fmt.Println("🔎 Router parts:", parts)

		if len(parts) > 1 {
			classId, err := strconv.Atoi(parts[0])
			if err != nil {
				http.Error(w, "Error with the class id", http.StatusInternalServerError)
			}
			switch parts[1] {
			case "asignaciones":

				assignments := database.ListAssignmentsOfClass(store, classId)
				helper.PrintArray(parts)

				professor, err := isProfessor(store, username)
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}

				fmt.Printf("👉 Checking assignments route: %+v\n", parts)

				if len(parts) > 2 && parts[2] == "update" {
					fmt.Println("📌 Routed to UpdateAssignment (professor)")
					handlers.HandleAssignmentUpdate(store, storage, w, r, classId, professor)
					return
				}

				if len(parts) > 2 && parts[2] == "new" {
					fmt.Println("📌 Routed to NewAssignment (professor)")
					handlers.HandleAssignmentNew(store, storage, w, r, classId, professor)
					return
				}

				if len(parts) > 2 && parts[2] == "delete" {
					fmt.Println("📌 Routed to DeleteAssignment (professor)")
					handlers.HandleAssignmentDelete(store, storage, w, r, classId, professor)
					return
				}

				if len(parts) > 2 && parts[3] == "submissions" {
					fmt.Println("📌 Routed to HandleAssignmentSubmissions")
					handlers.HandleAssignmentSubmissions(store, w, r, professor)
					return
				}

				if len(parts) > 4 && parts[3] == "submission" {
					fmt.Println("📌 Routed to HandleAssignmentSubmissions")
					handlers.HandleAssignmentSubmission(store, w, r, username, professor)
					return
				}

				if len(parts) > 3 && parts[3] == "details" {
					fmt.Println("📌 Routed to HandleAssignmentDetail")
					handlers.HandleAssignmentDetail(w, r, assignments, professor)
					return
				}

				fmt.Print("test starts")

				handlers.HandleAssignmentDefault(w, r, assignments, classId, professor, username)
				return

			case "entregas":
				classId, _ := strconv.Atoi(parts[0])
				assignments := database.ListAssignmentsOfClass(store, classId)

				professor, err := isProfessor(store, username)
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}

				fmt.Printf("👉 Checking entregas route: %+v\n", parts)

				if len(parts) > 4 && parts[3] == "submissions" && parts[4] == "detail" {
					fmt.Println("📌 Routed to HandleSubmissionDetail")
					assignmentId, _ := strconv.Atoi(parts[2])
					handlers.HandleSubmissionDetail(store, w, r, classId, assignmentId, professor)
					return
				}

				if len(parts) > 2 && parts[2] == "detail" {
					fmt.Println("📌 Routed to HandleSubmissionDetail (professor view)")
					assignmentId, _ := strconv.Atoi(parts[2])
					handlers.HandleSubmissionDetail(store, w, r, classId, assignmentId, professor)
					return
				}

				handlers.HandleSubmissionDefault(store, w, r, classId, assignments[0].Id, assignments, professor, username)

				return
			}
		}
		http.NotFound(w, r)
		return

	default:
		http.NotFound(w, r)
		return
	}
}
