package gonotes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var reWikiLink = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

const frontmatterSep = "---"

type Note struct {
	Frontmatter   *Frontmatter
	ID            string
	Date          time.Time // is zero if none
	Title         string
	Slug          string
	Tags          []string
	Body          string
	InternalLinks []string
	IgnoreLinks   []string // glob patterns from ignore-links frontmatter
}

func NewNote() *Note {
	return &Note{
		Frontmatter: NewFrontmatter(),
	}
}

// ReadNote parses a markdown note with optional YAML frontmatter from r.
// The id is set directly from the parameter and is not parsed from content.
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

	// Parse frontmatter if present.
	if fm != "" {
		if err := yaml.Unmarshal([]byte(fm), note.Frontmatter); err != nil {
			return nil, fmt.Errorf("read note: unmarshal frontmatter: %w", err)
		}
	}

	// Extract recognized fields.
	if title, ok := note.Frontmatter.Get("title"); ok {
		note.Title = title
		note.Slug = slugify(title)
	}

	if tags, ok := note.Frontmatter.Get("tags"); ok {
		note.Tags = parseTags(tags)
	}

	if dateStr, ok := note.Frontmatter.Get("date"); ok {
		t, err := time.Parse(dateLayout, dateStr)
		if err != nil {
			return note, fmt.Errorf("parse date %q: %w", dateStr, err)
		}
		note.Date = t
	}

	if ignoreLinks, ok := note.Frontmatter.Get("ignore-links"); ok {
		note.IgnoreLinks = parseTags(ignoreLinks)
	}

	note.InternalLinks = parseInternalLinks(body)

	return note, nil
}

// splitFrontmatterBody reads from r and separates the YAML frontmatter from
// the body. The frontmatter delimiters (---) are not included in either part.
// If there is no opening delimiter on the first line, everything is body.
func splitFrontmatterBody(r io.Reader) (fm string, body string, err error) {
	scanner := bufio.NewScanner(r)

	var fmLines []string
	var bodyLines []string
	sepCount := 0
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if sepCount < 2 && line == frontmatterSep {
			sepCount++
			continue
		}

		switch {
		case sepCount == 1:
			// Between first and second ---, this is frontmatter.
			fmLines = append(fmLines, line)
		default:
			// Before first --- or after second ---, this is body.
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

// parseTags splits a comma-separated tag string into individual tags.
// Each tag is trimmed of whitespace. Empty tags are dropped.
func parseTags(s string) []string {
	parts := strings.Split(s, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// parseInternalLinks extracts all [[target]] wiki-link targets from body.
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

// Markdown serializes the note back to markdown with YAML frontmatter.
// If there is no frontmatter (zero keys), only the body is returned.
func (n *Note) Markdown() string {
	var b strings.Builder

	if hasFrontmatterKeys(n.Frontmatter) {
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

// hasFrontmatterKeys reports whether the frontmatter has any key-value pairs.
func hasFrontmatterKeys(f *Frontmatter) bool {
	mn := f.mappingNode()
	return len(mn.Content) > 0
}

// noteJSON is the JSON-serializable representation of a Note.
type noteJSON struct {
	ID            string            `json:"id,omitempty"`
	Title         string            `json:"title,omitempty"`
	Slug          string            `json:"slug,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Body          string            `json:"body,omitempty"`
	InternalLinks []string          `json:"internalLinks,omitempty"`
	Frontmatter   map[string]string `json:"frontmatter,omitempty"`
}

// JSON returns a pretty-printed JSON representation of the note.
func (n *Note) JSON() ([]byte, error) {
	v := noteJSON{
		ID:            n.ID,
		Title:         n.Title,
		Slug:          n.Slug,
		Tags:          n.Tags,
		Body:          n.Body,
		InternalLinks: n.InternalLinks,
		Frontmatter:   n.Frontmatter.Map(),
	}
	return json.MarshalIndent(v, "", "  ")
}
