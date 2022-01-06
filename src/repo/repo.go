package repo

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"marcelbeumer.com/notes/note"
)

type record struct {
	note *note.Note
	path *string
}

type LoadingState int

const (
	Bare LoadingState = iota + 1
	Loading
	Loaded
)

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

type Repo struct {
	records      [](*record)
	loadingState LoadingState
	recordsMutex *sync.Mutex
}

func New() *Repo {
	return &Repo{
		records:      make([](*record), 0),
		recordsMutex: &sync.Mutex{},
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

	r.records = make([](*record), 0)

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
	rec := record{note: note}
	r.addRecord(&rec)
}

func foo(v *string) {
	fmt.Println(*v)
}

func (r *Repo) addRecord(record *record) {
	if record.path != nil {
		r.removeRecordWithPath(*record.path)
	}
	r.recordsMutex.Lock()
	defer r.recordsMutex.Unlock()
	r.records = append(r.records, record)
}

func (r *Repo) removeRecordWithPath(path string) {
	r.recordsMutex.Lock()
	defer r.recordsMutex.Unlock()
	records := make([](*record), 0)
	for _, v := range r.records {
		if *v.path != path {
			records = append(records, v)
		}
	}
	r.records = records
}

// func (r *Repo) addNoteOnPath
//
// func (r *Repo) RemovePath(note *note.Note) {
// }

func (r *Repo) Sync(newOnly bool) error {
	r.cleanTagsDir()
	for _, record := range r.records {
		if newOnly && record.path != nil {
			continue
		}
		path, err := r.notePath(record.note)
		if err != nil {
			return err
		}
		if record.path != nil && *record.path != path {
			_, err := os.Stat(*record.path)
			exists := !errors.Is(err, os.ErrNotExist)
			if exists {
				err := os.Remove(*record.path)
				if err != nil {
					return err
				}
			}
		}
		record.path = &path

	}
	// for _, record := range r.records {
	// 	if newOnly && !record.isNew {
	// 		continue
	// 	}
	// 	notePath := getNotePath(record.note)
	// 	if notePath != record.path {
	// 		// delete from disk
	// 		// delete record from map
	// 		// set new path on record
	// 		// add record to map
	// 	}
	// }
	// for _, record := range r.records {
	// 	if newOnly && !record.isNew {
	// 		continue
	// 	}
	// 	// write note to disk
	// 	record.isNew = false
	// 	// make errgroup for each tag link stuff
	// }
	return nil
}

func (r *Repo) notePath(note *note.Note) (string, error) {
	path, err := filepath.Abs(path.Join(
		r.NotesRootDir(),
		getFolderBaseStr(note),
		getFolderDateStr(note),
		getNoteFileName(note),
	))
	if err != nil {
		return "", err
	}
	return path, nil
}

func (r *Repo) tagsDir() string {
	// TODO: implement
	return "../tags"
}

func (r *Repo) cleanTagsDir() error {
	err := os.RemoveAll(r.tagsDir())
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) deleteFile() error { return nil }

func (r *Repo) loadNoteFromPath(path string) (*note.Note, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return new(note.Note), err
	}
	n, err := note.FromPath(path)
	if err != nil {
		return new(note.Note), err
	}
	newRecord := record{note: n, path: &absPath}
	r.addRecord(&newRecord)
	return n, nil
}
