package repo

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"golang.org/x/sync/errgroup"
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
	records      map[string]record
	loadingState LoadingState
}

func New() *Repo {
	return &Repo{
		records: make(map[string]record),
	}
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
	files, err := filepath.Glob(path.Join(r.NotesRootDir(), "**/*.md"))
	if err != nil {
		return errors.New(fmt.Sprintf("Could not glob for notes: %v", err))
	}

	r.records = make(map[string]record, 0)

	g := new(errgroup.Group)
	for _, path := range files {
		path := path
		g.Go(func() error {
			_, err := r.loadNoteFromPath(path)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	r.loadingState = Loading
	return nil
}

func (r *Repo) Notes() [](*note.Note) {
	fmt.Println(len(r.records))
	res := make([]*note.Note, 0)
	for _, v := range r.records {
		res = append(res, v.note)
	}
	return res
}

func (r *Repo) loadNoteFromPath(path string) (*note.Note, error) {
	n, err := note.FromPath(path)
	if err != nil {
		return new(note.Note), err
	}
	newRecord := record{note: n, path: path, isNew: true}
	r.records[path] = newRecord
	return n, nil
}
