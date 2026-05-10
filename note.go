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

const dateLayout = "2006-01-02 15:04:05"

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

	if title, ok := note.Frontmatter.Get("title"); ok {
		note.Title = title
		note.Slug = slugify(title)
	}

	if tags, ok := note.Frontmatter.Get("tags"); ok {
		note.Tags = ParseTags(tags)
	}

	if dateStr, ok := note.Frontmatter.Get("date"); ok {
		t, err := time.Parse(dateLayout, dateStr)
		if err != nil {
			return note, fmt.Errorf("parse date %q: %w", dateStr, err)
		}
		note.Date = t
	}

	if ignoreLinks, ok := note.Frontmatter.Get("ignore-links"); ok {
		note.IgnoreLinks = ParseTags(ignoreLinks)
	}

	note.InternalLinks = parseInternalLinks(body)

	return note, nil
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

func hasFrontmatterKeys(f *Frontmatter) bool {
	mn := f.mappingNode()
	return len(mn.Content) > 0
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

	if title, ok := note.Frontmatter.Get("title"); ok {
		note.Title = title
		note.Slug = slugify(title)
	} else {
		note.Title = ""
		note.Slug = ""
	}

	if tags, ok := note.Frontmatter.Get("tags"); ok {
		note.Tags = ParseTags(tags)
	} else {
		note.Tags = nil
	}

	for _, f := range opts.ExtraFrontmatter {
		note.Frontmatter.Set(f.Key, f.Value)
	}

	return note, nil
}

func mergeTags(existing, extra []string) []string {
	extraSet := make(map[string]struct{}, len(extra))
	for _, t := range extra {
		extraSet[t] = struct{}{}
	}
	var result []string
	seen := make(map[string]struct{})
	for _, t := range existing {
		if _, ok := extraSet[t]; ok {
			if _, ok := seen[t]; !ok {
				result = append(result, t)
				seen[t] = struct{}{}
			}
		}
	}
	for _, t := range extra {
		if _, ok := seen[t]; !ok {
			result = append(result, t)
			seen[t] = struct{}{}
		}
	}
	return result
}

func FormatTags(tags []string) string {
	return strings.Join(tags, ", ")
}

func StringPtr(s string) *string {
	return &s
}
