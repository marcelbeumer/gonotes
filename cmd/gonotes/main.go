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
  update     Update an existing note
  folder     Create a new folder for file storage
  rebuild    Scan notes, report issues, rename files, rebuild symlinks
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
	case "update":
		err = runUpdate(os.Args[2:])
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
	var tagMatches stringSliceFlag
	var tagReplaces stringSliceFlag
	var extraKeys stringSliceFlag
	var extraValues stringSliceFlag
	fs.Var(&tagMatches, "Tm", "tag regex match (repeatable; pair with -Tr)")
	fs.Var(&tagReplaces, "Tr", "tag regex replace (repeatable; pair with -Tm)")
	fs.Var(&extraKeys, "Fk", "set custom frontmatter key (repeatable; pair with -Fv)")
	fs.Var(&extraValues, "Fv", "set custom frontmatter value (repeatable; pair with -Fk)")
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
	if len(tagMatches) != len(tagReplaces) {
		return fmt.Errorf("-Tm and -Tr must be provided in equal counts")
	}
	if len(extraKeys) != len(extraValues) {
		return fmt.Errorf("-Fk and -Fv must be provided in equal counts")
	}

	tagRewrites := make([]gonotes.TagRewrite, len(tagMatches))
	for i := range tagMatches {
		tagRewrites[i] = gonotes.TagRewrite{Match: tagMatches[i], Replace: tagReplaces[i]}
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

	// Build prepare options; only set pointers for explicitly provided flags.
	opts := gonotes.PrepareOptions{
		TagRewrites:      tagRewrites,
		ExtraFrontmatter: extraFM,
	}
	providedTags := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "t":
			opts.Title = title
		case "T":
			opts.Tags = tags
			providedTags = true
		case "d":
			opts.Date = date
		}
	})
	if providedTags && len(tagMatches) > 0 {
		return fmt.Errorf("cannot combine -T with -Tm/-Tr")
	}

	if *output != "md" && *output != "json" {
		return fmt.Errorf("unknown output format: %q (use md or json)", *output)
	}

	// Use cwd as the base directory.
	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	note, plan, err := gonotes.CreateNote(baseDir, r, opts, *dryRun)
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

