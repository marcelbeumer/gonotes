package gonotes

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"sync"
)

func (r *Repository) writeNoteContents(name string) error {
	n, notePath, err := r.noteByName(name)
	if err != nil {
		return err
	}

	md, err := n.Marhsal()
	if err != nil {
		return err
	}
	err = os.MkdirAll(path.Dir(notePath), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(notePath, []byte(md), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) writeNoteTagsFs(name string) error {
	n, notePath, err := r.noteByName(name)
	if err != nil {
		return err
	}

	for _, tag := range n.Meta.Tags {
		tag := tag
		tagsDir, err := TagsDir()
		if err != nil {
			return err
		}
		tagPath := path.Join(tagsDir, tag)
		err = os.MkdirAll(tagPath, 0755)
		if err != nil {
			return err
		}
		fileName := path.Base(notePath)
		err = os.Symlink(notePath, path.Join(tagPath, fileName))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) renameNote(op *RenameNote) error {
	n, oldPath, err := r.noteByName(op.From)
	if err != nil {
		return err
	}

	delete(r.notes, op.From)
	delete(r.notePaths, op.From)

	if op.DeleteFrom {
		_, err := os.Stat(oldPath)
		exists := !errors.Is(err, os.ErrNotExist)
		if exists {
			err := os.Remove(oldPath)
			if err != nil {
				return err
			}
		}
	}

	notePath, err := NotePath(*n)
	if err != nil {
		return err
	}

	r.notes[op.To] = n
	r.notePaths[op.To] = notePath

	return r.writeNoteContents(op.To)
}

func (r *Repository) updateNote(op *UpdateNote) error {
	return r.writeNoteContents(op.Name)
}

func (r *Repository) cleanTagsDir() error {
	tagsDir, err := TagsDir()
	if err != nil {
		return err
	}
	err = os.RemoveAll(tagsDir)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) rebuildTagFs(op *RebuildTagsFs) error {
	if err := r.cleanTagsDir(); err != nil {
		return err
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error
	var names []string
	var noteCount = len(r.notes)
	var jobSize = int(math.Ceil(float64(noteCount) / 5))

	addError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, err)
	}

	execJob := func(jobSlice []string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, name := range jobSlice {
				if err := r.writeNoteTagsFs(name); err != nil {
					addError(err)
					return
				}
			}
		}()
	}

	var doneCount = 0
	for name := range r.notes {
		names = append(names, name)
		doneCount++

		if len(names) >= jobSize || doneCount == noteCount {
			execJob(names)
			names = nil
		}
	}

	wg.Wait()

	if len(errs) > 0 {
		return &BuildTagFsError{Errors: errs}
	}

	return nil
}

func (r *Repository) Apply(plan Plan, logw io.Writer) (bool, error) {
	if plan.HasErrors() {
		for _, err := range plan.Errors() {
			fmt.Fprintf(logw, "ERROR: %s\n", err.Error())
		}
		return false, nil
	}

	// For now we do each operation sequentially and expect performance
	// gains to be made within the operations themselves.
	for _, op := range plan {
		switch t := op.(type) {
		case *RenameNote:
			if err := r.renameNote(t); err != nil {
				return false, err
			}
		case *UpdateNote:
			if err := r.updateNote(t); err != nil {
				return false, err
			}
		case *RebuildTagsFs:
			if err := r.rebuildTagFs(t); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("unhandled operation type: %s", op.String())
		}
	}

	return true, nil
}
