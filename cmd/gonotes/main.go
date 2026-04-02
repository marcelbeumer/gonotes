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
  id         Print the next available note ID
  new        Create a new note
  prepare    Prepare a note: merge frontmatter fields, output to stdout
  rebuild    Scan notes, report issues, rename files, rebuild symlinks
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "id":
		err = runID()
	case "new":
		err = runNew(os.Args[2:])
	case "prepare":
		err = runPrepare(os.Args[2:])
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

func runPrepare(args []string) error {
	// Pull out bare "-" (stdin marker) before flag parsing, since flag.Parse
	// stops at the first non-flag argument.
	readStdin := false
	var flagArgs []string
	for _, a := range args {
		if a == "-" {
			readStdin = true
		} else {
			flagArgs = append(flagArgs, a)
		}
	}

	fs := flag.NewFlagSet("prepare", flag.ContinueOnError)

	title := fs.String("t", "", "set title")
	tags := fs.String("T", "", "set tags (comma-separated)")
	date := fs.String("d", "", "set date (default: now)")
	file := fs.String("f", "", "read note from file")
	output := fs.String("o", "md", "output format: md or json")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gonotes prepare [flags] [-]

Read a note, merge frontmatter fields, and write the result to stdout.

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

	// Build options; only set pointers for flags that were explicitly provided.
	opts := gonotes.PrepareOptions{}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "t":
			opts.Title = title
		case "T":
			opts.Tags = tags
		case "d":
			opts.Date = date
		}
	})

	if *output != "md" && *output != "json" {
		return fmt.Errorf("unknown output format: %q (use md or json)", *output)
	}

	note, err := gonotes.Prepare(r, opts)
	if err != nil {
		return err
	}

	switch *output {
	case "json":
		b, err := note.JSON()
		if err != nil {
			return err
		}
		b = append(b, '\n')
		_, err = os.Stdout.Write(b)
		return err
	default:
		_, err = fmt.Fprint(os.Stdout, note.Markdown())
		return err
	}
}

func runNew(args []string) error {
	// Pull out bare "-" (stdin marker) before flag parsing.
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
	tags := fs.String("T", "", "set tags (comma-separated)")
	date := fs.String("d", "", "set date (default: now)")
	file := fs.String("f", "", "read note from file")
	id := fs.String("i", "", "set id (default: generate)")
	output := fs.String("o", "md", "output format for dry run: md or json")
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

	// Build prepare options; only set pointers for explicitly provided flags.
	opts := gonotes.PrepareOptions{}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "t":
			opts.Title = title
		case "T":
			opts.Tags = tags
		case "d":
			opts.Date = date
		}
	})

	if *output != "md" && *output != "json" {
		return fmt.Errorf("unknown output format: %q (use md or json)", *output)
	}

	// Use cwd as the base directory.
	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	note, plan, err := gonotes.CreateNote(baseDir, r, opts, *id, *dryRun)
	if err != nil {
		return err
	}

	if *dryRun {
		switch *output {
		case "json":
			b, err := note.JSON()
			if err != nil {
				return err
			}
			b = append(b, '\n')
			if _, err := os.Stdout.Write(b); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprint(os.Stdout, note.Markdown()); err != nil {
				return err
			}
		}
		fmt.Fprint(os.Stderr, plan.String())
		return nil
	}

	// Normal mode: print the path of the created file.
	fmt.Fprintln(os.Stdout, filepath.Join(baseDir, plan.WritePath))
	return nil
}

func runID() error {
	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	idDir := filepath.Join(baseDir, "notes", "by", "id")
	id, err := gonotes.NextID(idDir, time.Now())
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, id)
	return nil
}

func runRebuild(args []string) error {
	fs := flag.NewFlagSet("rebuild", flag.ContinueOnError)
	confirm := fs.Bool("y", false, "skip confirmation prompts")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gonotes rebuild [-y]

Scan notes/by/id/, report broken links and filename mismatches,
rename files, and rebuild symlink structures.

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

	idDir := filepath.Join(baseDir, "notes", "by", "id")

	// Phase 1: Scan and report.
	report, err := gonotes.ScanNotes(idDir)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, report.String())

	// Phase 2: Renames.
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

	// Phase 3: Rebuild symlinks.
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

var stdinScanner = bufio.NewScanner(os.Stdin)

// promptYN prints a question to stderr and reads a y/n answer from stdin.
func promptYN(question string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", question)
	if !stdinScanner.Scan() {
		return false
	}
	ans := strings.TrimSpace(strings.ToLower(stdinScanner.Text()))
	return ans == "y" || ans == "yes"
}
