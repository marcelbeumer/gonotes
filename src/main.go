package main

import (
	"fmt"

	flag "github.com/spf13/pflag"
	"marcelbeumer.com/notes/note"
	"marcelbeumer.com/notes/repo"
)

var tags = flag.StringArrayP("tags", "t", make([]string, 0), "help message for flagname")

func main() {
	flag.Parse()
	repo := repo.New()
	fmt.Println(tags)
	if err := repo.LoadNotes(); err != nil {
		panic(err)
	}
	n := note.New()
	title := "Example title"
	n.Title = &title
	repo.AddNote(n)
	if err := repo.Sync(false); err != nil {
		panic(err)
	}
	// notes := repo.Notes()
	// fmt.Printf("%#q", notes)

	// _, err := note.FromPath("notes/2022-01/2022-01-04-2155-26-example-one.md")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("%q", n)
}
