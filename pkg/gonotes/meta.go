package gonotes

import (
	"errors"
	"fmt"
	"time"

	"github.com/marcelbeumer/gonotes/pkg/util"
	"gopkg.in/yaml.v3"
)

var metaSepLine = "---"

func getMetaLine(k string, v string) string {
	o := make(map[string]string)
	o[k] = v
	s, err := yaml.Marshal(o)
	if err == nil {
		return string(s)
	}
	return ""
}

// Meta implements the meta block of a note. We implement our own marshall/unmarshall
// because we do our own date handling and the gopkg.in/yaml package is not flexible enough.
type Meta struct {
	Title    string
	Href     string
	Date     time.Time
	Modified *time.Time
	Tags     []string
}

func (n *Meta) MarshalYAML() ([]byte, error) {
	out := []byte{}
	if n.Title != "" {
		out = append(out, []byte(getMetaLine("title", n.Title))...)
	}
	out = append(out, []byte(fmt.Sprintf("date: %s\n", util.SerializeTime(n.Date)))...)
	if n.Modified != nil {
		out = append(out, []byte(fmt.Sprintf("modified: %s\n", util.SerializeTime(*n.Modified)))...)
	}
	if len(n.Tags) > 0 {
		out = append(out, []byte(getMetaLine("tags", SerializeTags(n.Tags)))...)
	}
	if n.Href != "" {
		out = append(out, []byte(getMetaLine("href", n.Href))...)
	}
	return out, nil
}

func (n *Meta) UnmarshalYAML(in []byte) error {
	var meta struct {
		Title    string
		Href     string
		Date     string
		Modified string
		Tags     string
	}
	err := yaml.Unmarshal(in, &meta)
	if err != nil {
		return err
	}
	n.Title = meta.Title
	n.Href = meta.Href
	n.Tags = ParseTags(meta.Tags)

	if meta.Date == "" {
		return errors.New("no date value")
	}
	date, err := util.ParseTime(meta.Date)
	if err != nil {
		return err
	}
	n.Date = date

	if meta.Modified != "" {
		date, err := util.ParseTime(meta.Modified)
		if err != nil {
			return err
		}
		n.Modified = &date
	}

	return nil
}
