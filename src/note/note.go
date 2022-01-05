package note

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type MetaField interface {
	sealed()
	String() string
}

type StringField struct {
	string
}

func (f *StringField) sealed() {}

func (f *StringField) String() string {
	return f.string
}

type TimeField struct {
	time time.Time
}

func (f *TimeField) sealed() {}

func (f *TimeField) String() string {
	return fmt.Sprintf("%v", f.time)
}

func (f *TimeField) Time() time.Time {
	return f.time
}

type IntField struct {
	int
}

func (f *IntField) sealed() {}

func (f *IntField) String() string {
	return fmt.Sprintf("%v", f.int)
}

func (f *IntField) Int() int {
	return f.int
}

type UnknownField struct {
	value interface{}
}

func (f *UnknownField) sealed() {}

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

func New() *Note {
	return &Note{
		meta: make(map[string]MetaField, 0),
	}
}

func FromReader(reader io.Reader) (*Note, error) {
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
		return &Note{}, err
	}

	var parsedMeta map[string]interface{}
	bytes := []byte(strings.Join(metaLines, "\n"))
	err := yaml.Unmarshal(bytes, &parsedMeta)
	if err != nil {
		return &Note{}, err
	}

	note := New()
	note.contents = strings.Join(contentLines, "\n")

	for k, v := range parsedMeta {
		switch v := v.(type) {
		case string:
			note.meta[k] = &StringField{string: v}
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
		t, e := ParseTime(date.String())
		if e != nil {
			return note, e
		}
		note.createdTs = t
	} else {
		return note, errors.New("Date field not found")
	}

	if modified := note.meta["modified"]; modified != nil {
		t, e := ParseTime(modified.String())
		if e != nil {
			return note, e
		}
		note.modifiedTs = &t
	}

	if tags := note.meta["tags"]; tags != nil {
		note.tags = ParseTags(tags.String())
	}

	if href := note.meta["href"]; href != nil {
		v := strings.TrimSpace(href.String())
		note.href = &v
	}

	return note, nil
}

func FromString(string string) (*Note, error) {
	reader := strings.NewReader(string)
	return FromReader(reader)
}

func FromPath(path string) (*Note, error) {
	file, err := os.Open(path)
	if err != nil {
		return new(Note),
			errors.New(fmt.Sprintf("Could not open file %v: %v", path, err))
	}
	defer file.Close()
	n, err := FromReader(file)
	if err != nil {
		return new(Note), err
	}
	return n, nil
}
