package gonotes

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/marcelbeumer/gonotes/pkg/util"
)

var notesURIRegexp = regexp.MustCompile(`^notes(?:\+notes)?:\/\/(?P<name>.*?)(?:\..*)?$`)
var notesPathRegexp = regexp.MustCompile(`[/.]`)

func NoteNameFromURI(uri string) string {
	found := notesURIRegexp.FindStringSubmatch(uri)
	if len(found) > 0 {
		return found[1]
	}
	return ""
}

func IsURIRef(ref string) bool {
	return notesURIRegexp.Match([]byte(ref))
}

func IsNameRef(ref string) bool {
	return !IsURIRef(ref) && !IsPathRef(ref)
}

func IsPathRef(ref string) bool {
	return !IsURIRef(ref) && notesPathRegexp.Match([]byte(ref))
}

func RefToName(ref string) string {
	switch {
	case IsURIRef(ref):
		return NoteNameFromURI(ref)
	case IsPathRef(ref):
		return NoteNameFromPath(ref)
	default:
		return ref
	}
}

func NoteName(n Note) string {
	name := n.Meta.Date.Format("2006-01-02-1504-05")
	title := n.Meta.Title
	if title != "" {
		name = fmt.Sprintf("%s-%s", name, util.Slugify(title))
	}
	return name
}

func RootDir() (string, error) {
	rootFname := ".is_gonotes_root"
	cwd, err := os.Getwd()
	if err != nil {
		return cwd, err
	}
	_, err = os.Stat(path.Join(cwd, rootFname))
	exists := !errors.Is(err, os.ErrNotExist)
	if !exists {
		return cwd, fmt.Errorf("could not find %s file", rootFname)
	}
	if err != nil {
		return "", err
	}
	return cwd, nil
}

func NotesDir() (string, error) {
	base, err := RootDir()
	if err != nil {
		return "", err
	}
	return path.Join(base, "notes"), nil
}

func TagsDir() (string, error) {
	base, err := RootDir()
	if err != nil {
		return "", err
	}
	return path.Join(base, "tags"), nil
}

func NotePathsAbs() ([]string, error) {
	dir, err := NotesDir()
	if err != nil {
		return nil, err
	}
	paths, err := doublestar.Glob(os.DirFS(dir), "**/*md")
	if err != nil {
		return nil, fmt.Errorf("could not glob for notes: %v", err)
	}
	for i, path := range paths {
		absPath := path
		if !filepath.IsAbs(path) {
			absPath = filepath.Join(dir, path)
		}
		paths[i] = absPath
	}
	return paths, nil
}

func NoteNameFromPath(notePath string) string {
	ext := filepath.Ext(notePath)
	base := filepath.Base(notePath)
	return base[0 : len(base)-len(ext)]
}

func NotePath(n Note) (string, error) {
	notesDir, err := NotesDir()
	if err != nil {
		return "", err
	}
	return filepath.Abs(path.Join(
		notesDir,
		n.Meta.Date.Format("2006-01"),
		NoteName(n),
	) + ".md")
}

func WriteNoteToDisk(n Note, filePath string) error {
	err := os.MkdirAll(path.Dir(filePath), 0755)
	if err != nil {
		return err
	}

	b, err := n.Marhsal()
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadNoteFromDisk(filePath string) (Note, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return Note{}, err
	}
	return NoteFromReader(f)
}
