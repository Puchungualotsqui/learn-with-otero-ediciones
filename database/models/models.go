package models

type User struct {
	Username          string `json:"username"`
	PasswordHashed    string `json:"password_hashed"`
	PasswordNotHashed string `json:"password_now_hashed"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Role              string `json:"role"`
}

type Subject struct {
	InternalName string `json:"internal_name"`
	Name         string `json:"name"`
}

type Class struct {
	Id          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Subject     string   `json:"subject"`
	Users       []string `json:"users"`
}

type Assignment struct {
	Id          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Content     []string `json:"content"`  // url to some file
	DueDate     string   `json:"due_date"` // formatted "30/09/2025"
}

type Submission struct {
	Username    string   `json:"username"`
	Description string   `json:"description"`
	Content     []string `json:"content"`      // could be file path or text
	SubmittedAt string   `json:"submitted_at"` // timestamp
	Grade       string   `json:"grade,omitempty"`
}
