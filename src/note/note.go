package note

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type MetaField interface {
	String() string
}

type StringField struct {
	string
}

func (f *StringField) String() string {
	return f.string
}

type TimeField struct {
	time time.Time
}

func (f *TimeField) String() string {
	return fmt.Sprintf("%v", f.time)
}

func (f *TimeField) Time() time.Time {
	return f.time
}

type IntField struct {
	int
}

func (f *IntField) String() string {
	return fmt.Sprintf("%v", f.int)
}

func (f *IntField) Int() int {
	return f.int
}

type TagsField struct {
	string
}

func (f *TagsField) String() string {
	return f.string
}

func (f *TagsField) Tags() []string {
	r := regexp.MustCompile("[,\\s]")
	parts := r.Split(f.string, -1)
	tags := make([]string, 0)
	for _, v := range parts {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			tags = append(tags, v)
		}
	}
	return tags
}

type UnknownField struct {
	value interface{}
}

func (f *UnknownField) String() string {
	return fmt.Sprintf("%v", f.value)
}

type Note struct {
	meta       map[string]MetaField
	title      *string
	href       *string
	createdTs  time.Time
	modifiedTs *time.Time
	contents   string
	tags       []string
}

func (n *Note) Markdown() string {
	// TODO: implement
	return ""
}

func repoRoot() string {
	// TODO: implement
	return "../"
}

func FromReader(reader io.Reader) (Note, error) {
	scanner := bufio.NewScanner(reader)
	metaLines := []string{}
	contentLines := []string{}
	withinMeta := false

	for scanner.Scan() {
		text := scanner.Text()
		if text == "---" {
			withinMeta = !withinMeta
		} else {
			if withinMeta {
				metaLines = append(metaLines, text)
			} else {
				contentLines = append(contentLines, text)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Note{}, err
	}

	var parsedMeta map[string]interface{}
	bytes := []byte(strings.Join(metaLines, "\n"))
	err := yaml.Unmarshal(bytes, &parsedMeta)
	if err != nil {
		return Note{}, err
	}

	note := Note{
		contents: strings.Join(contentLines, "\n"),
		meta:     make(map[string]MetaField, 0),
	}

	for k, v := range parsedMeta {
		switch v := v.(type) {
		case string:
			if k == "tags" {
				field := &TagsField{string: v}
				note.meta[k] = field
				field.Tags()
			} else {
				note.meta[k] = &StringField{string: v}
			}
		case int:
			note.meta[k] = &IntField{int: v}
		case time.Time:
			note.meta[k] = &TimeField{time: v}
		default:
			note.meta[k] = &UnknownField{value: v}
		}
	}

	if title := note.meta["title"]; title != nil {
		switch v := title.(type) {
		case *StringField:
			s := v.String()
			note.title = &s
		default:
			return note, errors.New("Title not of correct type")
		}
	}

	if date := note.meta["date"]; date != nil {
		// TODO
		// note.createdTs = time.
	} else {
		return note, errors.New("Date field not found")
	}

	return note, nil
}

func FromString(string string) (Note, error) {
	reader := strings.NewReader(string)
	return FromReader(reader)
}

func FromPath(path string) (Note, error) {
	file, err := os.Open(filepath.Join(repoRoot(), path))
	if err != nil {
		return Note{}, err
	}
	defer file.Close()
	return FromReader(file)
}
