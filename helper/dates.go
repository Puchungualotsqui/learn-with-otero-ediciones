package helper

import (
	"fmt"
	"frontend/database/models"
	"sort"
	"time"
)

// DateStatus represents how far we are from a due date.
type DateStatus struct {
	Past     bool // true if due date already passed (after end of that day)
	DaysLeft int  // number of full days left (negative if past)
}

// GetDateStatus returns whether a date has passed and how many days remain.
func GetDateStatus(dueDate string) (DateStatus, error) {
	loc, err := time.LoadLocation("America/La_Paz")
	if err != nil {
		return DateStatus{}, err
	}

	due, err := time.ParseInLocation("02/01/2006", dueDate, loc)
	if err != nil {
		return DateStatus{}, err
	}

	now := time.Now().In(loc)

	// compare using whole days
	dueDateOnly := due.Truncate(24 * time.Hour)
	today := now.Truncate(24 * time.Hour)

	daysLeft := int(dueDateOnly.Sub(today).Hours() / 24)
	past := today.After(dueDateOnly)

	return DateStatus{Past: past, DaysLeft: daysLeft}, nil
}

// OrderAssignments sorts assignments in-place from older to newer due dates.
func OrderAssignments(assignments []*models.Assignment) ([]*models.Assignment, error) {
	loc, err := time.LoadLocation("America/La_Paz")
	if err != nil {
		return nil, fmt.Errorf("error loading location: %w", err)
	}

	layout := "02/01/2006"

	sort.Slice(assignments, func(i, j int) bool {
		dateI, errI := time.ParseInLocation(layout, assignments[i].DueDate, loc)
		dateJ, errJ := time.ParseInLocation(layout, assignments[j].DueDate, loc)

		// handle invalid dates gracefully
		if errI != nil || errJ != nil {
			return assignments[i].DueDate > assignments[j].DueDate
		}

		return dateI.After(dateJ)
	})

	return assignments, nil
}

func RemovePastAssignments(assignments []*models.Assignment) ([]*models.Assignment, error) {
	n := 0
	var status DateStatus
	var err error

	for _, a := range assignments {
		status, err = GetDateStatus(a.DueDate)
		if err != nil {
			fmt.Println("Error filtering")
		}

		// keep only if NOT past (today or future)
		if status.Past {
			assignments[n] = a
			n++
		}
	}

	// Trim the slice (no reallocation)
	return assignments[:n], nil
}
