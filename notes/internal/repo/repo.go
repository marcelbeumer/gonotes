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

	"github.com/bmatcuk/doublestar/v4"
	"github.com/marcelbeumer/notes-in-go/notes/internal/note"
	"golang.org/x/sync/errgroup"
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
	res = trailingSlash.ReplaceAllString(res, "")
	res = leadingSlash.ReplaceAllString(res, "")
	return res
}

func logStderr(msg string) {
	fmt.Fprintf(os.Stderr, msg)
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

func (r *Repo) CheckDir() error {
	_, err := r.rootDir()
	return err
}

func (r *Repo) rootDir() (string, error) {
	dotfilename := ".is_notes_root"
	cwd, err := os.Getwd()
	if err != nil {
		return cwd, err
	}
	_, err = os.Stat(path.Join(cwd, dotfilename))
	exists := !errors.Is(err, os.ErrNotExist)
	if !exists {
		return cwd,
			errors.New(fmt.Sprintf("Could not find %s file", dotfilename))
	}
	return cwd, nil
}

func (r *Repo) NotesSrcDir() (string, error) {
	rootDir, err := r.rootDir()
	if err != nil {
		return "", err
	}
	return path.Join(rootDir, "notes"), nil
}

func (r *Repo) tagsDir() (string, error) {
	rootDir, err := r.rootDir()
	if err != nil {
		return "", err
	}
	return path.Join(rootDir, "tags"), nil
}

func (r *Repo) LoadNotes() error {
	if r.loadingState == Loading {
		return errors.New("Already loading")
	}
	notesSrcDir, err := r.NotesSrcDir()
	if err != nil {
		return err
	}

	globRes, err := doublestar.Glob(os.DirFS(notesSrcDir), "**/*md")
	if err != nil {
		return errors.New(fmt.Sprintf("Could not glob for notes: %v", err))
	}

	files := make([]string, 0)
	for _, s := range globRes {
		files = append(files, path.Join(notesSrcDir, s))
	}

	r.records = make([](*record), 0)

	g := new(errgroup.Group)
	for _, path := range files {
		path := path
		_, err := r.loadNoteFromPath(path)
		if err != nil {
			return err
		}

		// g.Go(func() error {
		// 	_, err := r.loadNoteFromPath(path)
		// 	return err
		// })
	}

	if err := g.Wait(); err != nil {
		return err
	}

	r.loadingState = Loaded

	logStderr(fmt.Sprintf("Loaded %d notes\n", len(r.records)))

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

func (r *Repo) Sync(newOnly bool) error {
	if !newOnly {
		r.cleanTagsDir()
	}
	g := new(errgroup.Group)
	for _, record := range r.records {
		if newOnly && record.path != nil {
			continue
		}
		record := record
		g.Go(func() error {
			notePath, err := r.notePath(record.note)
			if err != nil {
				return err
			}
			if record.path != nil && *record.path != notePath {
				_, err := os.Stat(*record.path)
				exists := !errors.Is(err, os.ErrNotExist)
				if exists {
					err := os.Remove(*record.path)
					if err != nil {
						return err
					}
				}
			}
			record.path = &notePath
			md := record.note.Markdown()
			err = os.MkdirAll(path.Dir(*record.path), 0755)
			if err != nil {
				return err
			}
			err = os.WriteFile(*record.path, []byte(md), 0644)
			if err != nil {
				return errors.New(
					fmt.Sprintf("Could not write note to %s: %v", *record.path, err),
				)
			}

			for _, tag := range record.note.Tags {
				tag := tag
				tagsDir, err := r.tagsDir()
				if err != nil {
					return err
				}
				tagPath := path.Join(tagsDir, tag)
				err = os.MkdirAll(tagPath, 0755)
				if err != nil {
					return err
				}
				fileName := path.Base(*record.path)
				err = os.Symlink(*record.path, path.Join(tagPath, fileName))
				if err != nil {
					return err
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (r *Repo) PathIfStored(note *note.Note) (string, error) {
	for _, record := range r.records {
		if record.note == note {
			if record.path == nil {
				return "", errors.New("Note found but not yet stored")
			}
			return *record.path, nil
		}
	}
	return "", errors.New("Note not found")
}

func (r *Repo) notePath(note *note.Note) (string, error) {
	notesSrcDir, err := r.NotesSrcDir()
	if err != nil {
		return "", err
	}

	path, err := filepath.Abs(path.Join(
		notesSrcDir,
		getFolderBaseStr(note),
		getFolderDateStr(note),
		getNoteFileName(note),
	))
	if err != nil {
		return "", err
	}
	return path, nil
}

func (r *Repo) cleanTagsDir() error {
	tagsDir, err := r.tagsDir()
	if err != nil {
		return err
	}
	err = os.RemoveAll(tagsDir)
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
