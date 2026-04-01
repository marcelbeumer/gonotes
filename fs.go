package gonotes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	// reIDPrefix matches the new-format ID for NextID generation: yyyymmdd-num.
	reIDPrefix = regexp.MustCompile(`^(\d{8})-(\d+)`)
	// reNoteID extracts a structured numeric ID from a filename: one or more
	// groups of digits separated by dashes (e.g. "20260328-1", "2026-02-12-2233-05").
	reNoteID = regexp.MustCompile(`^([0-9]+(?:-[0-9]+)*)`)
)

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

// IDFromFilename extracts the note ID from a filename.
// It returns the ID and whether a structured numeric ID was found (parsed).
//
// For filenames starting with digits-and-dashes (e.g. "20260328-1-hello.md",
// "2026-02-12-2233-05-pacman.md"), it extracts the numeric prefix as the ID
// and parsed is true.
//
// For other .md filenames (e.g. "readme.md"), it uses the full stem (without
// .md) as the ID and parsed is false.
//
// For non-.md filenames, it returns empty string and false.
func IDFromFilename(name string) (id string, parsed bool) {
	if !strings.HasSuffix(name, ".md") {
		return "", false
	}

	stem := strings.TrimSuffix(name, ".md")

	m := reNoteID.FindString(stem)
	if m != "" {
		return m, true
	}

	// No numeric ID prefix; use the full stem.
	return stem, false
}

// BrokenLink records an internal link that references a non-existent note.
type BrokenLink struct {
	SourceID string
	TargetID string
}

// Rename records a file that should be renamed to match its title slug.
type Rename struct {
	OldName string // current filename
	NewName string // correct filename based on id + slug
}

// RebuildReport holds the results of scanning notes/by/id/.
type RebuildReport struct {
	BrokenLinks []BrokenLink
	Renames     []Rename
}

// String returns a human-readable summary of the report.
func (r *RebuildReport) String() string {
	var b strings.Builder

	if len(r.BrokenLinks) > 0 {
		fmt.Fprintf(&b, "Broken links (%d):\n", len(r.BrokenLinks))
		for _, bl := range r.BrokenLinks {
			fmt.Fprintf(&b, "  %s -> %s\n", bl.SourceID, bl.TargetID)
		}
	}

	if len(r.Renames) > 0 {
		fmt.Fprintf(&b, "Renames (%d):\n", len(r.Renames))
		for _, rn := range r.Renames {
			fmt.Fprintf(&b, "  %s -> %s\n", rn.OldName, rn.NewName)
		}
	}

	if len(r.BrokenLinks) == 0 && len(r.Renames) == 0 {
		b.WriteString("No issues found.\n")
	}

	return b.String()
}

// ScanNotes reads notes/by/id/ one file at a time and produces a report of
// broken internal links and filenames that need renaming.
func ScanNotes(idDir string) (*RebuildReport, error) {
	f, err := os.Open(idDir)
	if err != nil {
		return nil, fmt.Errorf("scan notes: %w", err)
	}
	defer f.Close()

	// First pass: collect all IDs, current filenames, correct filenames, and links.
	type noteInfo struct {
		id            string
		currentName   string
		correctName   string
		internalLinks []string
	}

	var notes []noteInfo

	for {
		entries, err := f.ReadDir(readDirBatch)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}

			id, parsed := IDFromFilename(name)

			path := filepath.Join(idDir, name)
			nf, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("scan notes: open %s: %w", name, err)
			}

			note, err := ReadNote(id, nf)
			nf.Close()
			if err != nil {
				return nil, fmt.Errorf("scan notes: read %s: %w", name, err)
			}

			// Only compute a correct name when we have a parsed ID.
			// Otherwise keep the current name (no rename).
			correctName := name
			if parsed {
				correctName = NoteFilename(id, note.Slug)
			}

			notes = append(notes, noteInfo{
				id:            id,
				currentName:   name,
				correctName:   correctName,
				internalLinks: note.InternalLinks,
			})
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("scan notes: read dir: %w", err)
		}
	}

	// Build set of known IDs.
	idSet := make(map[string]struct{}, len(notes))
	for _, n := range notes {
		idSet[n.id] = struct{}{}
	}

	// Sort for deterministic output.
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].id < notes[j].id
	})

	report := &RebuildReport{}

	for _, n := range notes {
		// Check for broken links.
		for _, target := range n.internalLinks {
			// Try to extract a numeric ID from the link target, then
			// fall back to the raw target string.
			targetID := target
			if m := reNoteID.FindString(target); m != "" {
				targetID = m
			}
			if _, exists := idSet[targetID]; !exists {
				report.BrokenLinks = append(report.BrokenLinks, BrokenLink{
					SourceID: n.id,
					TargetID: targetID,
				})
			}
		}

		// Check for renames.
		if n.currentName != n.correctName {
			report.Renames = append(report.Renames, Rename{
				OldName: n.currentName,
				NewName: n.correctName,
			})
		}
	}

	return report, nil
}

// ExecuteRenames performs the file renames in idDir.
func ExecuteRenames(idDir string, renames []Rename) error {
	for _, rn := range renames {
		oldPath := filepath.Join(idDir, rn.OldName)
		newPath := filepath.Join(idDir, rn.NewName)
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("rename %s -> %s: %w", rn.OldName, rn.NewName, err)
		}
	}
	return nil
}

// RebuildSymlinks deletes notes/by/date and notes/by/tags, then re-scans
// notes/by/id and creates all symlinks from scratch.
func RebuildSymlinks(baseDir string) error {
	byDir := filepath.Join(baseDir, "notes", "by")
	idDir := filepath.Join(byDir, "id")

	// Delete existing symlink directories.
	for _, dir := range []string{"date", "tags"} {
		p := filepath.Join(byDir, dir)
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("rebuild symlinks: remove %s: %w", dir, err)
		}
	}

	// Re-scan and create symlinks, one file at a time.
	f, err := os.Open(idDir)
	if err != nil {
		return fmt.Errorf("rebuild symlinks: %w", err)
	}
	defer f.Close()

	for {
		entries, err := f.ReadDir(readDirBatch)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}

			id, _ := IDFromFilename(name)

			path := filepath.Join(idDir, name)
			nf, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("rebuild symlinks: open %s: %w", name, err)
			}

			note, err := ReadNote(id, nf)
			nf.Close()
			if err != nil {
				return fmt.Errorf("rebuild symlinks: read %s: %w", name, err)
			}

			// Use the actual filename on disk for symlink targets,
			// not a computed one. The file may have an old ID format.
			links := linkEntries(note, name)
			plan := &Plan{
				WritePath: filepath.Join("notes", "by", "id", name),
				Links:     links,
			}
			if err := plan.Execute(baseDir); err != nil {
				return fmt.Errorf("rebuild symlinks: %w", err)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("rebuild symlinks: read dir: %w", err)
		}
	}

	return nil
}
