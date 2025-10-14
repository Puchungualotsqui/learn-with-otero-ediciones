package dto

type AdminOption struct {
	Title       string
	Description string
	SubUrl      string
	SubOptions  []*AdminSubOptionSlot
}

type AdminSubOptionSlot struct {
	Title string
	Url   string
}
