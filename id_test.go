package gonotes

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNextID(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		now   time.Time
		want  string
	}{
		{
			name:  "empty dir",
			files: nil,
			now:   testTime,
			want:  "20260328-1",
		},
		{
			name:  "one note today",
			files: []string{"20260328-1-hello.md"},
			now:   testTime,
			want:  "20260328-2",
		},
		{
			name:  "multiple notes today",
			files: []string{"20260328-1-hello.md", "20260328-3-world.md", "20260328-2-foo.md"},
			now:   testTime,
			want:  "20260328-4",
		},
		{
			name:  "notes from other dates only",
			files: []string{"20260327-1-old.md", "20260101-5-ancient.md"},
			now:   testTime,
			want:  "20260328-1",
		},
		{
			name:  "mixed dates",
			files: []string{"20260328-2-today.md", "20260327-99-yesterday.md"},
			now:   testTime,
			want:  "20260328-3",
		},
		{
			name:  "holes in numbering",
			files: []string{"20260328-1-a.md", "20260328-5-b.md"},
			now:   testTime,
			want:  "20260328-6",
		},
		{
			name:  "non-matching files ignored",
			files: []string{"readme.txt", ".hidden", "20260328-notanumber-bad.md"},
			now:   testTime,
			want:  "20260328-1",
		},
		{
			name:  "no slug in filename",
			files: []string{"20260328-3.md"},
			now:   testTime,
			want:  "20260328-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := NextID(dir, tt.now)
			if err != nil {
				t.Fatalf("NextID() err = %q", err)
			}
			if got != tt.want {
				t.Errorf("NextID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNextIDNonExistentDir(t *testing.T) {
	got, err := NextID("/tmp/gonotes-test-nonexistent-dir-12345", testTime)
	if err != nil {
		t.Fatalf("NextID() err = %q", err)
	}
	if got != "20260328-1" {
		t.Errorf("NextID() = %q, want %q", got, "20260328-1")
	}
}

func TestNoteFilename(t *testing.T) {
	tests := []struct {
		id, slug, want string
	}{
		{"20260328-1", "hello-world", "20260328-1-hello-world.md"},
		{"20260328-1", "", "20260328-1.md"},
		{"20260101-42", "a", "20260101-42-a.md"},
	}

	for _, tt := range tests {
		got := NoteFilename(tt.id, tt.slug)
		if got != tt.want {
			t.Errorf("NoteFilename(%q, %q) = %q, want %q", tt.id, tt.slug, got, tt.want)
		}
	}
}

func TestFolderName(t *testing.T) {
	tests := []struct {
		id, slug, want string
	}{
		{"20260403-1", "contract-pdfs", "20260403-1-contract-pdfs"},
		{"20260403-1", "", "20260403-1"},
		{"20260101-42", "a", "20260101-42-a"},
	}

	for _, tt := range tests {
		got := FolderName(tt.id, tt.slug)
		if got != tt.want {
			t.Errorf("FolderName(%q, %q) = %q, want %q", tt.id, tt.slug, got, tt.want)
		}
	}
}

func TestIDFromFilename(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantID     string
		wantParsed bool
	}{
		{"new format with slug", "20260328-1-hello-world.md", "20260328-1", true},
		{"new format no slug", "20260328-1.md", "20260328-1", true},
		{"new format large num", "20260328-42-foo.md", "20260328-42", true},
		{"old format with slug", "2026-02-12-2233-05-pacman-cheatcheat.md", "2026-02-12-2233-05-pacman-cheatcheat", false},
		{"old format no slug", "2026-02-12-2233-05.md", "2026-02-12-2233-05", false},
		{"no numeric prefix", "notes-abc.md", "notes-abc", false},
		{"plain name", "readme.md", "readme", false},
		{"not md", "readme.txt", "", false},
		{"hidden file", ".hidden", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, parsed := IDFromFilename(tt.input)
			if parsed != tt.wantParsed {
				t.Errorf("IDFromFilename(%q) parsed = %v, want %v", tt.input, parsed, tt.wantParsed)
			}
			if id != tt.wantID {
				t.Errorf("IDFromFilename(%q) = %q, want %q", tt.input, id, tt.wantID)
			}
		})
	}
}

func TestMaxNumFromDir(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		now   time.Time
		want  int
	}{
		{
			name:  "empty dir",
			files: nil,
			now:   testTime,
			want:  0,
		},
		{
			name:  "one note today",
			files: []string{"20260328-1-hello.md"},
			now:   testTime,
			want:  1,
		},
		{
			name:  "multiple notes today",
			files: []string{"20260328-1-a.md", "20260328-3-b.md", "20260328-2-c.md"},
			now:   testTime,
			want:  3,
		},
		{
			name:  "notes from other dates only",
			files: []string{"20260327-5-old.md"},
			now:   testTime,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := MaxNumFromDir(dir, tt.now)
			if err != nil {
				t.Fatalf("MaxNumFromDir() err = %q", err)
			}
			if got != tt.want {
				t.Errorf("MaxNumFromDir() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMaxNumFromDirNonExistent(t *testing.T) {
	got, err := MaxNumFromDir("/tmp/gonotes-test-nonexistent-maxnum-12345", testTime)
	if err != nil {
		t.Fatalf("MaxNumFromDir() err = %q", err)
	}
	if got != 0 {
		t.Errorf("MaxNumFromDir() = %d, want 0", got)
	}
}

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
