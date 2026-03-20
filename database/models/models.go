package models

type User struct {
	Username          string `json:"username"`
	PasswordHashed    string `json:"password_hashed"`
	PasswordNotHashed string `json:"password_now_hashed"` // type, to avoid migration keep it
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	School            string `json:"school"`
	Grade             string `json:"grade"`
	Role              string `json:"role"`
	PhoneNumber       string `json:"phone_number"`
	Email             string `json:"email"`
	Classes           []int  `json:"classes"`
}

type Subject struct {
	Name string `json:"name"`
}

type Class struct {
	Id          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Grade       string   `json:"grade"`
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

type Asset struct {
	Name                string `json:"name"`
	OriginalName        string `json:"original_name"`
	Url                 string `json:"url"`
	StudentVisibility   bool   `json:"student_visibility"`
	ProfessorVisibility bool   `json:"professor_visibility"`
}
