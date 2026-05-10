package gonotes

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestScanTagsFromFS(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo/bar, plain
---

Body.`)

	writeTestNote(t, idDir, "20260328-2-world.md", `---
title: World
date: 2026-03-28 15:00:00
tags: other
---

Body.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	fsTags, err := ScanTagsFromFS(baseDir)
	if err != nil {
		t.Fatalf("ScanTagsFromFS() err = %q", err)
	}

	sort.Strings(fsTags["20260328-1"])
	want1 := []string{"foo/bar", "plain"}
	if diff := cmp.Diff(want1, fsTags["20260328-1"]); diff != "" {
		t.Errorf("tags for 20260328-1 diff (-want, +got):\n%s", diff)
	}

	want2 := []string{"other"}
	if diff := cmp.Diff(want2, fsTags["20260328-2"]); diff != "" {
		t.Errorf("tags for 20260328-2 diff (-want, +got):\n%s", diff)
	}
}

func TestScanTagsFromFSHierarchical(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo/bar, foo/bar/zar
---

Body.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	fsTags, err := ScanTagsFromFS(baseDir)
	if err != nil {
		t.Fatalf("ScanTagsFromFS() err = %q", err)
	}

	sort.Strings(fsTags["20260328-1"])
	want := []string{"foo/bar", "foo/bar/zar"}
	if diff := cmp.Diff(want, fsTags["20260328-1"]); diff != "" {
		t.Errorf("tags diff (-want, +got):\n%s", diff)
	}
}

func TestScanTagsFromFSNoTags(t *testing.T) {
	baseDir := t.TempDir()

	_, err := ScanTagsFromFS(baseDir)
	if err != nil {
		t.Fatalf("ScanTagsFromFS() with no tags dir err = %q", err)
	}
}

