package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	gonotes "github.com/marcelbeumer/gonotes"
)

const usage = `Usage: gonotes2 <command> [flags]

Commands:
  prepare    Prepare a note: merge frontmatter fields, output to stdout
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "prepare":
		if err := runPrepare(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
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

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gonotes2 prepare [flags] [-]

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

	note, err := gonotes.Prepare(r, opts)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(os.Stdout, note.Markdown())
	return err
}
