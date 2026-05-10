package gonotes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func CreateNote(baseDir string, r io.Reader, opts PrepareOptions, dryRun bool) (*Note, *Plan, error) {
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	nowTime := now()
	opts.Now = func() time.Time { return nowTime }

	note, err := Prepare(r, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}

	idDir := filepath.Join(baseDir, "notes", "by", "id")
	id, err := NextID(idDir, nowTime)
	if err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}
	note.ID = id

	plan := NotePlan(note)

	if dryRun {
		return note, plan, nil
	}

	filename := NoteFilename(note.ID, note.Slug)
	writePath := filepath.Join(baseDir, "notes", "by", "id", filename)
	writeDir := filepath.Dir(writePath)
	if err := os.MkdirAll(writeDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}
	if err := os.WriteFile(writePath, []byte(note.Markdown()), 0o644); err != nil {
		return nil, nil, fmt.Errorf("create note: %w", err)
	}

	if err := plan.CreateLinks(baseDir); err != nil {
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