func TestScanTagsFromFSPlainFile(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: existing
---

Body.`)

	tagsDir := filepath.Join(baseDir, "notes", "by", "tags")
	newTag := filepath.Join(tagsDir, "new-tag")
	if err := os.MkdirAll(newTag, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newTag, "20260328-1-wrong-suffix.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	fsTags, err := ScanTagsFromFS(baseDir)
	if err != nil {
		t.Fatalf("ScanTagsFromFS() err = %q", err)
	}

	want := []string{"new-tag"}
	if diff := cmp.Diff(want, fsTags["20260328-1"]); diff != "" {
		t.Errorf("tags diff (-want, +got):\n%s", diff)
	}
}

func TestReverseRebuild(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo/bar, plain
---

Body.`)

	writeTestNote(t, idDir, "20260328-2-world.md", `---
title: World
date: 2026-03-28 15:00:00
tags: other
---

Body.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	tagsDir := filepath.Join(baseDir, "notes", "by", "tags")

	tagsNew := filepath.Join(tagsDir, "new-tag")
	if err := os.MkdirAll(tagsNew, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tagsNew, "20260328-1-hello.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := ReverseRebuild(baseDir)
	if err != nil {
		t.Fatalf("ReverseRebuild() err = %q", err)
	}

	if len(report.Errors) > 0 {
		t.Errorf("unexpected errors: %v", report.Errors)
	}

	changedIDs := make(map[string]bool)
	for _, tc := range report.Changes {
		changedIDs[tc.ID] = true
	}
	if !changedIDs["20260328-1"] {
		t.Error("expected 20260328-1 to have tag changes")
	}
	if changedIDs["20260328-2"] {
		t.Error("expected 20260328-2 to be unchanged")
	}

	for _, tc := range report.Changes {
		if tc.ID == "20260328-1" {
			want := []string{"foo/bar", "plain", "new-tag"}
			if diff := cmp.Diff(want, tc.NewTags); diff != "" {
				t.Errorf("new tags for 20260328-1 diff (-want, +got):\n%s", diff)
			}
		}
	}
}

func TestReverseRebuildRemoveTag(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo/bar, plain
---

Body.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	tagsDir := filepath.Join(baseDir, "notes", "by", "tags")
	os.RemoveAll(filepath.Join(tagsDir, "plain"))

	report, err := ReverseRebuild(baseDir)
	if err != nil {
		t.Fatalf("ReverseRebuild() err = %q", err)
	}

	if len(report.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(report.Changes))
	}

	tc := report.Changes[0]
	if tc.ID != "20260328-1" {
		t.Errorf("change ID = %q, want %q", tc.ID, "20260328-1")
	}
	want := []string{"foo/bar"}
	if diff := cmp.Diff(want, tc.NewTags); diff != "" {
		t.Errorf("new tags diff (-want, +got):\n%s", diff)
	}
}

func TestReverseRebuildRemoveAllTags(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo, bar
---

Body.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	os.RemoveAll(filepath.Join(baseDir, "notes", "by", "tags"))

	report, err := ReverseRebuild(baseDir)
	if err != nil {
		t.Fatalf("ReverseRebuild() err = %q", err)
	}

	if len(report.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(report.Changes))
	}

	tc := report.Changes[0]
	if len(tc.NewTags) != 0 {
		t.Errorf("expected empty new tags, got %v", tc.NewTags)
	}
}

func TestExecuteReverseRebuild(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo/bar
---

Body.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	tagsDir := filepath.Join(baseDir, "notes", "by", "tags")
	newDir := filepath.Join(tagsDir, "new-tag")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "20260328-1-hello.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := ReverseRebuild(baseDir)
	if err != nil {
		t.Fatalf("ReverseRebuild() err = %q", err)
	}

	if err := ExecuteReverseRebuild(baseDir, report.Changes); err != nil {
		t.Fatalf("ExecuteReverseRebuild() err = %q", err)
	}

	content, err := os.ReadFile(filepath.Join(idDir, "20260328-1-hello.md"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "foo/bar") {
		t.Error("file should still contain foo/bar tag")
	}
	if !strings.Contains(string(content), "new-tag") {
		t.Error("file should contain new-tag")
	}
}

func TestExecuteReverseRebuildRemovesAllTags(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo
---

Body.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	os.RemoveAll(filepath.Join(baseDir, "notes", "by", "tags"))

	report, err := ReverseRebuild(baseDir)
	if err != nil {
		t.Fatalf("ReverseRebuild() err = %q", err)
	}

	if err := ExecuteReverseRebuild(baseDir, report.Changes); err != nil {
		t.Fatalf("ExecuteReverseRebuild() err = %q", err)
	}

	content, err := os.ReadFile(filepath.Join(idDir, "20260328-1-hello.md"))
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "tags:") {
		t.Error("file should not contain tags field after removing all tags")
	}
}

func TestReconcileTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		fromFS   []string
		want     []string
	}{
		{
			name:     "both empty",
			existing: nil,
			fromFS:   nil,
			want:     nil,
		},
		{
			name:     "existing only",
			existing: []string{"a", "b"},
			fromFS:   []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "new tags from fs",
			existing: []string{"a"},
			fromFS:   []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "tags removed from fs",
			existing: []string{"a", "b"},
			fromFS:   []string{"a"},
			want:     []string{"a"},
		},
		{
			name:     "order preserved existing first new appended",
			existing: []string{"a"},
			fromFS:   []string{"b", "a"},
			want:     []string{"a", "b"},
		},
		{
			name:     "duplicates in existing and fs deduped",
			existing: []string{"a", "a"},
			fromFS:   []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "all tags removed",
			existing: []string{"a", "b"},
			fromFS:   nil,
			want:     nil,
		},
		{
			name:     "all tags new from fs",
			existing: nil,
			fromFS:   []string{"x", "y"},
			want:     []string{"x", "y"},
		},
		{
			name:     "duplicates in fs deduped",
			existing: []string{"a"},
			fromFS:   []string{"b", "a"},
			want:     []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconcileTags(tt.existing, tt.fromFS)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("reconcileTags() diff (-want, +got):\n%s", diff)
			}
		})
	}
}
