package gonotes

import (
	"io"
)

type Note struct {
	Frontmatter   *Frontmatter
	ID            string
	Title         string
	Slug          string
	Tags          []string
	Body          string
	InternalLinks []string
}

func NewNote() *Note {
	return &Note{
		Frontmatter: NewFrontmatter(),
	}
}

func ReadNote(id string, r io.Reader) *Note {
	// Read line by line
	// If first line is document sep, ignore it
	// Read frontmatter string until document sep
	// If document separator, read rest as body
	// Parse first document as frontmatter
	// Take title from frontmatter
	// Generate slug from title
	// Get tags from frontmatter
	// Set body
	// Set id from func param
	return nil
}

func (n *Note) Markdown() string {
	return ""
}

// var loc *time.Location
//
// func init() {
// 	var err error
// 	loc, err = time.LoadLocation("Europe/Amsterdam")
// 	if err != nil {
// 		log.Fatalf("Load location: %s", err)
// 	}
// }

// const dateLayout = "2006-01-02 15:04:05"

// func (f *Frontmatter) Title() (string, bool) {
// 	return f.Value("title")
// }
//
// func (f *Frontmatter) SetTitle(v string) {
// 	f.SetValue("title", v)
// }
//
// func (f *Frontmatter) Date() (time.Time, bool) {
// 	str, ok := f.Value("date")
// 	if !ok {
// 		return time.Time{}, false
// 	}
//
// 	t, err := time.ParseInLocation(dateLayout, str, loc)
// 	if err != nil {
// 		return time.Time{}, true
// 	}
//
// 	return t, true
// }
//
// func (f *Frontmatter) SetDate(t time.Time) {
// 	if t.IsZero() {
// 		f.SetValue("date", "")
// 		return
// 	}
// 	f.SetValue("date", t.Format(dateLayout))
// }
//
// func (f *Frontmatter) Tags() ([]string, bool) {
// 	str, ok := f.Value("tags")
// 	if !ok {
// 		return nil, false
// 	}
//
// 	str = strings.ToLower(str)
// 	tags := strings.Split(str, ",")
//
// 	existing := map[string]struct{}{}
// 	for i, v := range tags {
// 		slug := Slugify(v)
// 		if _, ok := existing[slug]; !ok {
// 			tags[i] = slug
// 			existing[slug] = struct{}{}
// 		}
// 	}
//
// 	return tags, true
// }
//
// func (f *Frontmatter) SetTags(tags []string) {
// 	normalized := make([]string, len(tags))
// 	existing := map[string]struct{}{}
//
// 	for i, v := range tags {
// 		slug := Slugify(v)
// 		if slug == "" {
// 			continue
// 		}
// 		if _, ok := existing[slug]; !ok {
// 			normalized[i] = slug
// 			existing[slug] = struct{}{}
// 		}
// 	}
//
// 	f.SetValue("tags", strings.Join(normalized, ", "))
// }
