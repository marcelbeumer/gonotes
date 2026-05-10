package gonotes

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var reWikiLink = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

const frontmatterSep = "---"

// dateLayout is the Go reference time format used for note dates.
const dateLayout = "2006-01-02 15:04:05"

// Note represents a single markdown note. The Frontmatter field is the
// source of truth; the other fields are derived from it by deriveFields.
type Note struct {
	Frontmatter   *Frontmatter
	ID            string
	Date          time.Time
	Title         string
	Slug          string
	Tags          []string
	Body          string
	InternalLinks []string
	IgnoreLinks   []string
}

func NewNote() *Note {
	return &Note{
		Frontmatter: NewFrontmatter(),
	}
}

// ReadNote parses a note from r. The id is set on the returned Note but is
// not expected to come from the file content itself.
func ReadNote(id string, r io.Reader) (*Note, error) {
	fm, body, err := splitFrontmatterBody(r)
	if err != nil {
		return nil, fmt.Errorf("read note: %w", err)
	}

	note := &Note{
		Frontmatter: NewFrontmatter(),
		ID:          id,
		Body:        body,
	}

	if fm != "" {
		if err := yaml.Unmarshal([]byte(fm), note.Frontmatter); err != nil {
			return nil, fmt.Errorf("read note: unmarshal frontmatter: %w", err)
		}
	}

	note.deriveFields()
	return note, nil
}

// deriveFields populates the computed fields (Title, Slug, Tags, Date,
// InternalLinks, IgnoreLinks) from Frontmatter and Body.
func (n *Note) deriveFields() {
	if title, ok := n.Frontmatter.Get("title"); ok {
		n.Title = title
		n.Slug = slugify(title)
	} else {
		n.Title = ""
		n.Slug = ""
	}

	if tags, ok := n.Frontmatter.Get("tags"); ok {
		n.Tags = ParseTags(tags)
	} else {
		n.Tags = nil
	}

	if dateStr, ok := n.Frontmatter.Get("date"); ok {
		t, err := time.Parse(dateLayout, dateStr)
		if err != nil {
			n.Date = time.Time{}
		} else {
			n.Date = t
		}
	} else {
		n.Date = time.Time{}
	}

	if ignoreLinks, ok := n.Frontmatter.Get("ignore-links"); ok {
		n.IgnoreLinks = ParseTags(ignoreLinks)
	} else {
		n.IgnoreLinks = nil
	}

	n.InternalLinks = parseInternalLinks(n.Body)
}

func splitFrontmatterBody(r io.Reader) (fm string, body string, err error) {
	scanner := bufio.NewScanner(r)

	var fmLines []string
	var bodyLines []string
	sepCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		if sepCount < 2 && line == frontmatterSep {
			sepCount++
			continue
		}

		switch {
		case sepCount == 1:
			fmLines = append(fmLines, line)
		default:
			bodyLines = append(bodyLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", err
	}

	fm = strings.Join(fmLines, "\n")
	body = strings.Join(bodyLines, "\n")
	return fm, body, nil
}

// ParseTags splits a tag string by commas and/or spaces, trims whitespace,
// and deduplicates. Returns nil for empty input.
func ParseTags(s string) []string {
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil
	}
	return dedupStrings(parts)
}

func dedupStrings(ss []string) []string {
	if len(ss) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func parseInternalLinks(body string) []string {
	matches := reWikiLink.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	links := make([]string, len(matches))
	for i, m := range matches {
		links[i] = m[1]
	}
	return links
}

func (n *Note) Markdown() string {
	var b strings.Builder

	if len(n.Frontmatter.mappingNode().Content) > 0 {
		fmBytes, err := yaml.Marshal(n.Frontmatter)
		if err == nil {
			b.WriteString(frontmatterSep)
			b.WriteByte('\n')
			b.Write(fmBytes)
			b.WriteString(frontmatterSep)
			b.WriteByte('\n')
		}
	}

	if n.Body != "" {
		b.WriteString(n.Body)
	}

	return b.String()
}

type PrepareOptions struct {
	Title            *string
	Tags             []string
	ExtraFrontmatter []FrontmatterField
	Now              func() time.Time
}

type FrontmatterField struct {
	Key   string
	Value string
}

// Prepare reads an existing note from r (or creates a blank one if r is nil),
// applies the options, and returns the prepared note. Options mutate the
// frontmatter; derived fields are populated once at the end via deriveFields.
func Prepare(r io.Reader, opts PrepareOptions) (*Note, error) {
	var note *Note
	var err error

	if r != nil {
		note, err = ReadNote("", r)
		if err != nil {
			return nil, fmt.Errorf("prepare: %w", err)
		}
	} else {
		note = NewNote()
	}

	if opts.Title != nil {
		note.Frontmatter.Set("title", *opts.Title)
	}

	if len(opts.Tags) > 0 {
		existing, _ := note.Frontmatter.Get("tags")
		merged := append(ParseTags(existing), opts.Tags...)
		merged = dedupStrings(merged)
		if len(merged) == 0 {
			note.Frontmatter.Unset("tags")
		} else {
			note.Frontmatter.Set("tags", FormatTags(merged))
		}
	}

	if _, ok := note.Frontmatter.Get("date"); !ok {
		now := time.Now
		if opts.Now != nil {
			now = opts.Now
		}
		note.Frontmatter.Set("date", now().Local().Format(dateLayout))
	}

	for _, f := range opts.ExtraFrontmatter {
		note.Frontmatter.Set(f.Key, f.Value)
	}

	note.deriveFields()
	return note, nil
}

func FormatTags(tags []string) string {
	return strings.Join(tags, ", ")
}

func StringPtr(s string) *string {
	return &s
}
