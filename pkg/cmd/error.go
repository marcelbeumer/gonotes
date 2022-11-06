package cmd

import (
	"fmt"
	"os"
)

func handleError(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
