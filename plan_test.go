package gonotes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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

	gotPaths := make([]string, len(plan.Links))
	for i, l := range plan.Links {
		gotPaths[i] = l.Path
	}

	wantPaths := []string{
		filepath.Join("notes", "by", "date", "2026-03-28", "20260328-1-hello-world.md"),
		filepath.Join("notes", "by", "tags", "foo", "bar", "20260328-1-hello-world.md"),
		filepath.Join("notes", "by", "tags", "plain", "20260328-1-hello-world.md"),
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

	if len(plan.Links) != 1 {
		t.Errorf("expected 1 link (date only), got %d", len(plan.Links))
	}
	if !strings.Contains(plan.Links[0].Path, "date") {
		t.Errorf("expected date link, got %s", plan.Links[0].Path)
	}
}

func TestNotePlanDuplicateTagComponents(t *testing.T) {
	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: Hello
date: 2026-03-28 14:30:00
tags: welcome/here, welcome/there, foo/bar/x, foo/bar/y
---`))
	if err != nil {
		t.Fatal(err)
	}

	plan := NotePlan(note)

	seen := map[string]bool{}
	for _, l := range plan.Links {
		if seen[l.Path] {
			t.Errorf("duplicate link path: %s", l.Path)
		}
		seen[l.Path] = true
	}

	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(idDir, "20260328-1-hello.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := plan.CreateLinks(baseDir); err != nil {
		t.Fatalf("CreateLinks() err = %q", err)
	}
}

func TestPlanCreateLinksCreatesSymlinks(t *testing.T) {
	baseDir := t.TempDir()
	filename := "20260328-1-hello-world.md"

	idDir := filepath.Join(baseDir, "notes", "by", "id")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(idDir, filename)
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &Plan{
		Links: []Link{
			{
				Path:   filepath.Join("notes", "by", "date", "2026-03-28", filename),
				Target: filepath.Join("..", "..", "id", filename),
			},
			{
				Path:   filepath.Join("notes", "by", "tags", "foo", "bar", filename),
				Target: filepath.Join("..", "..", "..", "id", filename),
			},
			{
				Path:   filepath.Join("notes", "by", "tags", "plain", filename),
				Target: filepath.Join("..", "..", "id", filename),
			},
		},
	}

	if err := plan.CreateLinks(baseDir); err != nil {
		t.Fatalf("CreateLinks() err = %q", err)
	}

	for _, l := range plan.Links {
		absLink := filepath.Join(baseDir, l.Path)

		info, err := os.Lstat(absLink)
		if err != nil {
			t.Errorf("symlink %s does not exist: %v", l.Path, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", l.Path)
			continue
		}

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
		Links: []Link{
			{Path: "notes/by/date/2026-03-28/20260328-1-hello.md", Target: "../../id/20260328-1-hello.md"},
		},
	}

	got := plan.String()
	if !strings.Contains(got, "link:  notes/by/date/2026-03-28/20260328-1-hello.md -> ../../id/20260328-1-hello.md") {
		t.Errorf("String() missing link line, got:\n%s", got)
	}
}
