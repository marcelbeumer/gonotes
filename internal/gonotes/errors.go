package gonotes

import (
	"fmt"
	"strings"
)

type EmptyRepositoryError struct{}

func (e EmptyRepositoryError) Error() string {
	return "empty repository"
}

type NoteNotFoundError struct {
	NoteName string
}

func (e NoteNotFoundError) Error() string {
	return fmt.Sprintf(`note with name "%s" not found`, e.NoteName)
}

type DuplicateNoteErr struct {
	NoteName      string
	ExistingPath  string
	DuplicatePath string
}

func (e DuplicateNoteErr) Error() string {
	return fmt.Sprintf(
		"duplicate note with name %s (path: %s, duplicate: %s)",
		e.NoteName, e.ExistingPath, e.DuplicatePath,
	)
}

type NoteExistsError struct {
	NoteName string
}

func (e NoteExistsError) Error() string {
	return fmt.Sprintf(`note with name "%s" already exists`, e.NoteName)
}

type NotePathExistsError struct {
	NotePath string
}

func (e NotePathExistsError) Error() string {
	return fmt.Sprintf(`note on path "%s" already exists`, e.NotePath)
}

type LoadError struct {
	Errors []error
}

func (e LoadError) Error() string {
	messages := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		messages[i] = fmt.Sprintf("[%d] %s", i+1, err.Error())
	}

	return fmt.Sprintf(
		"load encountered %d errors:\n%s",
		len(e.Errors),
		strings.Join(messages, "\n"),
	)
}

type BuildTagFsError struct {
	Errors []error
}

func (e BuildTagFsError) Error() string {
	messages := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		messages[i] = fmt.Sprintf("[%d] %s", i+1, err.Error())
	}

	return fmt.Sprintf(
		"build tag fs encountered %d errors:\n%s",
		len(e.Errors),
		strings.Join(messages, "\n"),
	)
}
