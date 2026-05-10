package gonotes

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var fixedNow = func() time.Time {
	return time.Date(2026, 3, 28, 14, 30, 0, 0, time.Local)
}

var testTime = time.Date(2026, 3, 28, 14, 30, 0, 0, time.UTC)

func writeTestNote(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshotNoteSymlinks(baseDir string) (map[string]string, error) {
	paths := []string{
		filepath.Join(baseDir, "notes", "by", "date"),
		filepath.Join(baseDir, "notes", "by", "tags"),
	}
	out := map[string]string{}
	for _, root := range paths {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return nil
			}
			rel, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			out[rel] = target
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
