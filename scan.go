package gonotes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func matchesAny(target string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, target); matched {
			return true
		}
	}
	return false
}

type BrokenLink struct {
	SourceID string
	TargetID string
}

type Rename struct {
	OldName string
	NewName string
}

type ScanError struct {
	Filename string
	Message  string
}

type RebuildReport struct {
	BrokenLinks []BrokenLink
	Renames     []Rename
	Errors      []ScanError
}

func (r *RebuildReport) String() string {
	var b strings.Builder

	if len(r.BrokenLinks) > 0 {
		fmt.Fprintf(&b, "Broken links (%d):\n", len(r.BrokenLinks))
		for _, bl := range r.BrokenLinks {
			fmt.Fprintf(&b, "  %s -> %s\n", bl.SourceID, bl.TargetID)
		}
	}

	if len(r.Renames) > 0 {
		fmt.Fprintf(&b, "Renames (%d):\n", len(r.Renames))
		for _, rn := range r.Renames {
			fmt.Fprintf(&b, "  %s -> %s\n", rn.OldName, rn.NewName)
		}
	}

	if len(r.Errors) > 0 {
		fmt.Fprintf(&b, "Errors (%d):\n", len(r.Errors))
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "  %s: %s\n", e.Filename, e.Message)
		}
	}

	if len(r.BrokenLinks) == 0 && len(r.Renames) == 0 && len(r.Errors) == 0 {
		b.WriteString("No issues found.\n")
	}

	return b.String()
}

func ScanNotes(baseDir string) (*RebuildReport, error) {
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	filesDir := filepath.Join(baseDir, "files")

	files, readErrs, err := readNoteFiles(idDir)
	if err != nil {
		return nil, fmt.Errorf("scan notes: %w", err)
	}

	type noteInfo struct {
		id            string
		currentName   string
		correctName   string
		internalLinks []string
		ignoreLinks   []string
	}

	var infos []noteInfo
	var scanErrors []ScanError
	maxNums := map[string]int{}
	idSet := make(map[string]struct{})

	for i := range files {
		nf := &files[i]
		name := nf.Filename
		id, parsed := IDFromFilename(name)

		correctName := name
		if parsed {
			correctName = NoteFilename(id, nf.Slug)
		} else if !nf.Date.IsZero() {
			prefix := idPrefix(nf.Date)
			if _, ok := maxNums[prefix]; !ok {
				maxNum, err := MaxNumFromDir(idDir, nf.Date)
				if err != nil {
					return nil, fmt.Errorf("max num from dir: %w", err)
				}
				maxNums[prefix] = maxNum
			}
			num := maxNums[prefix] + 1
			newID := fmtID(prefix, num)
			correctName = NoteFilename(newID, nf.Slug)
			maxNums[prefix] = num
		} else {
			scanErrors = append(scanErrors, ScanError{
				Filename: name,
				Message:  "cannot determine note ID (no parseable ID and no date)",
			})
			continue
		}

		if _, exists := idSet[id]; exists {
			scanErrors = append(scanErrors, ScanError{
				Filename: name,
				Message:  fmt.Sprintf("duplicate note ID %q", id),
			})
			continue
		}
		idSet[id] = struct{}{}

		infos = append(infos, noteInfo{
			id:            id,
			currentName:   name,
			correctName:   correctName,
			internalLinks: nf.InternalLinks,
			ignoreLinks:   nf.IgnoreLinks,
		})
	}

	scanErrors = append(scanErrors, readErrs...)

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].id < infos[j].id
	})

	report := &RebuildReport{
		Errors: scanErrors,
	}

	for _, n := range infos {
		for _, target := range n.internalLinks {
			if matchesAny(target, n.ignoreLinks) {
				continue
			}
			targetID := target
			if !strings.Contains(target, "/") {
				if m := reIDPrefix.FindStringSubmatch(target); m != nil {
					targetID = m[1] + "-" + m[2]
				}
			}
			if _, exists := idSet[targetID]; exists {
				continue
			}
			filePath := filepath.Join(filesDir, target)
			if _, err := os.Stat(filePath); err == nil {
				continue
			}
			report.BrokenLinks = append(report.BrokenLinks, BrokenLink{
				SourceID: n.id,
				TargetID: target,
			})
		}

		if n.currentName != n.correctName {
			report.Renames = append(report.Renames, Rename{
				OldName: n.currentName,
				NewName: n.correctName,
			})
		}
	}

	return report, nil
}

func ExecuteRenames(idDir string, renames []Rename) error {
	for _, rn := range renames {
		oldPath := filepath.Join(idDir, rn.OldName)
		newPath := filepath.Join(idDir, rn.NewName)
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("rename %s -> %s: %w", rn.OldName, rn.NewName, err)
		}
	}
	return nil
}

func RebuildSymlinks(baseDir string) error {
	byDir := filepath.Join(baseDir, "notes", "by")
	idDir := filepath.Join(byDir, "id")

	for _, dir := range []string{"date", "tags"} {
		p := filepath.Join(byDir, dir)
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("rebuild symlinks: remove %s: %w", dir, err)
		}
	}

	files, _, err := readNoteFiles(idDir)
	if err != nil {
		return fmt.Errorf("rebuild symlinks: %w", err)
	}

	for i := range files {
		nf := &files[i]
		links := linkEntries(&nf.Note, nf.Filename)
		plan := &Plan{Links: links}
		if err := plan.CreateLinks(baseDir); err != nil {
			return fmt.Errorf("rebuild symlinks: %w", err)
		}
	}

	return nil
}
