package gonotes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanNotes(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
---

See [[20260328-2]] and [[20260328-99]].`)

	writeTestNote(t, idDir, "20260328-2-old-name.md", `---
title: New Name
date: 2026-03-28 15:00:00
---

Links to [[20260328-1]].`)

	writeTestNote(t, idDir, "20260328-3.md", `---
date: 2026-03-28 16:00:00
---

Plain note.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 1 {
		t.Errorf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	} else {
		bl := report.BrokenLinks[0]
		if bl.SourceID != "20260328-1" || bl.TargetID != "20260328-99" {
			t.Errorf("broken link = %v, want {20260328-1 -> 20260328-99}", bl)
		}
	}

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

	report, err := ScanNotes(baseDir)
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

func TestScanNotesFileLinks(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	filesDir := filepath.Join(baseDir, "files")

	folderPath := filepath.Join(filesDir, "20260403-1-contract-pdfs")
	if err := os.MkdirAll(folderPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folderPath, "doc1.pdf"), []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeTestNote(t, idDir, "20260403-1-hello.md", `---
title: Hello
date: 2026-04-03 10:00:00
---

See [[20260403-1-contract-pdfs/doc1.pdf]] and [[20260403-1-contract-pdfs/bad.pdf]].`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 1 {
		t.Fatalf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	}
	bl := report.BrokenLinks[0]
	if bl.SourceID != "20260403-1" {
		t.Errorf("broken link source = %q, want %q", bl.SourceID, "20260403-1")
	}
	if bl.TargetID != "20260403-1-contract-pdfs/bad.pdf" {
		t.Errorf("broken link target = %q, want %q", bl.TargetID, "20260403-1-contract-pdfs/bad.pdf")
	}
}

func TestScanNotesFileLinksNoFilesDir(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260403-1-hello.md", `---
title: Hello
date: 2026-04-03 10:00:00
---

See [[20260403-1-docs/readme.txt]].`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 1 {
		t.Fatalf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	}
}

func TestScanNotesFileLinkAndNoteLink(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	filesDir := filepath.Join(baseDir, "files")

	folderPath := filepath.Join(filesDir, "20260403-1-docs")
	if err := os.MkdirAll(folderPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folderPath, "spec.pdf"), []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeTestNote(t, idDir, "20260403-1-hello.md", `---
title: Hello
date: 2026-04-03 10:00:00
---

See [[20260403-2]] and [[20260403-1-docs/spec.pdf]].`)

	writeTestNote(t, idDir, "20260403-2-world.md", `---
title: World
date: 2026-04-03 11:00:00
---

Hello.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 0 {
		t.Errorf("expected 0 broken links, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	}
}

func TestScanNotesIgnoreLinksExact(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260403-1-hello.md", `---
title: Hello
date: 2026-04-03 10:00:00
ignore-links: 20260403-99
---

See [[20260403-99]] and [[20260403-88]].`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 1 {
		t.Fatalf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	}
	if report.BrokenLinks[0].TargetID != "20260403-88" {
		t.Errorf("broken link target = %q, want %q", report.BrokenLinks[0].TargetID, "20260403-88")
	}
}

func TestScanNotesIgnoreLinksGlob(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260403-1-hello.md", `---
title: Hello
date: 2026-04-03 10:00:00
ignore-links: some-folder/*
---

See [[some-folder/doc.pdf]] and [[other-folder/doc.pdf]].`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 1 {
		t.Fatalf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	}
	if report.BrokenLinks[0].TargetID != "other-folder/doc.pdf" {
		t.Errorf("broken link target = %q, want %q", report.BrokenLinks[0].TargetID, "other-folder/doc.pdf")
	}
}

func TestScanNotesIgnoreLinksAll(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260403-1-hello.md", `---
title: Hello
date: 2026-04-03 10:00:00
ignore-links: 20260403-99, 20260403-88
---

See [[20260403-99]] and [[20260403-88]].`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 0 {
		t.Errorf("expected 0 broken links, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	}
}

func TestScanNotesOldFormat(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "2026-02-12-2233-05-pacman-cheatsheet.md", `---
title: Pacman Cheatsheet
date: 2026-02-12 22:33:05
tags: linux
---

Pacman tips.`)

	writeTestNote(t, idDir, "2026-01-05-1200-00-old-title.md", `---
title: New Title
date: 2026-01-05 12:00:00
---

Some content.`)

	writeTestNote(t, idDir, "20260328-1-linker.md", `---
title: Linker
date: 2026-03-28 14:30:00
---

See [[2026-02-12-2233-05-pacman-cheatsheet]] and [[9999-99-99]].`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 1 {
		t.Errorf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	} else if report.BrokenLinks[0].TargetID != "9999-99-99" {
		t.Errorf("broken link target = %q, want %q", report.BrokenLinks[0].TargetID, "9999-99-99")
	}

	if len(report.Renames) != 2 {
		t.Errorf("expected 2 renames, got %d: %v", len(report.Renames), report.Renames)
	}

	renameMap := map[string]string{}
	for _, rn := range report.Renames {
		renameMap[rn.OldName] = rn.NewName
	}
	if got, ok := renameMap["2026-02-12-2233-05-pacman-cheatsheet.md"]; !ok {
		t.Error("missing rename for pacman-cheatsheet")
	} else if got != "20260212-1-pacman-cheatsheet.md" {
		t.Errorf("pacman-cheatsheet rename = %q, want %q", got, "20260212-1-pacman-cheatsheet.md")
	}
	if got, ok := renameMap["2026-01-05-1200-00-old-title.md"]; !ok {
		t.Error("missing rename for old-title")
	} else if got != "20260105-1-new-title.md" {
		t.Errorf("old-title rename = %q, want %q", got, "20260105-1-new-title.md")
	}
}

func TestScanNotesNonMatchingFiles(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
---

Links to [[readme]].`)

	writeTestNote(t, idDir, "readme.md", `---
title: Readme
---

Some info.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.BrokenLinks) != 1 {
		t.Errorf("expected 1 broken link, got %d: %v", len(report.BrokenLinks), report.BrokenLinks)
	} else if report.BrokenLinks[0].TargetID != "readme" {
		t.Errorf("broken link target = %q, want %q", report.BrokenLinks[0].TargetID, "readme")
	}

	if len(report.Renames) != 0 {
		t.Errorf("expected 0 renames, got %d: %v", len(report.Renames), report.Renames)
	}

	if len(report.Errors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(report.Errors), report.Errors)
	} else if report.Errors[0].Filename != "readme.md" {
		t.Errorf("error filename = %q, want %q", report.Errors[0].Filename, "readme.md")
	}
}

func TestScanNotesNoTitleStripsSlug(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-some-slug.md", `---
date: 2026-03-28 14:30:00
---

No title here.`)

	report, err := ScanNotes(baseDir)
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

	for _, rn := range renames {
		if _, err := os.Stat(filepath.Join(dir, rn.OldName)); !os.IsNotExist(err) {
			t.Errorf("old file %s still exists", rn.OldName)
		}
	}

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

	staleDir := filepath.Join(baseDir, "notes", "by", "tags", "stale")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Error("stale tag directory still exists after rebuild")
	}

	wantLinks := []struct {
		link   string
		target string
	}{
		{"notes/by/date/2026-03-28/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/date/2026-03-29/20260328-2-world.md", "20260328-2-world.md"},
		{"notes/by/tags/foo/bar/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/tags/plain/20260328-1-hello.md", "20260328-1-hello.md"},
		{"notes/by/tags/other/20260328-2-world.md", "20260328-2-world.md"},
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

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("first RebuildSymlinks() err = %q", err)
	}

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("second RebuildSymlinks() err = %q", err)
	}

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

	writeTestNote(t, idDir, "2026-02-12-2233-05-pacman-cheatsheet.md", `---
title: Pacman Cheatsheet
date: 2026-02-12 22:33:05
tags: linux/pacman
---

Tips.`)

	writeTestNote(t, idDir, "readme.md", `---
title: Readme
date: 2026-01-01 00:00:00
tags: meta
---

Info.`)

	if err := RebuildSymlinks(baseDir); err != nil {
		t.Fatalf("RebuildSymlinks() err = %q", err)
	}

	wantLinks := []struct {
		link   string
		target string
	}{
		{"notes/by/date/2026-02-12/2026-02-12-2233-05-pacman-cheatsheet.md", "2026-02-12-2233-05-pacman-cheatsheet.md"},
		{"notes/by/tags/linux/pacman/2026-02-12-2233-05-pacman-cheatsheet.md", "2026-02-12-2233-05-pacman-cheatsheet.md"},
		{"notes/by/date/2026-01-01/readme.md", "readme.md"},
		{"notes/by/tags/meta/readme.md", "readme.md"},
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

	t.Run("with errors", func(t *testing.T) {
		r := &RebuildReport{
			Errors: []ScanError{{Filename: "readme.md", Message: "cannot determine note ID (no parseable ID and no date)"}},
		}
		got := r.String()
		if !strings.Contains(got, "Errors (1)") {
			t.Errorf("String() missing errors header, got:\n%s", got)
		}
		if !strings.Contains(got, "readme.md: cannot determine note ID") {
			t.Errorf("String() missing error entry, got:\n%s", got)
		}
		if strings.Contains(got, "No issues found") {
			t.Errorf("String() should not say 'No issues found' when there are errors")
		}
	})
}

func TestScanNotesOldFormatWithDate(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "2026-03-28-1430-00-my-note.md", `---
title: My Note
date: 2026-03-28 14:30:00
---

Content.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.Renames) != 1 {
		t.Fatalf("expected 1 rename, got %d: %v", len(report.Renames), report.Renames)
	}

	rn := report.Renames[0]
	if rn.OldName != "2026-03-28-1430-00-my-note.md" {
		t.Errorf("OldName = %q, want %q", rn.OldName, "2026-03-28-1430-00-my-note.md")
	}
	if rn.NewName != "20260328-1-my-note.md" {
		t.Errorf("NewName = %q, want %q", rn.NewName, "20260328-1-my-note.md")
	}
}

func TestScanNotesOldFormatMultipleSameDate(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "2026-03-28-1000-00-first.md", `---
title: First
date: 2026-03-28 10:00:00
---

First note.`)

	writeTestNote(t, idDir, "2026-03-28-1430-00-second.md", `---
title: Second
date: 2026-03-28 14:30:00
---

Second note.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.Renames) != 2 {
		t.Fatalf("expected 2 renames, got %d: %v", len(report.Renames), report.Renames)
	}

	newNames := map[string]bool{}
	for _, rn := range report.Renames {
		newNames[rn.NewName] = true
	}

	want1 := "20260328-1"
	want2 := "20260328-2"
	found1, found2 := false, false
	for _, rn := range report.Renames {
		if strings.HasPrefix(rn.NewName, want1+"-") {
			found1 = true
		}
		if strings.HasPrefix(rn.NewName, want2+"-") {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("expected renames with IDs %s and %s, got %v", want1, want2, report.Renames)
	}
}

func TestScanNotesOldFormatNoDate(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "2026-03-28-1430-00-no-date.md", `---
title: No Date Note
---

Content without date.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.Renames) != 0 {
		t.Errorf("expected 0 renames, got %d: %v", len(report.Renames), report.Renames)
	}

	if len(report.Errors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(report.Errors), report.Errors)
	} else if report.Errors[0].Filename != "2026-03-28-1430-00-no-date.md" {
		t.Errorf("error filename = %q, want %q", report.Errors[0].Filename, "2026-03-28-1430-00-no-date.md")
	}
}

func TestScanNotesCollectsErrors(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-hello.md", `---
title: Hello
date: 2026-03-28 14:30:00
---

Body.`)

	writeTestNote(t, idDir, "2026-01-05-1200-00-has-date.md", `---
title: Has Date
date: 2026-01-05 12:00:00
---

Content.`)

	writeTestNote(t, idDir, "readme.md", `---
title: Readme
---

Info.`)

	writeTestNote(t, idDir, "2026-02-12-2233-05-no-date.md", `---
title: No Date
---

Tips.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	if len(report.Renames) != 1 {
		t.Errorf("expected 1 rename, got %d: %v", len(report.Renames), report.Renames)
	} else if report.Renames[0].OldName != "2026-01-05-1200-00-has-date.md" {
		t.Errorf("rename old = %q, want %q", report.Renames[0].OldName, "2026-01-05-1200-00-has-date.md")
	}

	if len(report.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(report.Errors), report.Errors)
	}
	errFiles := map[string]bool{}
	for _, e := range report.Errors {
		errFiles[e.Filename] = true
	}
	if !errFiles["readme.md"] {
		t.Error("expected error for readme.md")
	}
	if !errFiles["2026-02-12-2233-05-no-date.md"] {
		t.Error("expected error for 2026-02-12-2233-05-no-date.md")
	}
}

func TestScanNotesDuplicateIDs(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-foo.md", `---
title: Foo
date: 2026-03-28 14:30:00
---

Foo content.`)

	writeTestNote(t, idDir, "20260328-1-bar.md", `---
title: Bar
date: 2026-03-28 15:00:00
---

Bar content.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	dupErrors := []ScanError{}
	for _, e := range report.Errors {
		if strings.Contains(e.Message, "duplicate note ID") {
			dupErrors = append(dupErrors, e)
		}
	}
	if len(dupErrors) != 1 {
		t.Fatalf("expected 1 duplicate error, got %d: %v", len(dupErrors), dupErrors)
	}
	if !strings.Contains(dupErrors[0].Message, "20260328-1") {
		t.Errorf("error message should mention the ID, got: %s", dupErrors[0].Message)
	}
}

func TestScanNotesMultipleDuplicateIDs(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")

	writeTestNote(t, idDir, "20260328-1-aaa.md", `---
title: Aaa
date: 2026-03-28 10:00:00
---

Aaa.`)

	writeTestNote(t, idDir, "20260328-1-bbb.md", `---
title: Bbb
date: 2026-03-28 11:00:00
---

Bbb.`)

	writeTestNote(t, idDir, "20260328-1-ccc.md", `---
title: Ccc
date: 2026-03-28 12:00:00
---

Ccc.`)

	report, err := ScanNotes(baseDir)
	if err != nil {
		t.Fatalf("ScanNotes() err = %q", err)
	}

	dupErrors := []ScanError{}
	for _, e := range report.Errors {
		if strings.Contains(e.Message, "duplicate note ID") {
			dupErrors = append(dupErrors, e)
		}
	}
	if len(dupErrors) != 2 {
		t.Fatalf("expected 2 duplicate errors, got %d: %v", len(dupErrors), dupErrors)
	}
	for _, e := range dupErrors {
		if !strings.Contains(e.Message, "20260328-1") {
			t.Errorf("error message should mention the ID, got: %s", e.Message)
		}
	}
}