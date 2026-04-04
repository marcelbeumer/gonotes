package gonotes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var reUpdateIDArg = regexp.MustCompile(`^(\d{8}-\d+)(?:-.+)?$`)

type UpdateResult struct {
	Note        *Note
	Plan        *Plan
	CurrentPath string
	NewPath     string
	Changed     bool
}

func NormalizeUpdateIDArg(v string) (string, error) {
	m := reUpdateIDArg.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return "", fmt.Errorf("-i must start with yyyymmdd-N")
	}
	return m[1], nil
}

func ResolveNotePathByID(baseDir, idArg string) (string, error) {
	id, err := NormalizeUpdateIDArg(idArg)
	if err != nil {
		return "", err
	}

	idDir := filepath.Join(baseDir, "notes", "by", "id")
	f, err := os.Open(idDir)
	if err != nil {
		return "", fmt.Errorf("resolve note by id: %w", err)
	}
	defer f.Close()

	var matches []string
	for {
		entries, err := f.ReadDir(readDirBatch)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			entryID, parsed := IDFromFilename(e.Name())
			if !parsed || entryID != id {
				continue
			}
			matches = append(matches, filepath.Join(idDir, e.Name()))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("resolve note by id: %w", err)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no note found for id %q", id)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple notes found for id %q", id)
	}

	return matches[0], nil
}

func ListCanonicalNotePaths(baseDir string) ([]string, error) {
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	f, err := os.Open(idDir)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer f.Close()

	var paths []string
	for {
		entries, err := f.ReadDir(readDirBatch)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			_, parsed := IDFromFilename(e.Name())
			if !parsed {
				continue
			}
			paths = append(paths, filepath.Join(idDir, e.Name()))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("list notes: %w", err)
		}
	}

	return paths, nil
}

func UpdateNoteFile(baseDir, currentPath string, opts PrepareOptions, dryRun bool) (*UpdateResult, error) {
	absCurrent := currentPath
	if !filepath.IsAbs(absCurrent) {
		absCurrent = filepath.Join(baseDir, currentPath)
	}
	absCurrent = filepath.Clean(absCurrent)

	origBytes, err := os.ReadFile(absCurrent)
	if err != nil {
		return nil, fmt.Errorf("update note: read file: %w", err)
	}

	id, parsed := IDFromFilename(filepath.Base(absCurrent))
	if !parsed {
		return nil, fmt.Errorf("update note: filename must start with yyyymmdd-N")
	}

	note, err := Prepare(bytes.NewReader(origBytes), opts)
	if err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	note.ID = id

	plan := NotePlan(note)
	absNew := filepath.Clean(filepath.Join(baseDir, plan.WritePath))
	updatedMarkdown := note.Markdown()
	changed := string(origBytes) != updatedMarkdown || absCurrent != absNew

	if !dryRun && changed {
		if err := os.MkdirAll(filepath.Dir(absNew), 0o755); err != nil {
			return nil, fmt.Errorf("update note: %w", err)
		}
		if absCurrent != absNew {
			if err := os.Rename(absCurrent, absNew); err != nil {
				return nil, fmt.Errorf("update note: rename: %w", err)
			}
		}
		if err := os.WriteFile(absNew, []byte(updatedMarkdown), 0o644); err != nil {
			return nil, fmt.Errorf("update note: write file: %w", err)
		}
	}

	return &UpdateResult{
		Note:        note,
		Plan:        plan,
		CurrentPath: absCurrent,
		NewPath:     absNew,
		Changed:     changed,
	}, nil
}
