package internal

import (
	"frontend/templates/body"
	"frontend/templates/components/assignment"
	"frontend/templates/components/class"
	"frontend/templates/components/home"
	"net/http"
	"strings"
)

func Router(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	switch parts[0] {
	case "login":
		RenderWithLayout(w, r, body.Auth())

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

func loadAssignments(subject string) []assignment.Assignment {
	switch strings.ToLower(subject) {
	case "matematicas":
		return []assignment.Assignment{
			{
				ID:          1,
				Title:       "Álgebra I",
				Description: "Resolver los ejercicios de la página 42 del libro de texto. Entregar en hoja cuadriculada.",
				SubjectID:   1,
				DueDate:     "2024-09-25",
			},
			{
				ID:          1,
				Title:       "Álgebra I",
				Description: "Resolver los ejercicios de la página 42 del libro de texto. Entregar en hoja cuadriculada.",
				SubjectID:   1,
				DueDate:     "2025-09-22",
			},
			{
				ID:          2,
				Title:       "Geometría básica",
				Description: "Dibujar y calcular las áreas de triángulos equiláteros y rectángulos.",
				SubjectID:   1,
				DueDate:     "2025-09-30",
			},
		}

	case "historia":
		return []assignment.Assignment{
			{
				ID:          3,
				Title:       "Independencia de Bolivia",
				Description: "Escribir un ensayo corto (1-2 páginas) sobre los líderes de la independencia.",
				SubjectID:   2,
				DueDate:     "2025-09-28",
			},
			{
				ID:          4,
				Title:       "Revolución Francesa",
				Description: "Hacer un resumen de las causas y consecuencias principales de la Revolución Francesa.",
				SubjectID:   2,
				DueDate:     "2025-10-05",
			},
		}

	case "ciencias":
		return []assignment.Assignment{
			{
				ID:          5,
				Title:       "Fotosíntesis",
				Description: "Elaborar un esquema del proceso de fotosíntesis con dibujos y explicaciones.",
				SubjectID:   3,
				DueDate:     "2025-09-27",
			},
			{
				ID:          6,
				Title:       "Ecosistemas",
				Description: "Investigar un ecosistema local y preparar una presentación corta.",
				SubjectID:   3,
				DueDate:     "2025-10-03",
			},
		}

	default:
		return []assignment.Assignment{}
	}
}
