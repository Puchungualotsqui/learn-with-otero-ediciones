package commonUsers

import (
	"fmt"
	"frontend/database"
	"frontend/database/models"
	"frontend/helper"
	"frontend/internal/render"
	"frontend/templates/body"
	"frontend/templates/components/assignment/assignmentDetail"
	"frontend/templates/components/assignment/submissionDetail"
	"frontend/templates/components/assignment/submissionEditor"
	"frontend/templates/components/calendarListProfessor"
	"frontend/templates/components/calendarListStudent"
	"frontend/templates/components/calendarWrapperStudent"
	"frontend/templates/components/panelsContent"
	"net/http"
	"time"

	"github.com/a-h/templ"
)

func HandleCalendarStudentDefault(store *database.Store, w http.ResponseWriter, r *http.Request, username string, professor bool) {
	fmt.Println("📅 [HandleCalendarStudent] Rendering student calendar")

	// Get current month
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	user, err := database.Get[models.User](store, database.Buckets["users"], username)
	if err != nil {
		fmt.Printf("Error getting user from database %v\n", err)
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	classesIds := helper.IntsToStrings(user.Classes...)

	var PrefixesClasses [][]string
	for _, p := range classesIds {
		PrefixesClasses = append(PrefixesClasses, []string{p})
	}

	classesAssignments, err := database.ListByManyPrefix[models.Assignment](store, database.Buckets["assignments"], 200, PrefixesClasses...)
	if err != nil {
		fmt.Printf("Error getting assignments %v\n", err)
		http.Error(w, "Error fetching assignments", http.StatusInternalServerError)
		return
	}

	assignmentsFiltered := make([]*models.Assignment, 0, len(classesAssignments))
	classIdsFiltered := make([]string, 0, len(classesAssignments))
	for i, assignments := range classesAssignments {
		for _, assignment := range assignments {
			if assignment.DueDate == "" {
				continue
			}

			t, err := time.Parse("02/01/2006", assignment.DueDate)
			if err != nil {
				continue // skip invalid dates
			}

			if int(t.Month()) == month && t.Year() == year {
				assignmentsFiltered = append(assignmentsFiltered, assignment)
				classIdsFiltered = append(classIdsFiltered, i)
			}
		}

	}

	submissionKeys := make([]string, len(assignmentsFiltered))
	for i, a := range assignmentsFiltered {
		key := fmt.Sprintf("%s:%d:%s", classIdsFiltered[i], a.Id, username)
		submissionKeys[i] = key
	}

	submissions, err := database.GetMany[models.Submission](store, database.Buckets["submissions"], submissionKeys...)
	if err != nil {
		fmt.Printf("Error getting submissions %v\n", err)
		http.Error(w, "Error fetching submissions", http.StatusInternalServerError)
		return
	}

	grades := make([]string, len(assignmentsFiltered))
	for i := range assignmentsFiltered {
		if i < len(submissions) && submissions[i] != nil {
			grades[i] = submissions[i].Grade
		} else {
			grades[i] = ""
		}
	}
	parts := make([]templ.Component, 3)
	if !professor {
		parts[0] = calendarListStudent.CalendarListStudent(assignmentsFiltered, username, grades)
		parts[1] = assignmentDetail.AssignmentDetail(nil, true)
		parts[2] = submissionEditor.SubmissionEditor(nil, 0, 0, "")
	} else {
		classIdsFilteredInts, err := helper.StringsToInts(classIdsFiltered...)
		if err != nil {
			fmt.Printf("Error converting Ids %v\n", err)
			http.Error(w, "Error converting Ids", http.StatusInternalServerError)
			return
		}
		parts[0] = calendarListProfessor.CalendarListProfessor(assignmentsFiltered, classIdsFilteredInts)
		parts[1] = assignmentDetail.AssignmentDetail(nil, true)
		parts[2] = submissionDetail.SubmissionDetail(nil, "", "", true, true)
	}

	panels := panelsContent.PanelsContent(parts...)

	render.RenderWithLayout(
		w, r,
		calendarWrapperStudent.CalendarWrapperStudent(month, year, panels),
		body.Home,
	)
}
