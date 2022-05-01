package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/marcelbeumer/gonotes/internal/log"
	"github.com/marcelbeumer/gonotes/internal/note"
	"github.com/marcelbeumer/gonotes/internal/repo"
	"github.com/marcelbeumer/gonotes/internal/scrape"
	flag "github.com/spf13/pflag"
)

func printHelpAndExit() {
	fmt.Println("expected 'new', 'sync' or 'rename-tag' subcommands")
	os.Exit(1)
}

func errAndExit(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func main() {
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	newTitle := newCmd.String("title", "", "Title")
	newHref := newCmd.String("href", "", "Href")
	newTags := newCmd.StringArray("tag", make([]string, 0), "Tags")
	shouldScrape := newCmd.Bool("scrape", false, "Scrape href")

	syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)

	renameTagCmd := flag.NewFlagSet("rename-tag", flag.ExitOnError)
	renameFrom := renameTagCmd.String("from", "", "Where to rename from, ex: foo/bar")
	renameTo := renameTagCmd.String("to", "", "Where to rename to, ex: something/else")

	lastCmd := flag.NewFlagSet("last", flag.ExitOnError)

	r := repo.New()
	err := r.CheckDir()
	if err != nil {
		errAndExit(err)
	}

	if len(os.Args) < 2 {
		printHelpAndExit()
	}

	switch os.Args[1] {
	case "new":
		if err := newCmd.Parse(os.Args[2:]); err != nil {
			errAndExit(err)
		}
		n := note.New()

		if *shouldScrape {
			if newHref == nil || *newHref == "" {
				errAndExit(errors.New("can not scrape without href"))
			}
			scrapeRes, err := scrape.Scrape(*newHref)
			if err != nil {
				errAndExit(err)
			}
			n.Title = scrapeRes.Title
			if scrapeRes.Description != nil {
				n.Contents = *scrapeRes.Description
			}
		}

		if newHref != nil && *newHref != "" {
			n.Href = newHref
		}
		if newTitle != nil && *newTitle != "" {
			n.Title = newTitle
		}
		if newTags != nil && len(*newTags) > 0 {
			n.Tags = *newTags
		}
		r.AddNote(n)
		err := r.Sync(true, true)
		if err != nil {
			errAndExit(err)
		}
		notePath, err := r.PathIfStored(n)
		if err != nil {
			errAndExit(err)
		}
		fmt.Println(notePath)
	case "sync":
		if err := syncCmd.Parse(os.Args[2:]); err != nil {
			errAndExit(err)
		}
		err := r.LoadNotes()
		if err != nil {
			errAndExit(err)
		}
		err = r.Sync(false, false)
		if err != nil {
			errAndExit(err)
		}
	case "rename-tag":
		if err := renameTagCmd.Parse(os.Args[2:]); err != nil {
			errAndExit(err)
		}
		err := r.LoadNotes()
		if err != nil {
			errAndExit(err)
		}
		from := ""
		to := ""
		if renameFrom != nil {
			from = *renameFrom
		} else {
			errAndExit(errors.New("please provide \"from\""))
		}
		if renameTo != nil {
			to = *renameTo
		} else {
			errAndExit(errors.New("please provide \"to\""))
		}
		for _, note := range r.Notes() {
			note.RenameTag(from, to)
		}
		err = r.Sync(false, false)
		if err != nil {
			errAndExit(err)
		}
	case "last":
		if err := lastCmd.Parse(os.Args[2:]); err != nil {
			errAndExit(err)
		}
		if err := r.LoadNotes(); err != nil {
			errAndExit(err)
		}
		last, err := r.LastStoredPath()
		if err != nil {
			errAndExit(err)
		} else {
			log.Stderrln("Found last note")
			fmt.Println(last)
		}
	default:
		printHelpAndExit()
	}
}
