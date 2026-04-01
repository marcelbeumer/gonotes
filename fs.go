package gonotes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var reIDPrefix = regexp.MustCompile(`^(\d{8})-(\d+)`)

const readDirBatch = 256

// NextID generates the next available note ID by scanning dir for existing
// notes with today's date prefix. The ID format is yyyymmdd-<num>.
func NextID(dir string, now time.Time) (string, error) {
	prefix := now.Format("20060102")

	f, err := os.Open(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return prefix + "-1", nil
		}
		return "", fmt.Errorf("next id: %w", err)
	}
	defer f.Close()

	maxNum := 0
	for {
		entries, err := f.ReadDir(readDirBatch)
		for _, e := range entries {
			m := reIDPrefix.FindStringSubmatch(e.Name())
			if m == nil {
				continue
			}
			if m[1] != prefix {
				continue
			}
			n, err := strconv.Atoi(m[2])
			if err != nil {
				continue
			}
			if n > maxNum {
				maxNum = n
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("next id: read dir: %w", err)
		}
	}

	return fmt.Sprintf("%s-%d", prefix, maxNum+1), nil
}

// NoteFilename returns the markdown filename for a note given its ID and slug.
// If slug is empty, the filename is just <id>.md.
func NoteFilename(id, slug string) string {
	if slug == "" {
		return id + ".md"
	}
	return id + "-" + slug + ".md"
}

// WriteNote writes the note's markdown to dir/<filename> and returns the full
// path. The directory is created if it does not exist.
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

// Link represents a symlink to be created.
type Link struct {
	Path   string // symlink path, relative to the notes base dir
	Target string // relative symlink target
}

// Plan describes the filesystem operations for a single note.
type Plan struct {
	WritePath string // file path relative to base dir (notes/by/id/<filename>)
	Links     []Link
}

// String returns a human-readable summary of the plan.
func (p *Plan) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "write: %s\n", p.WritePath)
	for _, l := range p.Links {
		fmt.Fprintf(&b, "link:  %s -> %s\n", l.Path, l.Target)
	}
	return b.String()
}

// Execute creates the symlinks described in the plan. baseDir is the absolute
// path to the notes root directory (parent of "notes/").
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

// NotePlan computes the filesystem plan for a single note (write path and
// symlinks). The note must have its ID set. Date and tags are read from the
// frontmatter.
func NotePlan(note *Note) *Plan {
	filename := NoteFilename(note.ID, note.Slug)
	writePath := filepath.Join("notes", "by", "id", filename)

	return &Plan{
		WritePath: writePath,
		Links:     linkEntries(note, filename),
	}
}

// linkEntries computes the symlink entries for a note.
func linkEntries(note *Note, filename string) []Link {
	var links []Link

	// Date symlink: notes/by/date/<yyyy-mm-dd>/<filename> -> ../../id/<filename>
	if dateStr, ok := note.Frontmatter.Get("date"); ok {
		t, err := time.Parse(dateLayout, dateStr)
		if err == nil {
			datePath := filepath.Join("notes", "by", "date", t.Format("2006-01-02"), filename)
			links = append(links, Link{
				Path:   datePath,
				Target: filepath.Join("..", "..", "id", filename),
			})
		}
	}

	// Tag symlinks.
	for _, tag := range note.Tags {
		parts := strings.Split(tag, "/")

		// Nested: notes/by/tags/nested/<a>/<b>/<c>/<filename>
		nestedPath := filepath.Join(append([]string{"notes", "by", "tags", "nested"}, append(parts, filename)...)...)
		// From symlink dir back to notes/by/: up through each tag part + nested + tags = 2 + len(parts).
		nestedUp := make([]string, 2+len(parts))
		for i := range nestedUp {
			nestedUp[i] = ".."
		}
		nestedTarget := filepath.Join(append(nestedUp, "id", filename)...)
		links = append(links, Link{
			Path:   nestedPath,
			Target: nestedTarget,
		})

		// Flat: one entry per path component.
		// notes/by/tags/flat/<component>/<filename>
		// From symlink dir back to notes/by/: up through <component> + flat + tags = 3.
		flatTarget := filepath.Join("..", "..", "..", "id", filename)
		for _, part := range parts {
			flatPath := filepath.Join("notes", "by", "tags", "flat", part, filename)
			links = append(links, Link{
				Path:   flatPath,
				Target: flatTarget,
			})
		}
	}

	return links
}

// CreateNote is the high-level entry point for creating a new note on disk.
// It prepares the note, generates an ID if needed, writes the file, and creates
// symlinks. Returns the path of the created note file.
//
// If id is empty, a new ID is generated by scanning baseDir/notes/by/id/.
// If dryRun is true, no files or symlinks are created; the plan is returned
// along with the prepared note.
func CreateNote(baseDir string, r io.Reader, opts PrepareOptions, id string, dryRun bool) (*Note, *Plan, error) {
	note, err := Prepare(r, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}

	if id == "" {
		now := time.Now
		if opts.Now != nil {
			now = opts.Now
		}
		idDir := filepath.Join(baseDir, "notes", "by", "id")
		id, err = NextID(idDir, now())
		if err != nil {
			return nil, nil, fmt.Errorf("create note: %w", err)
		}
	}
	note.ID = id

	plan := NotePlan(note)

	if dryRun {
		return note, plan, nil
	}

	writePath := filepath.Join(baseDir, plan.WritePath)
	writeDir := filepath.Dir(writePath)
	if err := os.MkdirAll(writeDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}
	if err := os.WriteFile(writePath, []byte(note.Markdown()), 0o644); err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}

	if err := plan.Execute(baseDir); err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}

	return note, plan, nil
}
