package gonotes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func CreateNote(baseDir string, r io.Reader, opts PrepareOptions, dryRun bool) (*Note, *Plan, error) {
	note, err := Prepare(r, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}

	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	id, err := NextID(idDir, now())
	if err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
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

func CreateFolder(baseDir, title string, now func() time.Time) (string, error) {
	filesDir := filepath.Join(baseDir, "files")

	id, err := NextID(filesDir, now())
	if err != nil {
		return "", fmt.Errorf("create folder: %w", err)
	}

	slug := slugify(title)
	name := FolderName(id, slug)
	absPath := filepath.Join(filesDir, name)

	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return "", fmt.Errorf("create folder: %w", err)
	}

	return absPath, nil
}
