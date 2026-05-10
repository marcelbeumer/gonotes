package gonotes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TagChange struct {
	ID      string
	Path    string
	OldTags []string
	NewTags []string
}

func (tc TagChange) String() string {
	return fmt.Sprintf("%s: %v -> %v", tc.ID, tc.OldTags, tc.NewTags)
}

type ReverseRebuildReport struct {
	Changes   []TagChange
	Unchanged int
	Errors    []ScanError
}

func (r *ReverseRebuildReport) String() string {
	var b strings.Builder

	if len(r.Changes) > 0 {
		fmt.Fprintf(&b, "Tag changes (%d):\n", len(r.Changes))
		for _, tc := range r.Changes {
			fmt.Fprintf(&b, "  %s\n", tc.String())
		}
	}

	if r.Unchanged > 0 {
		fmt.Fprintf(&b, "Unchanged: %d\n", r.Unchanged)
	}

	if len(r.Errors) > 0 {
		fmt.Fprintf(&b, "Errors (%d):\n", len(r.Errors))
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "  %s: %s\n", e.Filename, e.Message)
		}
	}

	if len(r.Changes) == 0 && r.Unchanged == 0 && len(r.Errors) == 0 {
		b.WriteString("No notes found.\n")
	}

	return b.String()
}

func ScanTagsFromFS(baseDir string) (map[string][]string, error) {
	tagsDir := filepath.Join(baseDir, "notes", "by", "tags")

	result := make(map[string][]string)

	if _, err := os.Stat(tagsDir); os.IsNotExist(err) {
		return result, nil
	}

	err := filepath.WalkDir(tagsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		id, parsed := IDFromFilename(d.Name())
		if !parsed {
			return nil
		}

		rel, err := filepath.Rel(tagsDir, path)
		if err != nil {
			return fmt.Errorf("scan tags: %w", err)
		}

		tagPath := filepath.Dir(rel)
		tag := strings.ReplaceAll(tagPath, string(filepath.Separator), "/")

		result[id] = append(result[id], tag)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan tags: %w", err)
	}

	for id, tags := range result {
		sort.Strings(tags)
		result[id] = tags
	}

	return result, nil
}

func tagsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ReverseRebuild(baseDir string) (*ReverseRebuildReport, error) {
	fsTags, err := ScanTagsFromFS(baseDir)
	if err != nil {
		return nil, fmt.Errorf("reverse rebuild: %w", err)
	}

	idDir := filepath.Join(baseDir, "notes", "by", "id")

	f, err := os.Open(idDir)
	if err != nil {
		return nil, fmt.Errorf("reverse rebuild: %w", err)
	}
	defer f.Close()

	report := &ReverseRebuildReport{}

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
			if !parsed {
				continue
			}

			path := filepath.Join(idDir, name)
			nf, err := os.Open(path)
			if err != nil {
				report.Errors = append(report.Errors, ScanError{
					Filename: name,
					Message:  fmt.Sprintf("open: %v", err),
				})
				continue
			}

			note, err := ReadNote(id, nf)
			nf.Close()
			if err != nil {
				report.Errors = append(report.Errors, ScanError{
					Filename: name,
					Message:  fmt.Sprintf("read note: %v", err),
				})
				continue
			}

			fromFS := fsTags[id]
			newTags := mergeTags(note.Tags, fromFS)

			if tagsEqual(note.Tags, newTags) {
				report.Unchanged++
				continue
			}

			report.Changes = append(report.Changes, TagChange{
				ID:      id,
				Path:    path,
				OldTags: note.Tags,
				NewTags: newTags,
			})
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("reverse rebuild: %w", err)
		}
	}

	sort.Slice(report.Changes, func(i, j int) bool {
		return report.Changes[i].ID < report.Changes[j].ID
	})

	return report, nil
}

func ExecuteReverseRebuild(baseDir string, changes []TagChange) error {
	for _, tc := range changes {
		data, err := os.ReadFile(tc.Path)
		if err != nil {
			return fmt.Errorf("reverse rebuild: read %s: %w", tc.Path, err)
		}

		note, err := ReadNote(tc.ID, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("reverse rebuild: parse %s: %w", tc.Path, err)
		}

		if len(tc.NewTags) == 0 {
			note.Frontmatter.Unset("tags")
		} else {
			note.Frontmatter.Set("tags", FormatTags(tc.NewTags))
		}
		note.Tags = tc.NewTags

		if err := os.WriteFile(tc.Path, []byte(note.Markdown()), 0o644); err != nil {
			return fmt.Errorf("reverse rebuild: write %s: %w", tc.Path, err)
		}
	}

	return RebuildSymlinks(baseDir)
}
