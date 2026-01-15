package gonotes

import (
	"log"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const dateLayout = "2006-01-02 15:04:05"

var loc *time.Location

func init() {
	var err error
	loc, err = time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		log.Fatalf("Load location: %s", err)
	}
}

// TODO: consider moving specifics for tags, date and title etc out of
// frontmatter, that way frontmatter can stay generic.

type Frontmatter struct {
	yaml.Node
}

func NewFrontmatter() *Frontmatter {
	return &Frontmatter{
		Node: yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{},
		},
	}
}

func (f *Frontmatter) MarshalYAML() (any, error) {
	return f.Node, nil
}

func (f *Frontmatter) UnmarshalYAML(node *yaml.Node) error {
	f.Node = *node
	return nil
}

func (f *Frontmatter) Normalize() {
	if tags, ok := f.Tags(); ok {
		f.SetTags(tags)
	}
	if date, ok := f.Date(); ok {
		f.SetDate(date)
	}
}

func (f *Frontmatter) ensureMappingNode() *yaml.Node {
	// Root is already mapping node.
	if f.Kind == yaml.MappingNode {
		return &f.Node
	}

	// Root is document node.
	if f.Kind == yaml.DocumentNode && len(f.Content) > 0 {
		if f.Content[0].Kind == yaml.MappingNode {
			return f.Content[0]
		}
	}

	// No mapping node found, create and replace root.
	f.Node = yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}

	return &f.Node
}

func (f *Frontmatter) Value(key string) (string, bool) {
	mapping := f.ensureMappingNode()

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1].Value, true
		}
	}

	return "", false
}

func (f *Frontmatter) SetValue(key, value string) {
	mapping := f.ensureMappingNode()

	// Update existing.
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			// Ensure value node is scalar, or convert it.
			if mapping.Content[i+1].Kind != yaml.ScalarNode {
				mapping.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: value}
			} else {
				mapping.Content[i+1].Value = value
			}
			return
		}
	}

	// Add new.
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
}

func (f *Frontmatter) RemoveValue(key string) {
	mapping := f.ensureMappingNode()

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

func (f *Frontmatter) Title() (string, bool) {
	return f.Value("title")
}

func (f *Frontmatter) SetTitle(v string) {
	f.SetValue("title", v)
}

func (f *Frontmatter) Date() (time.Time, bool) {
	str, ok := f.Value("date")
	if !ok {
		return time.Time{}, false
	}

	t, err := time.ParseInLocation(dateLayout, str, loc)
	if err != nil {
		return time.Time{}, true
	}

	return t, true
}

func (f *Frontmatter) SetDate(t time.Time) {
	if t.IsZero() {
		f.SetValue("date", "")
		return
	}
	f.SetValue("date", t.Format(dateLayout))
}

func (f *Frontmatter) Tags() ([]string, bool) {
	str, ok := f.Value("tags")
	if !ok {
		return nil, false
	}

	str = strings.ToLower(str)
	tags := strings.Split(str, ",")

	existing := map[string]struct{}{}
	for i, v := range tags {
		slug := Slugify(v)
		if _, ok := existing[slug]; !ok {
			tags[i] = slug
			existing[slug] = struct{}{}
		}
	}

	return tags, true
}

func (f *Frontmatter) SetTags(tags []string) {
	normalized := make([]string, len(tags))
	existing := map[string]struct{}{}

	for i, v := range tags {
		slug := Slugify(v)
		if slug == "" {
			continue
		}
		if _, ok := existing[slug]; !ok {
			normalized[i] = slug
			existing[slug] = struct{}{}
		}
	}

	f.SetValue("tags", strings.Join(normalized, ", "))
}
