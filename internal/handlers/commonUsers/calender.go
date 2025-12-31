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

	// --- 1. Date Parsing ---
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	if m, err := strconv.Atoi(r.URL.Query().Get("month")); err == nil {
		month = m
	}
	if y, err := strconv.Atoi(r.URL.Query().Get("year")); err == nil {
		year = y
	}

	// Month wrap-around logic
	if month < 1 {
		month = 12
		year--
	} else if month > 12 {
		month = 1
		year++
	}

	// --- 2. Get User & Class List ---
	user, err := database.Get[models.User](store, database.Buckets["users"], username)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	classesIds := helper.IntsToStrings(user.Classes...)
	var prefixes [][]string
	for _, p := range classesIds {
		prefixes = append(prefixes, []string{p})
	}

	// --- 3. Fetch Assignments (Map: ClassID -> []*Assignment) ---
	classesAssignments, err := database.ListByManyPrefix[models.Assignment](store, database.Buckets["assignments"], 200, prefixes...)
	if err != nil {
		http.Error(w, "Error fetching assignments", http.StatusInternalServerError)
		return
	}

	// --- 4. Create Bundles (The Single Source of Truth) ---
	type AssignmentBundle struct {
		Assignment *models.Assignment
		ClassID    string
		ClassName  string // Filled later
		Grade      string // Filled later
	}
	var bundles []AssignmentBundle

	// Flatten the map into our bundle slice immediately
	for classId, assignments := range classesAssignments {
		for _, a := range assignments {
			if a.DueDate == "" {
				continue
			}
			t, err := time.Parse("02/01/2006", a.DueDate)
			if err != nil {
				continue
			}

			// Filter by Month/Year
			if int(t.Month()) == month && t.Year() == year {
				bundles = append(bundles, AssignmentBundle{
					Assignment: a,
					ClassID:    classId, // LOCKING THE ID HERE
				})
			}
		}
	}

	// --- 5. Fetch Class Names (Metadata) ---
	// Collect unique Class IDs from bundles
	idSet := make(map[string]bool)
	for _, b := range bundles {
		idSet[b.ClassID] = true
	}
	uniqueIds := make([]string, 0, len(idSet))
	for id := range idSet {
		uniqueIds = append(uniqueIds, id)
	}

	classesList, err := database.GetMany[models.Class](store, database.Buckets["classes"], uniqueIds...)
	if err != nil {
		http.Error(w, "Error fetching classes", http.StatusInternalServerError)
		return
	}

	// Create Lookup Map: ClassID -> ClassName
	classLookup := make(map[string]string)
	for _, c := range classesList {
		classLookup[strconv.Itoa(c.Id)] = c.Name
	}

	// --- 6. Fetch Submissions (Grades) ---
	// Only needed if we want to show grades (Students usually)
	// Key Format: username:classId:assignmentId
	submissionKeys := make([]string, len(bundles))
	for i, b := range bundles {
		submissionKeys[i] = fmt.Sprintf("%s:%s:%d", username, b.ClassID, b.Assignment.Id)
	}

	submissions, err := database.GetMany[models.Submission](store, database.Buckets["submissions"], submissionKeys...)

	// Create Lookup Map: Key -> Grade (Safeguard against index misalignment)
	gradeLookup := make(map[string]string)
	if err == nil {
		for i, sub := range submissions {
			if sub != nil {
				// We use the same key we generated to store the lookup
				gradeLookup[submissionKeys[i]] = sub.Grade
			}
		}
	}

	// --- 7. Enrich and Sort Bundles ---
	for i := range bundles {
		// Populate Name
		if name, ok := classLookup[bundles[i].ClassID]; ok {
			bundles[i].ClassName = name
		} else {
			bundles[i].ClassName = "Clase Desconocida"
		}

		// Populate Grade
		// Re-generate the key to find it in the map
		key := fmt.Sprintf("%s:%s:%d", username, bundles[i].ClassID, bundles[i].Assignment.Id)
		if g, ok := gradeLookup[key]; ok {
			bundles[i].Grade = g
		}
	}

	// Sort by Due Date (Newest first)
	sort.Slice(bundles, func(i, j int) bool {
		t1, _ := time.Parse("02/01/2006", bundles[i].Assignment.DueDate)
		t2, _ := time.Parse("02/01/2006", bundles[j].Assignment.DueDate)
		return t1.After(t2)
	})

	// --- 8. Unpack for Templates ---
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

	// --- 9. Render ---
	parts := make([]templ.Component, 3)
	if professor {
		// Convert String IDs to Ints for the Professor Component
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
