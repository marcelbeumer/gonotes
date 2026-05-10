package gonotes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Link struct {
	Path   string
	Target string
}

type Plan struct {
	WritePath string
	Links     []Link
}

func (p *Plan) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "write: %s\n", p.WritePath)
	for _, l := range p.Links {
		fmt.Fprintf(&b, "link:  %s -> %s\n", l.Path, l.Target)
	}
	return b.String()
}

func (p *Plan) Execute(baseDir string) error {
	for _, l := range p.Links {
		abs := filepath.Join(baseDir, l.Path)
		dir := filepath.Dir(abs)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("plan execute: mkdir %s: %w", dir, err)
		}
		if err := os.Symlink(l.Target, abs); err != nil {
			return fmt.Errorf("plan execute: symlink %s: %w", l.Path, err)
		}
	}
	return nil
}

func NotePlan(note *Note) *Plan {
	filename := NoteFilename(note.ID, note.Slug)
	writePath := filepath.Join("notes", "by", "id", filename)

	return &Plan{
		WritePath: writePath,
		Links:     linkEntries(note, filename),
	}
}

func linkEntries(note *Note, filename string) []Link {
	var links []Link
	seen := map[string]bool{}

	if dateStr, ok := note.Frontmatter.Get("date"); ok {
		t, err := time.Parse(dateLayout, dateStr)
		if err == nil {
			datePath := filepath.Join("notes", "by", "date", t.Format("2006-01-02"), filename)
			seen[datePath] = true
			links = append(links, Link{
				Path:   datePath,
				Target: filepath.Join("..", "..", "id", filename),
			})
		}
	}

	for _, tag := range note.Tags {
		parts := strings.Split(tag, "/")

		nestedPath := filepath.Join(append([]string{"notes", "by", "tags"}, append(parts, filename)...)...)
		if !seen[nestedPath] {
			nestedUp := make([]string, 1+len(parts))
			for i := range nestedUp {
				nestedUp[i] = ".."
			}
			nestedTarget := filepath.Join(append(nestedUp, "id", filename)...)
			seen[nestedPath] = true
			links = append(links, Link{
				Path:   nestedPath,
				Target: nestedTarget,
			})
		}
	}

	return links
}

func WriteNote(dir string, note *Note) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("write note: %w", err)
	}

	filename := NoteFilename(note.ID, note.Slug)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, []byte(note.Markdown()), 0o644); err != nil {
		return "", fmt.Errorf("write note: %w", err)
	}

	return path, nil
}
