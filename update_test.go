package gonotes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeUpdateIDArg(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "20260328-1", want: "20260328-1"},
		{in: "20260328-1-some-title", want: "20260328-1"},
		{in: "2026-03-28-1", wantErr: true},
		{in: "old-id", wantErr: true},
	}

	for _, tt := range tests {
		got, err := NormalizeUpdateIDArg(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("NormalizeUpdateIDArg(%q) err = <nil>, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeUpdateIDArg(%q) err = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("NormalizeUpdateIDArg(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveNotePathByID(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	writeTestNote(t, idDir, "20260328-1-hello.md", "---\ntitle: Hello\n---\n")

	path, err := ResolveNotePathByID(baseDir, "20260328-1-any-slug")
	if err != nil {
		t.Fatalf("ResolveNotePathByID() err = %v", err)
	}
	want := filepath.Join(idDir, "20260328-1-hello.md")
	if path != want {
		t.Fatalf("ResolveNotePathByID() = %q, want %q", path, want)
	}
}

func TestListCanonicalNotePaths(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	writeTestNote(t, idDir, "20260328-1-hello.md", "---\ntitle: Hello\n---\n")
	writeTestNote(t, idDir, "readme.md", "---\ntitle: Readme\n---\n")

	paths, err := ListCanonicalNotePaths(baseDir)
	if err != nil {
		t.Fatalf("ListCanonicalNotePaths() err = %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("ListCanonicalNotePaths() count = %d, want 1", len(paths))
	}
}

func TestUpdateNoteFileRenamesOnTitleChange(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	oldName := "20260328-1-old-title.md"
	writeTestNote(t, idDir, oldName, "---\ntitle: Old Title\ndate: 2026-03-28 14:30:00\n---\n\nBody\n")

	newTitle := "New Title"
	res, err := UpdateNoteFile(baseDir, filepath.Join(idDir, oldName), PrepareOptions{
		Title: &newTitle,
	}, false)
	if err != nil {
		t.Fatalf("UpdateNoteFile() err = %v", err)
	}

	if !res.Changed {
		t.Fatal("UpdateNoteFile() Changed = false, want true")
	}
	if filepath.Base(res.NewPath) != "20260328-1-new-title.md" {
		t.Fatalf("new path = %q", res.NewPath)
	}
	if _, err := os.Stat(filepath.Join(idDir, "20260328-1-new-title.md")); err != nil {
		t.Fatalf("new file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(idDir, oldName)); !os.IsNotExist(err) {
		t.Fatalf("old file still exists")
	}
}
