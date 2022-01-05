package main

import (
	"marcelbeumer.com/notes/repo"
)

func main() {
	repo := repo.New()
	if err := repo.LoadNotes(); err != nil {
		panic(err)
	}
	// _, err := note.FromPath("notes/2022-01/2022-01-04-2155-26-example-one.md")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("%q", n)
}
