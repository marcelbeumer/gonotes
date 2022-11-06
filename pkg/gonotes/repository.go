package gonotes

import (
	"fmt"
	"math"
	"sync"
)

type Repository struct {
	// notePaths is a map of name to file path.
	notePaths map[string]string
	// notes is a map of name to Note.
	// For each entry there must an entry in notePaths too.
	notes map[string]*Note
}

func (r *Repository) NotePaths() map[string]string {
	return r.notePaths
}

func (r *Repository) Notes() map[string]*Note {
	return r.notes
}

func (r *Repository) LoadPaths() error {
	paths, err := NotePathsAbs()
	if err != nil {
		return err
	}

	for _, p := range paths {
		name := NoteNameFromPath(p)
		if existingPath, hasKey := r.notePaths[name]; hasKey {
			return &DuplicateNoteErr{
				NoteName:      name,
				ExistingPath:  existingPath,
				DuplicatePath: p,
			}
		} else {
			r.notePaths[name] = p
		}
	}
	return nil
}

func (r *Repository) AddNote(n Note) (string, error) {
	name := NoteName(n)
	if _, ok := r.notes[name]; ok {
		return "", &NoteExistsError{
			NoteName: name,
		}
	}

	notePath, err := NotePath(n)
	if err != nil {
		return "", err
	}

	for _, p := range r.notePaths {
		if notePath == p {
			return "", &NotePathExistsError{
				NotePath: notePath,
			}
		}
	}

	if err = WriteNoteToDisk(n, notePath); err != nil {
		return "", err
	}

	r.notePaths[name] = notePath
	r.notes[name] = &n
	return notePath, nil
}

func (r *Repository) LoadNotes() error {
	type jobEntry struct {
		name     string
		filePath string
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error
	var job []jobEntry
	var noteCount = len(r.notePaths)
	var jobSize = int(math.Ceil(float64(noteCount) / 5))

	addError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, err)
	}

	setNote := func(n *Note, name string) {
		mu.Lock()
		defer mu.Unlock()
		r.notes[name] = n
	}

	execJob := func(jobSlice []jobEntry) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, entry := range jobSlice {
				n, err := LoadNoteFromDisk(entry.filePath)
				if err != nil {
					addError(err)
					return // end job; forget about other entries
				}
				setNote(&n, entry.name)
			}
		}()
	}

	var doneCount = 0
	for name, filePath := range r.notePaths {
		name := name
		filePath := filePath
		job = append(job, jobEntry{name: name, filePath: filePath})
		doneCount++

		if len(job) >= jobSize || doneCount == noteCount {
			execJob(job[:])
			job = nil
		}
	}

	wg.Wait()

	if len(errs) > 0 {
		return &LoadError{Errors: errs}
	}

	return nil
}

func (r *Repository) FindNote(ref string) *Note {
	name := RefToName(ref)
	return r.notes[name]
}

func (r *Repository) LastNote() (noteName string, err error) {
	var note *Note
	for name, n := range r.notes {
		if note == nil || n.Meta.Date.UnixNano() > note.Meta.Date.UnixNano() {
			note = n
			noteName = name
		}
	}
	if noteName == "" {
		return "", &EmptyRepositoryError{}
	}
	return noteName, nil
}

func (r *Repository) GetTree() *Tree {
	return NewTree(r.notes)
}

func (r *Repository) RenameTag(from string, to string) (int, error) {
	var renameCount int
	for _, n := range r.notes {
		changed, err := n.RenameTag(from, to)
		if err != nil {
			return renameCount, err
		}
		if changed {
			renameCount++
		}
	}
	return renameCount, nil
}

func (r *Repository) noteByName(name string) (*Note, string, error) {
	notePath, ok := r.notePaths[name]
	if !ok {
		return nil, "", fmt.Errorf("data integrity: no path found for %s", name)
	}
	n, ok := r.notes[name]
	if !ok {
		return nil, notePath, fmt.Errorf("data integrity: no note found for %s", name)
	}
	return n, notePath, nil
}

func NewRepository() *Repository {
	return &Repository{
		notePaths: map[string]string{},
		notes:     map[string]*Note{},
	}
}
