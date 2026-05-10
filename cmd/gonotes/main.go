package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marcelbeumer/gonotes"
)

const usage = `Usage: gonotes <command> [flags]

Commands:
  new        Create a new note
  folder     Create a new folder for file storage
  rebuild    Scan notes, report issues, rename files, rebuild symlinks
             Use -r for reverse rebuild: sync tags from filesystem into notes
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "new":
		err = runNew(os.Args[2:])
	case "folder":
		err = runFolder(os.Args[2:])
	case "rebuild":
		err = runRebuild(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runNew(args []string) error {
	readStdin := false
	var flagArgs []string
	for _, a := range args {
		if a == "-" {
			readStdin = true
		} else {
			flagArgs = append(flagArgs, a)
		}
	}

	fs := flag.NewFlagSet("new", flag.ContinueOnError)

	title := fs.String("t", "", "set title")
	var tags stringSliceFlag
	file := fs.String("f", "", "read note from file")
	var extraKeys stringSliceFlag
	var extraValues stringSliceFlag
	fs.Var(&tags, "T", "add tags (repeatable; comma or space separated)")
	fs.Var(&extraKeys, "Fk", "set custom frontmatter key (repeatable; pair with -Fv)")
	fs.Var(&extraValues, "Fv", "set custom frontmatter value (repeatable; pair with -Fk)")
	dryRun := fs.Bool("n", false, "dry run: print prepared note and plan, don't write")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gonotes new [flags] [-]

Create a new note. Writes the note to notes/by/id/ and creates symlinks
under notes/by/date/ and notes/by/tags/.

Input sources (at most one):
  -           read from stdin
  -f file     read from file
  (none)      start with an empty note

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if readStdin && *file != "" {
		return fmt.Errorf("cannot use both stdin (-) and -f")
	}
	if len(extraKeys) != len(extraValues) {
		return fmt.Errorf("-Fk and -Fv must be provided in equal counts")
	}

	allTags := make([]string, 0)
	for _, t := range tags {
		allTags = append(allTags, gonotes.ParseTags(t)...)
	}
	extraFM := make([]gonotes.FrontmatterField, len(extraKeys))
	for i := range extraKeys {
		extraFM[i] = gonotes.FrontmatterField{Key: extraKeys[i], Value: extraValues[i]}
	}

	var r io.Reader
	switch {
	case readStdin:
		r = os.Stdin
	case *file != "":
		f, err := os.Open(*file)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}

	opts := gonotes.PrepareOptions{
		Tags:             allTags,
		ExtraFrontmatter: extraFM,
	}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "t":
			opts.Title = title
		}
	})

	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	note, plan, err := gonotes.CreateNote(baseDir, r, opts, *dryRun)
	if err != nil {
		return err
	}

	filename := gonotes.NoteFilename(note.ID, note.Slug)
	writePath := filepath.Join("notes", "by", "id", filename)

	if *dryRun {
		if _, err := fmt.Fprint(os.Stdout, note.Markdown()); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "write: %s\n", writePath)
		fmt.Fprint(os.Stderr, plan.String())
		return nil
	}

	fmt.Fprintln(os.Stdout, filepath.Join(baseDir, writePath))
	return nil
}

func runFolder(args []string) error {
	fs := flag.NewFlagSet("folder", flag.ContinueOnError)
	title := fs.String("t", "", "set title (optional)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gonotes folder [flags]

Create a new folder under files/ for file storage. The folder name follows
the same ID format as notes: yyyymmdd-N-slug.

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	path, err := gonotes.CreateFolder(baseDir, *title, time.Now)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, path)
	return nil
}

func runRebuild(args []string) error {
	fs := flag.NewFlagSet("rebuild", flag.ContinueOnError)
	reverse := fs.Bool("r", false, "reverse rebuild: sync tags from filesystem into note files")
	confirm := fs.Bool("y", false, "skip confirmation prompts")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gonotes rebuild [-y] [-r]

Scan notes/by/id/, report broken links and filename mismatches,
rename files, and rebuild symlink structures.

With -r, scan tags from the symlink structure and update
note frontmatter to match, replacing the normal rebuild flow.

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if *reverse {
		return runReverseRebuild(baseDir, *confirm)
	}

	idDir := filepath.Join(baseDir, "notes", "by", "id")

	report, err := gonotes.ScanNotes(baseDir)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, report.String())

	if len(report.Renames) > 0 {
		if !*confirm && !promptYN("Perform renames?") {
			fmt.Fprintln(os.Stderr, "Skipping renames.")
		} else {
			if err := gonotes.ExecuteRenames(idDir, report.Renames); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Renamed %d file(s).\n", len(report.Renames))
		}
	}

	if !*confirm && !promptYN("Rebuild symlinks?") {
		fmt.Fprintln(os.Stderr, "Skipping symlink rebuild.")
		return nil
	}

	if err := gonotes.RebuildSymlinks(baseDir); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Symlinks rebuilt.")

	return nil
}

func runReverseRebuild(baseDir string, confirm bool) error {
	report, err := gonotes.ReverseRebuild(baseDir)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, report.String())

	if len(report.Changes) == 0 {
		return nil
	}

	if !confirm && !promptYN("Apply tag changes?") {
		fmt.Fprintln(os.Stderr, "Skipping tag changes.")
		return nil
	}

	if err := gonotes.ExecuteReverseRebuild(baseDir, report.Changes); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Updated %d note(s).\n", len(report.Changes))

	return nil
}

var stdinScanner = bufio.NewScanner(os.Stdin)

func promptYN(question string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", question)
	if !stdinScanner.Scan() {
		return false
	}
	ans := strings.TrimSpace(strings.ToLower(stdinScanner.Text()))
	return ans == "y" || ans == "yes"
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}
