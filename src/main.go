package main

import (
	"marcelbeumer.com/notes/note"
)

func main() {
	_, err := note.FromPath("notes/2022-01/2022-01-04-2155-26-example-one.md")
	if err != nil {
		panic(err)
	}
	// fmt.Printf("%q", n)
}
