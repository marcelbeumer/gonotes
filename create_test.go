package gonotes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestCreateNote(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }

	input := strings.NewReader(`---
title: Integration Test
tags: test/integration, demo
---

Test body with [[20260101-1]] link.`)

	opts := PrepareOptions{Now: now}

	note, plan, err := CreateNote(baseDir, input, opts, false)
	if err != nil {
		t.Fatalf("CreateNote() err = %q", err)
	}

	if note.ID != "20260328-1" {
		t.Errorf("ID = %q, want %q", note.ID, "20260328-1")
	}

	filename := NoteFilename(note.ID, note.Slug)
	writePath := filepath.Join(baseDir, "notes", "by", "id", filename)
	content, err := os.ReadFile(writePath)
	if err != nil {
		t.Fatalf("note file not found: %v", err)
	}
	if !strings.Contains(string(content), "Integration Test") {
		t.Error("note file missing title")
	}

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

func TestCreateNoteUsesNextID(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	writeTestNote(t, idDir, "20260328-3-existing.md", "---\ntitle: Existing\n---\n")

	opts := PrepareOptions{
		Title: StringPtr("Generated ID"),
		Now:   now,
	}

	note, _, err := CreateNote(baseDir, nil, opts, false)
	if err != nil {
		t.Fatalf("CreateNote() err = %q", err)
	}

	if note.ID != "20260328-4" {
		t.Errorf("ID = %q, want %q", note.ID, "20260328-4")
	}

	path := filepath.Join(baseDir, "notes", "by", "id", "20260328-4-generated-id.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("note file not found at expected path: %v", err)
	}
}

func TestCreateNoteDryRun(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }

	opts := PrepareOptions{
		Title: StringPtr("Dry Run"),
		Tags:  []string{"test"},
		Now:   now,
	}

	note, plan, err := CreateNote(baseDir, nil, opts, true)
	if err != nil {
		t.Fatalf("CreateNote() err = %q", err)
	}

	if note.ID != "20260328-1" {
		t.Errorf("ID = %q, want %q", note.ID, "20260328-1")
	}
	if len(plan.Links) == 0 {
		t.Error("plan has no links")
	}

	filename := NoteFilename(note.ID, note.Slug)
	writePath := filepath.Join(baseDir, "notes", "by", "id", filename)
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

	note1, _, err := CreateNote(baseDir, nil, opts, false)
	if err != nil {
		t.Fatalf("first CreateNote() err = %q", err)
	}
	if note1.ID != "20260328-1" {
		t.Errorf("first ID = %q, want %q", note1.ID, "20260328-1")
	}

	opts.Title = StringPtr("Second")
	note2, _, err := CreateNote(baseDir, nil, opts, false)
	if err != nil {
		t.Fatalf("second CreateNote() err = %q", err)
	}
	if note2.ID != "20260328-2" {
		t.Errorf("second ID = %q, want %q", note2.ID, "20260328-2")
	}
}

func TestCreateNoteAndRebuildSymlinksAreStable(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time { return testTime }

	opts := PrepareOptions{
		Title: StringPtr("Stable Symlinks"),
		Tags:  []string{"foo/bar", "baz"},
		Now:   now,
	}

	_, _, err := CreateNote(baseDir, nil, opts, false)
	if err != nil {
		t.Fatalf("CreateNote() err = %q", err)
	}

	before, err := snapshotNoteSymlinks(baseDir)
	if err != nil {
		t.Fatalf("snapshot before rebuild: %v", err)
	}

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	after, err := snapshotNoteSymlinks(baseDir)
	if err != nil {
		t.Fatalf("snapshot after rebuild: %v", err)
	}

	if diff := cmp.Diff(before, after); diff != "" {
		t.Errorf("symlink snapshot diff (-before, +after):\n%s", diff)
	}
}

func TestCreateFolder(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time {
		return time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	}

	path, err := CreateFolder(baseDir, "Contract PDFs", now)
	if err != nil {
		t.Fatalf("CreateFolder() err = %q", err)
	}

	wantDir := filepath.Join(baseDir, "files", "20260403-1-contract-pdfs")
	if path != wantDir {
		t.Errorf("CreateFolder() = %q, want %q", path, wantDir)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("created directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("created path is not a directory")
	}
}

func TestCreateFolderSequentialIDs(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time {
		return time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	}

	path1, err := CreateFolder(baseDir, "First", now)
	if err != nil {
		t.Fatalf("CreateFolder(1) err = %q", err)
	}

	path2, err := CreateFolder(baseDir, "Second", now)
	if err != nil {
		t.Fatalf("CreateFolder(2) err = %q", err)
	}

	want1 := filepath.Join(baseDir, "files", "20260403-1-first")
	want2 := filepath.Join(baseDir, "files", "20260403-2-second")

	if path1 != want1 {
		t.Errorf("first folder = %q, want %q", path1, want1)
	}
	if path2 != want2 {
		t.Errorf("second folder = %q, want %q", path2, want2)
	}
}

func TestCreateFolderNoTitle(t *testing.T) {
	baseDir := t.TempDir()
	now := func() time.Time {
		return time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	}

	path, err := CreateFolder(baseDir, "", now)
	if err != nil {
		t.Fatalf("CreateFolder() err = %q", err)
	}

	wantDir := filepath.Join(baseDir, "files", "20260403-1")
	if path != wantDir {
		t.Errorf("CreateFolder() = %q, want %q", path, wantDir)
	}
}
