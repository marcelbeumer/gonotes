package gonotes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var testTime = time.Date(2026, 3, 28, 14, 30, 0, 0, time.UTC)

func TestNextID(t *testing.T) {
	tests := []struct {
		name  string
		files []string // filenames to create in the dir
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

func TestWriteNote(t *testing.T) {
	dir := t.TempDir()

	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: Hello
date: 2026-03-28 14:30:00
---

Body.`))
	if err != nil {
		t.Fatal(err)
	}

	path, err := WriteNote(dir, note)
	if err != nil {
		t.Fatalf("WriteNote() err = %q", err)
	}

	wantPath := filepath.Join(dir, "20260328-1-hello.md")
	if path != wantPath {
		t.Errorf("WriteNote() path = %q, want %q", path, wantPath)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() err = %q", err)
	}

	if !strings.Contains(string(content), "title: Hello") {
		t.Errorf("written file does not contain expected frontmatter")
	}
	if !strings.Contains(string(content), "Body.") {
		t.Errorf("written file does not contain expected body")
	}
}

func TestWriteNoteCreatesDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "notes", "by", "id")

	note := NewNote()
	note.ID = "20260328-1"

	_, err := WriteNote(dir, note)
	if err != nil {
		t.Fatalf("WriteNote() err = %q", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "20260328-1.md")); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestNotePlan(t *testing.T) {
	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: Hello World
date: 2026-03-28 14:30:00
tags: foo/bar, plain
---

Body.`))
	if err != nil {
		t.Fatal(err)
	}

	plan := NotePlan(note)

	// Check write path.
	wantWrite := filepath.Join("notes", "by", "id", "20260328-1-hello-world.md")
	if plan.WritePath != wantWrite {
		t.Errorf("WritePath = %q, want %q", plan.WritePath, wantWrite)
	}

	// Collect link paths for easier assertion.
	gotPaths := make([]string, len(plan.Links))
	for i, l := range plan.Links {
		gotPaths[i] = l.Path
	}

	wantPaths := []string{
		// Date
		filepath.Join("notes", "by", "date", "2026-03-28", "20260328-1-hello-world.md"),
		// Nested foo/bar
		filepath.Join("notes", "by", "tags", "nested", "foo", "bar", "20260328-1-hello-world.md"),
		// Flat foo
		filepath.Join("notes", "by", "tags", "flat", "foo", "20260328-1-hello-world.md"),
		// Flat bar
		filepath.Join("notes", "by", "tags", "flat", "bar", "20260328-1-hello-world.md"),
		// Nested plain
		filepath.Join("notes", "by", "tags", "nested", "plain", "20260328-1-hello-world.md"),
		// Flat plain
		filepath.Join("notes", "by", "tags", "flat", "plain", "20260328-1-hello-world.md"),
	}

	if diff := cmp.Diff(wantPaths, gotPaths); diff != "" {
		t.Errorf("link paths diff (-want, +got):\n%s", diff)
	}
}

func TestNotePlanNoDate(t *testing.T) {
	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: No Date
tags: foo
---`))
	if err != nil {
		t.Fatal(err)
	}

	plan := NotePlan(note)

	datePfx := filepath.Join("notes", "by", "date")
	for _, l := range plan.Links {
		if strings.HasPrefix(l.Path, datePfx) {
			t.Errorf("unexpected date link: %s", l.Path)
		}
	}
}

func TestNotePlanNoTags(t *testing.T) {
	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: No Tags
date: 2026-03-28 14:30:00
---`))
	if err != nil {
		t.Fatal(err)
	}

	plan := NotePlan(note)

	// Should only have date link.
	if len(plan.Links) != 1 {
		t.Errorf("expected 1 link (date only), got %d", len(plan.Links))
	}
	if !strings.Contains(plan.Links[0].Path, "date") {
		t.Errorf("expected date link, got %s", plan.Links[0].Path)
	}
}

