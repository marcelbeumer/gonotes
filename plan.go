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
	Links []Link
}

func (p *Plan) String() string {
	var b strings.Builder
	for _, l := range p.Links {
		fmt.Fprintf(&b, "link:  %s -> %s\n", l.Path, l.Target)
	}
	return b.String()
}

func (p *Plan) CreateLinks(baseDir string) error {
	for _, l := range p.Links {
		abs := filepath.Join(baseDir, l.Path)
		dir := filepath.Dir(abs)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create links: mkdir %s: %w", dir, err)
		}
		if err := os.Symlink(l.Target, abs); err != nil {
			return fmt.Errorf("create links: symlink %s: %w", l.Path, err)
		}
	}
	return nil
}

func NotePlan(note *Note) *Plan {
	filename := NoteFilename(note.ID, note.Slug)
	return &Plan{
		Links: linkEntries(note, filename),
	}
}

func linkEntries(note *Note, filename string) []Link {
	var links []Link
	seen := map[string]struct{}{}

	if dateStr, ok := note.Frontmatter.Get("date"); ok {
		t, err := time.Parse(dateLayout, dateStr)
		if err == nil {
			datePath := filepath.Join("notes", "by", "date", t.Format("2006-01-02"), filename)
			seen[datePath] = struct{}{}
			links = append(links, Link{
				Path:   datePath,
				Target: filepath.Join("..", "..", "id", filename),
			})
		}
	}

	for _, tag := range note.Tags {
		parts := strings.Split(tag, "/")
		nestedPath := filepath.Join(append([]string{"notes", "by", "tags"}, append(parts, filename)...)...)
		if _, ok := seen[nestedPath]; !ok {
			nestedUp := make([]string, 1+len(parts))
			for i := range nestedUp {
				nestedUp[i] = ".."
			}
			nestedTarget := filepath.Join(append(nestedUp, "id", filename)...)
			seen[nestedPath] = struct{}{}
			links = append(links, Link{
				Path:   nestedPath,
				Target: nestedTarget,
			})
		}
	}

	return links
}
