package helper

import (
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
