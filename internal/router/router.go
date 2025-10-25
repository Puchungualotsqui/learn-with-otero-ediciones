package router

import (
	"fmt"
	"frontend/auth"
	"frontend/database"
	"frontend/database/models"
	"frontend/helper"
	"frontend/internal/handlers/admin"
	"frontend/internal/handlers/commonUsers"
	"frontend/internal/handlers/generics"
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

	helper.PrintArray(parts)

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
		user, err := database.Get[models.User](store, database.Buckets["users"], username)
		if err != nil {
			log.Printf("panic: user %s info not loaded: %v", username, err)
			return
		}
		classes, err := database.GetManyWithPrefix[models.Class](store, database.Buckets["classes"], helper.IntsToStrings(user.Classes...))
		if err != nil {
			log.Printf("fallback: user %s classes not loaded: %v", username, err)
			classes = []*models.Class{}
		}

		professor, err := isProfessor(store, username)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		if user.Role == "admin" {
			fmt.Println("📌 Routed to HandleAdminDefault")
			admin.HandleAdminDefault(w, r)
			return
		}

		render.RenderWithLayout(w, r, home.Home(classes, professor), body.Home)
		return

	case parts[0] == "calendar":
		professor, err := isProfessor(store, username)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		fmt.Println("📌 Routed to HandleCalendarStudentDefault")
		commonUsers.HandleCalendarStudentDefault(store, w, r, username, professor)
		return

	case isClassValid(store, username, parts[0]):
		fmt.Println("🔎 Router parts:", parts)

		if len(parts) > 1 {
			classId, err := strconv.Atoi(parts[0])
			if err != nil {
				http.Error(w, "Error with the class id", http.StatusInternalServerError)
			}

			professor, err := isProfessor(store, username)
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}

			switch parts[1] {
			case "asignaciones":

				switch len(parts) {

				case 3:
					switch parts[2] {
					case "new":
						fmt.Println("📌 Routed to NewAssignment (professor)")
						commonUsers.HandleAssignmentNew(store, storage, w, r, classId, professor)
						return

					case "delete":
						fmt.Println("📌 Routed to DeleteAssignment (professor)")
						commonUsers.HandleAssignmentDelete(store, storage, w, r, classId, professor)
						return
					}

				case 4:
					switch parts[3] {
					case "update":
						fmt.Println("📌 Routed to UpdateAssignment (professor)")
						commonUsers.HandleAssignmentUpdate(store, storage, w, r, classId, parts[2], professor)
						return

					case "submissions":
						fmt.Println("📌 Routed to HandleAssignmentSubmissions")
						commonUsers.HandleAssignmentSubmissions(store, w, r, professor)
						return

					case "details":
						fmt.Println("📌 Routed to HandleAssignmentDetail")
						commonUsers.HandleAssignmentDetail(store, w, r, classId, professor)
						return
					}

				case 5:
					if parts[3] == "submission" && parts[4] == "update" {
						fmt.Println("📌 Routed to HandleAssignmentSubmissionsUpdate")
						commonUsers.HandleSubmissionUpdate(store, storage, w, r, classId, parts[2], username, professor)
						return
					}

					if parts[3] == "submission" {
						fmt.Println("📌 Routed to HandleAssignmentSubmissions")
						commonUsers.HandleAssignmentSubmission(store, w, r, username, professor)
						return
					}

				case 6:
					if parts[3] == "submission" && parts[5] == "grade" {
						fmt.Println("📌 Routed to HandleAssignmentGrade")
						commonUsers.HandleSubmissionGrade(store, w, r, classId, parts[4], professor)
						return
					}
				}

				fmt.Println("📌 Routed to HandleAssignmentDefault")
				commonUsers.HandleAssignmentDefault(store, w, r, classId, professor, username)
				return

			case "entregas":

				fmt.Printf("📌 Routed to HandleSubmissionDefault")
				commonUsers.HandleSubmissionDefault(store, w, r, classId, professor, username)
				return

			case "recursos":
				switch len(parts) {
				case 4:
					if parts[2] == "get-asset" {
						fmt.Printf("📌 Routed to HandleGetAsset")
						generics.HandleGetAsset(store, storage, w, r, classId, parts[3])
						return
					}
				}
				fmt.Printf("📌 Routed to HandleAssetsDefault")
				generics.HandleAssetsDefault(store, w, r, classId)
				return
			}
		}
		http.NotFound(w, r)
		return

	case parts[0] == "admin":
		user, err := database.Get[models.User](store, database.Buckets["users"], username)
		if err != nil {
			log.Printf("panic: user %s info not loaded: %v", username, err)
			return
		}

		if user.Role != "admin" {
			log.Printf("panic: user %s not allowed", username)
			return
		}

		switch parts[1] {
		case "user":
			switch parts[2] {
			case "create":
				switch r.Method {
				case http.MethodGet:
					fmt.Printf("📌 Routed to HandleAdminUserCreate")
					admin.HandleAdminUserCreateDefault(w, r)
					return

				case http.MethodPost:
					fmt.Printf("📌 Routed to HandleAdminUserCreatePost")
					admin.HandleAdminUserCreatePost(store, w, r)
					return
				}

			case "modify":
				if len(parts) >= 4 {
					switch parts[3] {
					case "search":
						fmt.Printf("📌 Routed to HandleAdminUsserModifySearch")
						admin.HandleAdminUserModifySearch(store, w, r)
						return

					case "reveal-password":
						fmt.Printf("📌 Routed to HandleAdminUserRevealPassword")
						admin.HandleAdminUserRevealPassword(store, w, r)
						return

					case "update":
						fmt.Printf("📌 Routed to HandleAdminUserModifyUpdate")
						admin.HandleAdminUserModifyUpdate(store, w, r)
						return
					}
				}

				fmt.Printf("📌 Routed to HandleAdminUserModify")
				admin.HandleAdminUserModifyDefault(w, r)
				return

			case "search":
				if len(parts) >= 4 {
					if parts[3] == "look-up" {
						fmt.Printf("📌 Routed to HandleAdminUserSearchLookUp")
						admin.HandleAdminUserSearchLookUp(store, w, r)
						return
					}
				}

				fmt.Printf("📌 Routed to HandleAdminUserSearchDefault")
				admin.HandleAdminUserSearchDefault(w, r)
				return
			}

		case "class":
			switch parts[2] {
			case "create":
				switch r.Method {
				case http.MethodGet:
					fmt.Printf("📌 Routed to HandleAdminClassCreateDefault")
					admin.HandleAdminClassCreateDefault(store, w, r)
					return

				case http.MethodPost:
					fmt.Printf("📌 Routed to HandleAdminClassCreatePost")
					admin.HandleAdminClassCreatePost(store, w, r)
					return
				}

			case "modify":
				if len(parts) >= 4 {
					switch parts[3] {
					case "search":
						fmt.Printf("📌 Routed to HandleAdminClassModifySearch")
						admin.HandleAdminClassModifySearch(store, w, r)
						return

					case "update":
						fmt.Printf("📌 Routed to HandleAdminClassModifyUpdate")
						admin.HandleAdminClassModifyUpdate(store, w, r)
						return
					}
				}

				fmt.Printf("📌 Routed to HandleAdminClassModifyDefault")
				admin.HandleAdminClassModifyDefault(w, r)
				return

			case "search":
				if len(parts) >= 4 {
					if parts[3] == "look-up" {
						fmt.Printf("📌 Routed to HandleAdminClassSearchLookUp")
						admin.HandleAdminClassSearchLookUp(store, w, r)
						return
					}
				}

				fmt.Printf("📌 Routed to HandleAdminClassSearchDefault")
				admin.HandleAdminClassSearchDefault(store, w, r)
				return
			}

		case "subject":
			switch parts[2] {
			case "manage":
				if len(parts) >= 4 {
					switch parts[3] {
					case "update":
						fmt.Printf("📌 Routed to HandleAdminSubjectUpdate")
						admin.HandleAdminSubjectManagerUpdate(store, w, r)
						return
					}
				}

				fmt.Printf("📌 Routed to HandleAdminSubjectsDefault")
				admin.HandleAdminSubjectsManagerDefault(store, w, r)
				return

			case "rename":
				if len(parts) >= 4 {
					switch parts[3] {
					case "update":
						fmt.Printf("📌 Routed to HandleAdminSubjectRenameUpdate")
						admin.HandleAdminSubjectRenameUpdate(store, w, r)
						return
					}
				}

				fmt.Printf("📌 Routed to HandleAdminSubjectsDefault")
				admin.HandleAdminSubjectRenameDefault(store, w, r)
				return
			}

		case "asset":
			switch parts[2] {
			case "manage":
				if len(parts) >= 4 {
					switch parts[3] {
					case "list":
						fmt.Printf("📌 Routed to HandleAdminAssetList")
						admin.HandleAdminAssetList(store, w, r)
						return

					case "upload":
						fmt.Printf("📌 Routed to HandleAdminAssetManageUpload")
						admin.HandleAdminAssetManageUpload(store, storage, w, r)
						return

					case "delete":
						fmt.Printf("📌 Routed to HandleAdminAssetManageDelete")
						admin.HandleAdminAssetManageDelete(store, storage, w, r)
						return
					}
				}

				fmt.Printf("📌 Routed to HandleAdminSubjectsDefault")
				admin.HandleAdminAssetManagerDefault(store, w, r)
				return

			case "view":
				fmt.Printf("📌 Routed to HandleAdminAssetView")
				admin.HandleAdminAssetView(store, storage, w, r)
				return

			case "refresh":
				fmt.Printf("📌 Routed to HandleAdminAssetRefresh")
				admin.HandleAdminAssetRefresh(store, storage, w, r)
				return
			}
		}

	default:
		http.NotFound(w, r)
		return
	}
}