func TestPlanExecuteCreatesSymlinks(t *testing.T) {
	baseDir := t.TempDir()
	filename := "20260328-1-hello-world.md"

	// Create the source file first.
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(idDir, filename)
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &Plan{
		WritePath: filepath.Join("notes", "by", "id", filename),
		Links: []Link{
			{
				Path:   filepath.Join("notes", "by", "date", "2026-03-28", filename),
				Target: filepath.Join("..", "..", "id", filename),
			},
			{
				Path:   filepath.Join("notes", "by", "tags", "nested", "foo", "bar", filename),
				Target: filepath.Join("..", "..", "..", "..", "id", filename),
			},
			{
				Path:   filepath.Join("notes", "by", "tags", "flat", "foo", filename),
				Target: filepath.Join("..", "..", "..", "id", filename),
			},
		},
	}

	if err := plan.Execute(baseDir); err != nil {
		t.Fatalf("Execute() err = %q", err)
	}

	// Verify each symlink exists and resolves to the source file.
	for _, l := range plan.Links {
		absLink := filepath.Join(baseDir, l.Path)

		// Symlink itself exists.
		info, err := os.Lstat(absLink)
		if err != nil {
			t.Errorf("symlink %s does not exist: %v", l.Path, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", l.Path)
			continue
		}

		// Resolves to real file.
		resolved, err := filepath.EvalSymlinks(absLink)
		if err != nil {
			t.Errorf("cannot resolve symlink %s: %v", l.Path, err)
			continue
		}
		if resolved != srcPath {
			t.Errorf("symlink %s resolves to %q, want %q", l.Path, resolved, srcPath)
		}
	}
}

func TestPlanString(t *testing.T) {
	plan := &Plan{
		WritePath: "notes/by/id/20260328-1-hello.md",
		Links: []Link{
			{Path: "notes/by/date/2026-03-28/20260328-1-hello.md", Target: "../../id/20260328-1-hello.md"},
		},
	}

	got := plan.String()
	if !strings.Contains(got, "write: notes/by/id/20260328-1-hello.md") {
		t.Errorf("String() missing write line, got:\n%s", got)
	}
	if !strings.Contains(got, "link:  notes/by/date/2026-03-28/20260328-1-hello.md -> ../../id/20260328-1-hello.md") {
		t.Errorf("String() missing link line, got:\n%s", got)
	}
}

func TestCreateNote(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }

	input := strings.NewReader(`---
title: Integration Test
tags: test/integration, demo
---

Test body with [[20260101-1]] link.`)

	opts := PrepareOptions{Now: now}

	note, plan, err := CreateNote(baseDir, input, opts, "", false)
	if err != nil {
		t.Fatalf("CreateNote() err = %q", err)
	}

	// ID was generated.
	if note.ID != "20260328-1" {
		t.Errorf("ID = %q, want %q", note.ID, "20260328-1")
	}

	// File was written.
	writePath := filepath.Join(baseDir, plan.WritePath)
	content, err := os.ReadFile(writePath)
	if err != nil {
		t.Fatalf("note file not found: %v", err)
	}
	if !strings.Contains(string(content), "Integration Test") {
		t.Error("note file missing title")
	}

	// Symlinks exist and resolve.
	for _, l := range plan.Links {
		absLink := filepath.Join(baseDir, l.Path)
		resolved, err := filepath.EvalSymlinks(absLink)
		if err != nil {
			t.Errorf("symlink %s cannot be resolved: %v", l.Path, err)
			continue
		}
		if resolved != writePath {
			t.Errorf("symlink %s resolves to %q, want %q", l.Path, resolved, writePath)
		}
	}
}

func TestCreateNoteWithExplicitID(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }

	opts := PrepareOptions{
		Title: StringPtr("Explicit ID"),
		Now:   now,
	}

	note, _, err := CreateNote(baseDir, nil, opts, "20260101-42", false)
	if err != nil {
		t.Fatalf("CreateNote() err = %q", err)
	}

	if note.ID != "20260101-42" {
		t.Errorf("ID = %q, want %q", note.ID, "20260101-42")
	}

	path := filepath.Join(baseDir, "notes", "by", "id", "20260101-42-explicit-id.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("note file not found at expected path: %v", err)
	}
}

func TestCreateNoteDryRun(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }

	opts := PrepareOptions{
		Title: StringPtr("Dry Run"),
		Tags:  StringPtr("test"),
		Now:   now,
	}

	note, plan, err := CreateNote(baseDir, nil, opts, "", true)
	if err != nil {
		t.Fatalf("CreateNote() err = %q", err)
	}

	// Note and plan should be populated.
	if note.ID != "20260328-1" {
		t.Errorf("ID = %q, want %q", note.ID, "20260328-1")
	}
	if plan.WritePath == "" {
		t.Error("plan.WritePath is empty")
	}
	if len(plan.Links) == 0 {
		t.Error("plan has no links")
	}

	// But no files should have been created.
	writePath := filepath.Join(baseDir, plan.WritePath)
	if _, err := os.Stat(writePath); !os.IsNotExist(err) {
		t.Errorf("dry run created file at %s", writePath)
	}
	for _, l := range plan.Links {
		absLink := filepath.Join(baseDir, l.Path)
		if _, err := os.Lstat(absLink); !os.IsNotExist(err) {
			t.Errorf("dry run created symlink at %s", l.Path)
		}
	}
}

func TestCreateNoteSequentialIDs(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }

	opts := PrepareOptions{
		Title: StringPtr("First"),
		Now:   now,
	}

	note1, _, err := CreateNote(baseDir, nil, opts, "", false)
	if err != nil {
		t.Fatalf("first CreateNote() err = %q", err)
	}
	if note1.ID != "20260328-1" {
		t.Errorf("first ID = %q, want %q", note1.ID, "20260328-1")
	}

	opts.Title = StringPtr("Second")
	note2, _, err := CreateNote(baseDir, nil, opts, "", false)
	if err != nil {
		t.Fatalf("second CreateNote() err = %q", err)
	}
	if note2.ID != "20260328-2" {
		t.Errorf("second ID = %q, want %q", note2.ID, "20260328-2")
	}
}
