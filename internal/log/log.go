package log

import (
	"fmt"
	"os"
)

func Stderr(msg string) {
	fmt.Fprint(os.Stderr, msg)
}

func Stderrln(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

func Fstderr(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}
