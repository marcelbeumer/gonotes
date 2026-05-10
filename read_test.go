package gonotes

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadNotesFromDir(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
tags: foo/bar
---

Body.`)

	writeTestNote(t, idDir, "20260328-2-world.md", `---
title: World
date: 2026-03-28 15:00:00
---

Another body.`)

	writeTestNote(t, idDir, "20260328-3.md", `---
date: 2026-03-28 16:00:00
---

No title note.`)

	notes, errs, err := readNotesFromDir(idDir)
	if err != nil {
		t.Fatalf("readNotesFromDir() err = %q", err)
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	if len(notes) != 3 {
		t.Fatalf("got %d notes, want 3", len(notes))
	}

	gotTitles := make([]string, len(notes))
	for i, n := range notes {
		gotTitles[i] = n.Title
	}
	sort.Strings(gotTitles)
	wantTitles := []string{"", "Hello", "World"}
	if diff := cmp.Diff(wantTitles, gotTitles); diff != "" {
		t.Errorf("titles diff (-want, +got):\n%s", diff)
	}
}
