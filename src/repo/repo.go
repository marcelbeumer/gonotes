package repo

import (
	"errors"

	"marcelbeumer.com/notes/note"
)

type record struct {
	note  *note.Note
	path  string
	isNew bool
}

type LoadingState int

const (
	Bare LoadingState = iota + 1
	Loading
	Loaded
)

type Repo struct {
	records      []record
	loadingState LoadingState
}

func New() *Repo {
	return &Repo{}
}

func (r *Repo) LoadingState() LoadingState {
	return r.loadingState
}

func (r *Repo) NotesRootDir() string {
	// TODO: implement
	return "../notes"
}

func (r *Repo) LoadNotes() error {
	if r.loadingState == Loading {
		return errors.New("Already loading")
	}
	r.records = make([]record, 0)
	r.loadingState = Loading
	return nil
}
