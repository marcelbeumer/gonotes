package gonotes

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

type Note struct {
	Meta    Meta
	Content string
	Raw     string
}

func (n *Note) RenameTag(from string, to string) (bool, error) {
	var changed bool
	re, err := regexp.Compile("^" + regexp.QuoteMeta(from))
	if err != nil {
		return changed, err
	}
	tagMap := map[string]struct{}{}
	for _, tag := range n.Meta.Tags {
		name := re.ReplaceAllString(tag, to)
		if name != tag {
			changed = true
		}
		tagMap[name] = struct{}{}
	}

	newTags := make([]string, 0, len(tagMap))
	for name := range tagMap {
		newTags = append(newTags, name)
	}

	n.Meta.Tags = newTags
	return changed, nil
}

func (n *Note) Marhsal() ([]byte, error) {
	out := []byte{}
	out = append(out, []byte(metaSepLine)...)
	out = append(out, []byte("\n")...)
	meta, err := n.Meta.MarshalYAML()
	if err != nil {
		return out, err
	}
	out = append(out, meta...)
	out = append(out, []byte(metaSepLine)...)
	if n.Content != "" {
		out = append(out, []byte("\n")...)
		out = append(out, []byte(n.Content)...)
	}
	return out, nil
}

func NoteFromReader(r io.Reader) (Note, error) {
	scanner := bufio.NewScanner(r)
	rawLines := []string{}
	lines := []string{}
	metaLines := []string{}
	metaCount := 0
	note := Note{}

	for scanner.Scan() {
		text := scanner.Text()
		rawLines = append(rawLines, text)

		if metaCount < 2 && text == metaSepLine {
			metaCount += 1
			continue
		}

		switch metaCount {
		case 1:
			metaLines = append(metaLines, text)
		default:
			lines = append(lines, text)
		}
	}

	yaml := []byte(strings.Join(metaLines, "\n"))
	var meta Meta
	err := meta.UnmarshalYAML(yaml)
	if err != nil {
		return note, err
	}

	note.Meta = meta
	note.Content = strings.Join(lines, "\n")
	note.Raw = strings.Join(rawLines, "\n")

	return note, nil
}
