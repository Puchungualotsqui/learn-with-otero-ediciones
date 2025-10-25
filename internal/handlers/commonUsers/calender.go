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
	"frontend/templates/components/calendarWrapper"
	"frontend/templates/components/panelsContent"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/a-h/templ"
)

func HandleCalendarStudentDefault(store *database.Store, w http.ResponseWriter, r *http.Request, username string, professor bool) {
	fmt.Println("📅 [HandleCalendarStudent] Rendering student calendar")

	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	// Provide defaults if missing
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil {
			month = m
		}
	}
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	// Optional: handle wrap-around (e.g., month = 0 → previous year)
	if month < 1 {
		month = 12
		year--
	} else if month > 12 {
		month = 1
		year++
	}

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

	classes, err := database.GetMany[models.Class](store, database.Buckets["classes"], classIdsFiltered...)
	if err != nil {
		fmt.Printf("Error getting classes %v\n", err)
		http.Error(w, "Error fetching classes", http.StatusInternalServerError)
		return
	}

	classNames := make([]string, len(classIdsFiltered))
	for i, class := range classes {
		classNames[i] = class.Name
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

	// Bundle all related data before sorting to keep them aligned
	type AssignmentBundle struct {
		Assignment *models.Assignment
		ClassID    string
		ClassName  string
		Grade      string
	}

	bundles := make([]AssignmentBundle, len(assignmentsFiltered))
	for i := range assignmentsFiltered {
		bundles[i] = AssignmentBundle{
			Assignment: assignmentsFiltered[i],
			ClassID:    classIdsFiltered[i],
			ClassName:  classNames[i],
			Grade:      grades[i],
		}
	}

	// Sort by date (newest first)
	sort.Slice(bundles, func(i, j int) bool {
		t1, err1 := time.Parse("02/01/2006", bundles[i].Assignment.DueDate)
		t2, err2 := time.Parse("02/01/2006", bundles[j].Assignment.DueDate)
		if err1 != nil || err2 != nil {
			return false
		}
		return t1.After(t2)
	})

	// Unpack back into your slices
	for i := range bundles {
		assignmentsFiltered[i] = bundles[i].Assignment
		classIdsFiltered[i] = bundles[i].ClassID
		classNames[i] = bundles[i].ClassName
		grades[i] = bundles[i].Grade
	}

	parts := make([]templ.Component, 3)
	if professor {
		classIdsFilteredInts, err := helper.StringsToInts(classIdsFiltered...)
		if err != nil {
			fmt.Printf("Error converting Ids %v\n", err)
			http.Error(w, "Error converting Ids", http.StatusInternalServerError)
			return
		}
		parts[0] = calendarListProfessor.CalendarListProfessor(assignmentsFiltered, classIdsFilteredInts, classNames)
		parts[1] = assignmentDetail.AssignmentDetail(nil, true)
		parts[2] = submissionDetail.SubmissionDetail(nil, "", "", true, true)
	} else {
		parts[0] = calendarListStudent.CalendarListStudent(assignmentsFiltered, username, grades, classNames)
		parts[1] = assignmentDetail.AssignmentDetail(nil, true)
		parts[2] = submissionEditor.SubmissionEditor(nil, 0, 0, "")
	}

	panels := panelsContent.PanelsContent(parts...)

	monthName := helper.MonthNameES(month)
	render.RenderWithLayout(
		w, r,
		calendarWrapper.CalendarWrapper(month, monthName, year, panels),
		body.Home,
	)
}
