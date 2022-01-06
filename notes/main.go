package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/marcelbeumer/notes-in-go/notes/internal/note"
	"github.com/marcelbeumer/notes-in-go/notes/internal/repo"
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

	syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)

	renameTagCmd := flag.NewFlagSet("rename-tag", flag.ExitOnError)
	renameFrom := renameTagCmd.String("from", "", "Where to rename from, ex: foo/bar")
	renameTo := renameTagCmd.String("to", "", "Where to rename to, ex: something/else")

	if len(os.Args) < 2 {
		printHelpAndExit()
	}

	r := repo.New()

	switch os.Args[1] {
	case "new":
		newCmd.Parse(os.Args[2:])
		n := note.New()
		if newTitle != nil && *newTitle != "" {
			n.Title = *&newTitle
		}
		if newHref != nil && *newHref != "" {
			n.Href = *&newHref
		}
		if newTags != nil && len(*newTags) > 0 {
			n.Tags = *newTags
		}
		r.AddNote(n)
		err := r.Sync(true)
		if err != nil {
			errAndExit(err)
		}
		notePath, err := r.PathIfStored(n)
		if err != nil {
			errAndExit(err)
		}
		fmt.Println(notePath)
	case "sync":
		syncCmd.Parse(os.Args[2:])
		err := r.LoadNotes()
		if err != nil {
			errAndExit(err)
		}
		err = r.Sync(false)
		if err != nil {
			errAndExit(err)
		}
	case "rename-tag":
		renameTagCmd.Parse(os.Args[2:])
		err := r.LoadNotes()
		if err != nil {
			errAndExit(err)
		}
		from := ""
		to := ""
		if renameFrom != nil {
			from = *renameFrom
		} else {
			errAndExit(errors.New("Please provide \"from\""))
		}
		if renameTo != nil {
			to = *renameTo
		} else {
			errAndExit(errors.New("Please provide \"to\""))
		}
		for _, note := range r.Notes() {
			note.RenameTag(from, to)
		}
		err = r.Sync(false)
		if err != nil {
			errAndExit(err)
		}
	default:
		printHelpAndExit()
	}
}
