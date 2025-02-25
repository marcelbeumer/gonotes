package gonotes

import "time"

type Meta struct {
	Title    string
	Href     string
	Date     time.Time
	Modified time.Time
}

type Note struct {
	Meta Meta
	Body string
}
