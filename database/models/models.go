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
	Name string `json:"name"`
}

type Class struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	Subject    string `json:"subject"`
	StudentIds []int  `json:"student_ids"`
	TeacherIds []int  `json:"teacher_ids"`
}

type Assignment struct {
	Id          int    `json:"id"`
	ClassId     int    `json:"class_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"` // formatted "2006-01-02"
}

type Submission struct {
	Id          int    `json:"id"`
	StudentId   int    `json:"student_id"`
	Content     string `json:"content"`      // could be file path or text
	SubmittedAt string `json:"submitted_at"` // timestamp
	Grade       string `json:"grade,omitempty"`
}
