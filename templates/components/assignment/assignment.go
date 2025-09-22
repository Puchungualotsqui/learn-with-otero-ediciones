package assignment

type Assignment struct {
	ID          int
	Title       string
	Description string
	SubjectID   int    // Foreign key, links back to Subject
	DueDate     string // Formatted as "2025-09-30"
}

type Subject struct {
	ID          int
	Name        string
	Level       string // e.g. "Primaria", "Secundaria"
	Assignments []Assignment
}
