package repo

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

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

func (r *Repo) AddNote(note *note.Note) {
	rec := record{
		note:  note,
		path:  getNotePath(note),
		isNew: true,
	}
	r.records[rec.path] = rec
}

func (r *Repo) Sync(newOnly bool) error {
	cleanTagsDir()
	for _, record := range r.records {
		if newOnly && !record.isNew {
			continue
		}
		notePath := getNotePath(record.note)
		if notePath != record.path {
			// delete from disk
			// delete record from map
			// set new path on record
			// add record to map
		}
	}
	for _, record := range r.records {
		if newOnly && !record.isNew {
			continue
		}
		// write note to disk
		record.isNew = false
		// make errgroup for each tag link stuff
	}
	return nil
}

func cleanTagsDir() error { return nil }
func deleteFile() error   { return nil }

func getFolderBaseStr(note *note.Note) string {
	return ""
}

func getFolderDateStr(note *note.Note) string {
	return note.CreatedTs.Format("2006-01")
}

func slugify(v string) string {
	disallowedChars := regexp.MustCompile("[^a-z0-9-]")
	doubleDash := regexp.MustCompile("-{2,}")
	trailingSlash := regexp.MustCompile("-$")
	leadingSlash := regexp.MustCompile("^-")
	res := strings.ToLower(v)
	res = disallowedChars.ReplaceAllString(res, "-")
	res = doubleDash.ReplaceAllString(res, "-")
	res = trailingSlash.ReplaceAllString(res, "-")
	res = leadingSlash.ReplaceAllString(res, "-")
	return res
}

func getNoteFileName(note *note.Note) string {
	dateStr := note.CreatedTs.Format("2006-01-02-1504-05")
	title := ""
	if note.Title != nil {
		title = *note.Title
	}
	res := dateStr
	if title != "" {
		res = res + "-" + slugify(title)
	}
	return res + ".md"
}

func getNotePath(note *note.Note) string {
	return path.Join(
		getFolderBaseStr(note),
		getFolderDateStr(note),
		getNoteFileName(note),
	)
}

func (r *Repo) loadNoteFromPath(path string) (*note.Note, error) {
	// absPath, err := filepath.Abs(path)
	// if err != nil {
	// 	return new(note.Note), err
	// }
	n, err := note.FromPath(path)
	if err != nil {
		return new(note.Note), err
	}
	newRecord := record{note: n, path: path, isNew: true}
	fmt.Println(path, getNotePath(n))
	r.records[path] = newRecord
	return n, nil
}
