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
	Meta       map[string]MetaField
	Title      *string
	Href       *string
	CreatedTs  time.Time
	ModifiedTs *time.Time
	Contents   string
	Tags       []string
}

func getMetaLine(k string, v string) string {
	o := make(map[string]string)
	o[k] = v
	s, err := yaml.Marshal(o)
	if err == nil {
		return string(s)
	}
	return ""
}

func (n *Note) Markdown() string {
	md := "---\n"
	if n.Title != nil && *n.Title != "" {
		md += getMetaLine("title", *n.Title)
	}
	md += fmt.Sprintf("date: %s\n", serializeTime(&n.CreatedTs))
	if n.ModifiedTs != nil {
		md += fmt.Sprintf("modified: %s\n", serializeTime(n.ModifiedTs))
	}
	if len(n.Tags) > 0 {
		md += fmt.Sprintf("tags: %s\n", strings.Join(n.Tags, ", "))
	}
	if n.Href != nil && *n.Href != "" {
		md += getMetaLine("href", *n.Href)
	}
	md += "---\n"
	md += n.Contents
	return md
}

func (n *Note) RenameTag(from string, to string) {
	fromParts := strings.Split(from, "/")
	toParts := strings.Split(to, "/")
	for tagI, tag := range n.Tags {
		tagParts := strings.Split(tag, "/")
		shouldRename := true
		for fromI, fromPart := range fromParts {
			if fromPart != tagParts[fromI] {
				shouldRename = false
				break
			}
		}
		if shouldRename {
			newParts := append(make([]string, 0), toParts...)
			for i, p := range tagParts {
				if i >= len(fromParts) {
					newParts = append(newParts, p)
				}
			}
			n.Tags[tagI] = strings.Join(newParts, "/")
		}
	}
}

func New() *Note {
	return &Note{
		Meta:      make(map[string]MetaField),
		CreatedTs: time.Now(),
	}
}

func FromReader(reader io.Reader) (*Note, error) {
	scanner := bufio.NewScanner(reader)
	metaLines := []string{}
	contentLines := []string{}
	withinMeta := false
	metaDone := false

	for scanner.Scan() {
		text := scanner.Text()
		if text == "---" && !metaDone {
			withinMeta = !withinMeta
			if !withinMeta {
				metaDone = true
			}
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
	note.Contents = strings.Join(contentLines, "\n") + "\n"

	for k, v := range parsedMeta {
		switch v := v.(type) {
		case string:
			note.Meta[k] = &StringField{string: v}
		case int:
			note.Meta[k] = &IntField{int: v}
		case time.Time:
			note.Meta[k] = &TimeField{time: v}
		default:
			note.Meta[k] = &UnknownField{value: v}
		}
	}

	if title := note.Meta["title"]; title != nil {
		switch v := title.(type) {
		case *StringField:
			s := v.String()
			note.Title = &s
		default:
			return note, errors.New("title not of correct type")
		}
	}

	if date := note.Meta["date"]; date != nil {
		t, e := parseTime(date.String())
		if e != nil {
			return note, e
		}
		note.CreatedTs = t
	} else {
		note.CreatedTs = time.Now()
		// return note, errors.New("Date field not found")
	}

	if modified := note.Meta["modified"]; modified != nil {
		t, e := parseTime(modified.String())
		if e != nil {
			return note, e
		}
		note.ModifiedTs = &t
	}

	if tags := note.Meta["tags"]; tags != nil {
		note.Tags = ParseTags(tags.String())
	}

	if href := note.Meta["href"]; href != nil {
		v := strings.TrimSpace(href.String())
		note.Href = &v
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
		return new(Note), fmt.Errorf("could not open file %v: %v", path, err)
	}
	defer file.Close()
	n, err := FromReader(file)
	if err != nil {
		return new(Note), err
	}
	return n, nil
}
