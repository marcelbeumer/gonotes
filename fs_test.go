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

func TestIDFromFilename(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantID     string
		wantParsed bool
	}{
		// New format.
		{"new format with slug", "20260328-1-hello-world.md", "20260328-1", true},
		{"new format no slug", "20260328-1.md", "20260328-1", true},
		{"new format large num", "20260328-42-foo.md", "20260328-42", true},
		// Old format.
		{"old format with slug", "2026-02-12-2233-05-pacman-cheatcheat.md", "2026-02-12-2233-05", true},
		{"old format no slug", "2026-02-12-2233-05.md", "2026-02-12-2233-05", true},
		// Non-matching .md files: still processed, ID is full stem.
		{"no numeric prefix", "notes-abc.md", "notes-abc", false},
		{"plain name", "readme.md", "readme", false},
		// Non-.md files: not processed.
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

// writeTestNote is a helper that writes a note file directly to a directory.
func writeTestNote(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanNotes(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	// Note 1: correct filename, links to note 2 (exists) and note 99 (doesn't exist).
	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
---

See [[20260328-2]] and [[20260328-99]].`)

	// Note 2: wrong filename (slug doesn't match title).
	writeTestNote(t, idDir, "20260328-2-old-name.md", `---
title: New Name
date: 2026-03-28 15:00:00
---

Links to [[20260328-1]].`)

	// Note 3: no title, no links.
	writeTestNote(t, idDir, "20260328-3.md", `---
date: 2026-03-28 16:00:00
---

Plain note.`)

	report, err := ScanNotes(idDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	// Check broken links: only 20260328-99 should be broken.
	if len(report.BrokenLinks) != 1 {
		t.Errorf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	} else {
		bl := report.BrokenLinks[0]
		if bl.SourceID != "20260328-1" || bl.TargetID != "20260328-99" {
			t.Errorf("broken link = %v, want {20260328-1 -> 20260328-99}", bl)
		}
	}

	// Check renames: note 2 should be renamed.
	if len(report.Renames) != 1 {
		t.Errorf("expected 1 rename, got %d: %v", len(report.Renames), report.Renames)
	} else {
		rn := report.Renames[0]
		if rn.OldName != "20260328-2-old-name.md" || rn.NewName != "20260328-2-new-name.md" {
			t.Errorf("rename = %v, want {20260328-2-old-name.md -> 20260328-2-new-name.md}", rn)
		}
	}
}

func TestScanNotesNoIssues(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
---

No links.`)

	report, err := ScanNotes(idDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 0 {
		t.Errorf("expected 0 broken links, got %d", len(report.BrokenLinks))
	}
	if len(report.Renames) != 0 {
		t.Errorf("expected 0 renames, got %d", len(report.Renames))
	}
}

func TestScanNotesOldFormat(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	// Old-format note with correct slug.
	writeTestNote(t, idDir, "2026-02-12-2233-05-pacman-cheatsheet.md", `---
title: Pacman Cheatsheet
date: 2026-02-12 22:33:05
tags: linux
---

Pacman tips.`)

	// Old-format note with wrong slug (title differs).
	writeTestNote(t, idDir, "2026-01-05-1200-00-old-title.md", `---
title: New Title
date: 2026-01-05 12:00:00
---

Some content.`)

	// New-format note that links to the old-format note.
	writeTestNote(t, idDir, "20260328-1-linker.md", `---
title: Linker
date: 2026-03-28 14:30:00
---

See [[2026-02-12-2233-05]] and [[9999-99-99]].`)

	report, err := ScanNotes(idDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	// Broken link: 9999-99-99 doesn't exist. The old-format link should NOT be broken.
	if len(report.BrokenLinks) != 1 {
		t.Errorf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	} else if report.BrokenLinks[0].TargetID != "9999-99-99" {
		t.Errorf("broken link target = %q, want %q", report.BrokenLinks[0].TargetID, "9999-99-99")
	}

	// Rename: old-title should become new-title.
	if len(report.Renames) != 1 {
		t.Errorf("expected 1 rename, got %d: %v", len(report.Renames), report.Renames)
	} else {
		rn := report.Renames[0]
		if rn.OldName != "2026-01-05-1200-00-old-title.md" || rn.NewName != "2026-01-05-1200-00-new-title.md" {
			t.Errorf("rename = {%s -> %s}, want {2026-01-05-1200-00-old-title.md -> 2026-01-05-1200-00-new-title.md}", rn.OldName, rn.NewName)
		}
	}
}

func TestScanNotesNonMatchingFiles(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	// Normal note.
	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
---

Links to [[readme]].`)

	// Non-matching .md file: should be picked up, no rename attempted.
	writeTestNote(t, idDir, "readme.md", `---
title: Readme
---

Some info.`)

	report, err := ScanNotes(idDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	// The link to [[readme]] should NOT be broken (readme.md is in the set).
	if len(report.BrokenLinks) != 0 {
		t.Errorf("expected 0 broken links, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	}

	// No renames: readme.md has no parsed ID, so no rename check.
	if len(report.Renames) != 0 {
		t.Errorf("expected 0 renames, got %d: %v", len(report.Renames), report.Renames)
	}
}

func TestScanNotesNoTitleStripsSlug(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	// Note with parsed ID but no title: should be flagged for rename
	// to strip the slug from the filename.
	writeTestNote(t, idDir, "20260328-1-some-slug.md", `---
date: 2026-03-28 14:30:00
---

No title here.`)

	report, err := ScanNotes(idDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.Renames) != 1 {
		t.Errorf("expected 1 rename, got %d: %v", len(report.Renames), report.Renames)
	} else {
		rn := report.Renames[0]
		if rn.OldName != "20260328-1-some-slug.md" || rn.NewName != "20260328-1.md" {
			t.Errorf("rename = {%s -> %s}, want {20260328-1-some-slug.md -> 20260328-1.md}", rn.OldName, rn.NewName)
		}
	}
}

func TestExecuteRenames(t *testing.T) {
	dir := t.TempDir()

	writeTestNote(t, dir, "20260328-1-old.md", "content1")
	writeTestNote(t, dir, "20260328-2-wrong.md", "content2")

	renames := []Rename{
		{OldName: "20260328-1-old.md", NewName: "20260328-1-new.md"},
		{OldName: "20260328-2-wrong.md", NewName: "20260328-2-right.md"},
	}

	if err := ExecuteRenames(dir, renames); err != nil {
		t.Fatalf("ExecuteRenames() err = %q", err)
	}

	// Old names should not exist.
	for _, rn := range renames {
		if _, err := os.Stat(filepath.Join(dir, rn.OldName)); !os.IsNotExist(err) {
			t.Errorf("old file %s still exists", rn.OldName)
		}
	}

	// New names should exist with correct content.
	content1, _ := os.ReadFile(filepath.Join(dir, "20260328-1-new.md"))
	if string(content1) != "content1" {
		t.Errorf("renamed file content = %q, want %q", content1, "content1")
	}
	content2, _ := os.ReadFile(filepath.Join(dir, "20260328-2-right.md"))
	if string(content2) != "content2" {
		t.Errorf("renamed file content = %q, want %q", content2, "content2")
	}
}

func TestRebuildSymlinks(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	// Create two notes.
	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo/bar, plain
---

Body.`)

	writeTestNote(t, idDir, "20260328-2-world.md", `---
title: World
date: 2026-03-29 10:00:00
tags: other
---

Body.`)

	// Create stale symlink dirs (should be deleted).
	staleDir := filepath.Join(baseDir, "notes", "by", "tags", "flat", "stale")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	// Stale dir should be gone.
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Error("stale tag directory still exists after rebuild")
	}

	// Check expected symlinks exist and resolve.
	wantLinks := []struct {
		link   string
		target string // the note filename it should resolve to
	}{
		{"notes/by/date/2026-03-28/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/date/2026-03-29/20260328-2-world.md", "20260328-2-world.md"},
		{"notes/by/tags/nested/foo/bar/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/tags/nested/plain/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/tags/flat/foo/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/tags/flat/bar/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/tags/flat/plain/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/tags/nested/other/20260328-2-world.md", "20260328-2-world.md"},
		{"notes/by/tags/flat/other/20260328-2-world.md", "20260328-2-world.md"},
	}

	for _, wl := range wantLinks {
		absLink := filepath.Join(baseDir, wl.link)
		resolved, err := filepath.EvalSymlinks(absLink)
		if err != nil {
			t.Errorf("symlink %s cannot be resolved: %v", wl.link, err)
			continue
		}
		wantTarget := filepath.Join(idDir, wl.target)
		if resolved != wantTarget {
			t.Errorf("symlink %s resolves to %q, want %q", wl.link, resolved, wantTarget)
		}
	}
}

func TestRebuildSymlinksIdempotent(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: test
---`)

	// First rebuild.
	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("first RebuildSymlinks() err = %q", err)
	}

	// Second rebuild should also succeed (deletes old, creates new).
	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("second RebuildSymlinks() err = %q", err)
	}

	// Symlink should still work.
	link := filepath.Join(baseDir, "notes", "by", "date", "2026-03-28", "20260328-1-hello.md")
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatalf("symlink cannot be resolved after second rebuild: %v", err)
	}
	wantTarget := filepath.Join(idDir, "20260328-1-hello.md")
	if resolved != wantTarget {
		t.Errorf("symlink resolves to %q, want %q", resolved, wantTarget)
	}
}

func TestRebuildSymlinksOldFormat(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	// Old-format note.
	writeTestNote(t, idDir, "2026-02-12-2233-05-pacman-cheatsheet.md", `---
title: Pacman Cheatsheet
date: 2026-02-12 22:33:05
tags: linux/pacman
---

Tips.`)

	// Non-matching .md file.
	writeTestNote(t, idDir, "readme.md", `---
title: Readme
date: 2026-01-01 00:00:00
tags: meta
---

Info.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	// Old-format note should have symlinks using the actual filename.
	wantLinks := []struct {
		link   string
		target string
	}{
		// Old-format note symlinks.
		{"notes/by/date/2026-02-12/2026-02-12-2233-05-pacman-cheatsheet.md", "2026-02-12-2233-05-pacman-cheatsheet.md"},
		{"notes/by/tags/nested/linux/pacman/2026-02-12-2233-05-pacman-cheatsheet.md", "2026-02-12-2233-05-pacman-cheatsheet.md"},
		{"notes/by/tags/flat/linux/2026-02-12-2233-05-pacman-cheatsheet.md", "2026-02-12-2233-05-pacman-cheatsheet.md"},
		{"notes/by/tags/flat/pacman/2026-02-12-2233-05-pacman-cheatsheet.md", "2026-02-12-2233-05-pacman-cheatsheet.md"},
		// Non-matching .md file symlinks.
		{"notes/by/date/2026-01-01/readme.md", "readme.md"},
		{"notes/by/tags/nested/meta/readme.md", "readme.md"},
		{"notes/by/tags/flat/meta/readme.md", "readme.md"},
	}

	for _, wl := range wantLinks {
		absLink := filepath.Join(baseDir, wl.link)
		resolved, err := filepath.EvalSymlinks(absLink)
		if err != nil {
			t.Errorf("symlink %s cannot be resolved: %v", wl.link, err)
			continue
		}
		wantTarget := filepath.Join(idDir, wl.target)
		if resolved != wantTarget {
			t.Errorf("symlink %s resolves to %q, want %q", wl.link, resolved, wantTarget)
		}
	}
}

func TestRebuildReportString(t *testing.T) {
	t.Run("no issues", func(t *testing.T) {
		r := &RebuildReport{}
		got := r.String()
		if got != "No issues found.\n" {
			t.Errorf("String() = %q, want %q", got, "No issues found.\n")
		}
	})

	t.Run("with issues", func(t *testing.T) {
		r := &RebuildReport{
			BrokenLinks: []BrokenLink{{SourceID: "a", TargetID: "b"}},
			Renames:     []Rename{{OldName: "old.md", NewName: "new.md"}},
		}
		got := r.String()
		if !strings.Contains(got, "Broken links (1)") {
			t.Errorf("String() missing broken links header")
		}
		if !strings.Contains(got, "a -> b") {
			t.Errorf("String() missing broken link entry")
		}
		if !strings.Contains(got, "Renames (1)") {
			t.Errorf("String() missing renames header")
		}
		if !strings.Contains(got, "old.md -> new.md") {
			t.Errorf("String() missing rename entry")
		}
	})
}