func runUpdate(args []string) error {
	readStdin := false
	var flagArgs []string
	for _, a := range args {
		if a == "-" {
			readStdin = true
		} else {
			flagArgs = append(flagArgs, a)
		}
	}

	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	title := fs.String("t", "", "set title")
	tags := fs.String("T", "", "set tags (comma-separated)")
	date := fs.String("d", "", "set date")
	id := fs.String("i", "", "target note by id (yyyymmdd-N, optional slug suffix)")
	file := fs.String("f", "", "target note by file path")
	all := fs.Bool("a", false, "target all notes under notes/by/id")
	var tagMatches stringSliceFlag
	var tagReplaces stringSliceFlag
	var extraKeys stringSliceFlag
	var extraValues stringSliceFlag
	fs.Var(&tagMatches, "Tm", "tag regex match (repeatable; pair with -Tr)")
	fs.Var(&tagReplaces, "Tr", "tag regex replace (repeatable; pair with -Tm)")
	fs.Var(&extraKeys, "Fk", "set custom frontmatter key (repeatable; pair with -Fv)")
	fs.Var(&extraValues, "Fv", "set custom frontmatter value (repeatable; pair with -Fk)")
	output := fs.String("o", "md", "output format for dry run: md or json")
	dryRun := fs.Bool("n", false, "dry run: print prepared note and plan, don't write")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gonotes update [flags] [-]

Update existing notes.

Target selectors (exactly one):
  -i id       target by note id (yyyymmdd-N, optional slug suffix)
  -f file     target by file path
  -           read from stdin (always dry run)
  -a          target all notes under notes/by/id/

Mutations (at least one):
  -t title
  -T tags
  -d date
  -Tm/-Tr     tag regex rewrite pairs (repeatable)
  -Fk/-Fv     custom frontmatter key/value pairs (repeatable)

Rules:
  - cannot combine -T with -Tm/-Tr
  - with -a, -t is not allowed

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	selectorCount := 0
	if *id != "" {
		selectorCount++
	}
	if *file != "" {
		selectorCount++
	}
	if readStdin {
		selectorCount++
	}
	if *all {
		selectorCount++
	}
	if selectorCount != 1 {
		return fmt.Errorf("exactly one target selector is required: -i, -f, -, or -a")
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	if len(tagMatches) != len(tagReplaces) {
		return fmt.Errorf("-Tm and -Tr must be provided in equal counts")
	}
	if len(extraKeys) != len(extraValues) {
		return fmt.Errorf("-Fk and -Fv must be provided in equal counts")
	}
	if provided["T"] && len(tagMatches) > 0 {
		return fmt.Errorf("cannot combine -T with -Tm/-Tr")
	}
	if *all && provided["t"] {
		return fmt.Errorf("-t is not allowed with -a")
	}

	mutationCount := 0
	if provided["t"] {
		mutationCount++
	}
	if provided["T"] {
		mutationCount++
	}
	if provided["d"] {
		mutationCount++
	}
	if len(tagMatches) > 0 {
		mutationCount++
	}
	if len(extraKeys) > 0 {
		mutationCount++
	}
	if mutationCount == 0 {
		return fmt.Errorf("at least one mutation is required: -t, -T, -d, -Tm/-Tr, or -Fk/-Fv")
	}

	if readStdin {
		*dryRun = true
	}
	if *output != "md" && *output != "json" {
		return fmt.Errorf("unknown output format: %q (use md or json)", *output)
	}

	tagRewrites := make([]gonotes.TagRewrite, len(tagMatches))
	for i := range tagMatches {
		tagRewrites[i] = gonotes.TagRewrite{Match: tagMatches[i], Replace: tagReplaces[i]}
	}
	extraFM := make([]gonotes.FrontmatterField, len(extraKeys))
	for i := range extraKeys {
		extraFM[i] = gonotes.FrontmatterField{Key: extraKeys[i], Value: extraValues[i]}
	}

	opts := gonotes.PrepareOptions{
		TagRewrites:      tagRewrites,
		ExtraFrontmatter: extraFM,
	}
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

	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if readStdin {
		note, err := gonotes.Prepare(os.Stdin, opts)
		if err != nil {
			return err
		}
		return printDryRunNote(note, *output)
	}

	if *all {
		paths, err := gonotes.ListCanonicalNotePaths(baseDir)
		if err != nil {
			return err
		}
		changed := 0
		for _, p := range paths {
			res, err := gonotes.UpdateNoteFile(baseDir, p, opts, *dryRun)
			if err != nil {
				return err
			}
			if res.Changed {
				changed++
			}
		}
		if !*dryRun && changed > 0 {
			if err := gonotes.RebuildSymlinks(baseDir); err != nil {
				return err
			}
		}
		fmt.Fprintf(os.Stderr, "notes: %d changed, %d unchanged\n", changed, len(paths)-changed)
		return nil
	}

	targetPath := *file
	if *id != "" {
		path, err := gonotes.ResolveNotePathByID(baseDir, *id)
		if err != nil {
			return err
		}
		targetPath = path
	} else {
		targetPath = resolveUpdateFilePath(baseDir, targetPath)
	}

	res, err := gonotes.UpdateNoteFile(baseDir, targetPath, opts, *dryRun)
	if err != nil {
		return err
	}

	if *dryRun {
		if err := printDryRunNote(res.Note, *output); err != nil {
			return err
		}
		fmt.Fprint(os.Stderr, res.Plan.String())
		return nil
	}

	if res.Changed {
		if err := gonotes.RebuildSymlinks(baseDir); err != nil {
			return err
		}
	}
	fmt.Fprintln(os.Stdout, res.NewPath)
	return nil
}

func printDryRunNote(note *gonotes.Note, output string) error {
	switch output {
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
	return nil
}

func resolveUpdateFilePath(baseDir, fileArg string) string {
	if filepath.IsAbs(fileArg) {
		return fileArg
	}
	idPath := filepath.Join(baseDir, "notes", "by", "id", fileArg)
	if _, err := os.Stat(idPath); err == nil {
		return idPath
	}
	return filepath.Join(baseDir, fileArg)
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
	report, err := gonotes.ScanNotes(baseDir)
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

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}
