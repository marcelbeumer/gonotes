package gonotes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// noteFile is a parsed note together with its original filename on disk.
type noteFile struct {
	Note
	Filename string
}

// readNoteFile reads a single note file by name from dir.
// On error it appends to errs and returns nil.
func readNoteFile(dir, name string, errs *[]ScanError) *noteFile {
	id, _ := IDFromFilename(name)
	path := filepath.Join(dir, name)

	f, err := os.Open(path)
	if err != nil {
		*errs = append(*errs, ScanError{
			Filename: name,
			Message:  fmt.Sprintf("open: %v", err),
		})
		return nil
	}
	defer f.Close()

	note, err := ReadNote(id, f)
	if err != nil {
		*errs = append(*errs, ScanError{
			Filename: name,
			Message:  fmt.Sprintf("read note: %v", err),
		})
		return nil
	}
	return &noteFile{Note: *note, Filename: name}
}

// readNoteFiles reads all .md files from dir and parses them.
// It returns the parsed noteFiles and any per-file errors.
func readNoteFiles(dir string) ([]noteFile, []ScanError, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("open notes dir: %w", err)
	}
	defer f.Close()

	var files []noteFile
	var errs []ScanError

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
			nf := readNoteFile(dir, name, &errs)
			if nf != nil {
				files = append(files, *nf)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("read dir: %w", err)
		}
	}

	return files, errs, nil
}

// readNotesFromDir reads all .md files from dir and returns just the Notes.
// Used by callers that don't need the original filenames.
func readNotesFromDir(dir string) ([]Note, []ScanError, error) {
	files, errs, err := readNoteFiles(dir)
	if err != nil {
		return nil, nil, err
	}
	notes := make([]Note, len(files))
	for i := range files {
		notes[i] = files[i].Note
	}
	return notes, errs, nil
}
