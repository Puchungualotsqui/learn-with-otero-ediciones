package commonUsers

import (
	"fmt"
	"frontend/database/models"
	"frontend/database/sqlite"
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

func HandleCalendarStudentDefault(store *sqlite.Store, w http.ResponseWriter, r *http.Request, username string, professor bool) {
	fmt.Println("📅 [HandleCalendarStudent] Rendering student calendar")

	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	if m, err := strconv.Atoi(r.URL.Query().Get("month")); err == nil {
		month = m
	}
	if y, err := strconv.Atoi(r.URL.Query().Get("year")); err == nil {
		year = y
	}

	if month < 1 {
		month = 12
		year--
	} else if month > 12 {
		month = 1
		year++
	}

	user, err := store.GetUser(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	type AssignmentBundle struct {
		Assignment *models.Assignment
		ClassID    string
		ClassName  string
		Grade      string
	}

	bundles := make([]AssignmentBundle, 0)

	for _, classID := range user.Classes {
		assignments, err := store.ListAssignmentsOfClass(classID)
		if err != nil {
			http.Error(w, "Error fetching assignments", http.StatusInternalServerError)
			return
		}

		classModel, err := store.GetClass(classID)
		if err != nil {
			http.Error(w, "Error fetching classes", http.StatusInternalServerError)
			return
		}

		for _, a := range assignments {
			if a.DueDate == "" {
				continue
			}

			t, err := time.Parse("02/01/2006", a.DueDate)
			if err != nil {
				continue
			}

			if int(t.Month()) != month || t.Year() != year {
				continue
			}

			gradeValue := ""
			submission, err := store.GetSubmission(classID, a.Id, username)
			if err == nil && submission != nil {
				gradeValue = submission.Grade
			}

			bundles = append(bundles, AssignmentBundle{
				Assignment: a,
				ClassID:    strconv.Itoa(classID),
				ClassName:  classModel.Name,
				Grade:      gradeValue,
			})
		}
	}

	sort.Slice(bundles, func(i, j int) bool {
		t1, _ := time.Parse("02/01/2006", bundles[i].Assignment.DueDate)
		t2, _ := time.Parse("02/01/2006", bundles[j].Assignment.DueDate)
		return t1.After(t2)
	})

	finalAssignments := make([]*models.Assignment, len(bundles))
	finalClassIds := make([]string, len(bundles))
	finalClassNames := make([]string, len(bundles))
	finalGrades := make([]string, len(bundles))

	for i, b := range bundles {
		finalAssignments[i] = b.Assignment
		finalClassIds[i] = b.ClassID
		finalClassNames[i] = b.ClassName
		finalGrades[i] = b.Grade
	}

	parts := make([]templ.Component, 3)
	if professor {
		idsInts, _ := helper.StringsToInts(finalClassIds...)

		parts[0] = calendarListProfessor.CalendarListProfessor(finalAssignments, idsInts, finalClassNames)
		parts[1] = assignmentDetail.AssignmentDetail(nil, true)
		parts[2] = submissionDetail.SubmissionDetail(nil, "", "", true, true)
	} else {
		parts[0] = calendarListStudent.CalendarListStudent(finalAssignments, finalClassIds, username, finalGrades, finalClassNames)
		parts[1] = assignmentDetail.AssignmentDetail(nil, true)
		parts[2] = submissionEditor.SubmissionEditor(nil, 0, 0, "")
	}

	panels := panelsContent.PanelsContent(parts...)
	render.RenderWithLayout(
		w, r,
		calendarWrapper.CalendarWrapper(month, helper.MonthNameES(month), year, panels),
		body.Home,
	)
}
