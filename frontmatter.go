package gonotes

import (
	"gopkg.in/yaml.v3"
)

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

func (f *Frontmatter) mappingNode() *yaml.Node {
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

func (f *Frontmatter) Get(key string) (string, bool) {
	mn := f.mappingNode()

	for i := 0; i+1 < len(mn.Content); i += 2 {
		if mn.Content[i].Value == key {
			return mn.Content[i+1].Value, true
		}
	}

	return "", false
}

func (f *Frontmatter) Set(key, value string) {
	mn := f.mappingNode()

	// Update existing.
	for i := 0; i+1 < len(mn.Content); i += 2 {
		if mn.Content[i].Value == key {
			// Ensure value node is scalar, or convert it.
			if mn.Content[i+1].Kind != yaml.ScalarNode {
				mn.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: value}
			} else {
				mn.Content[i+1].Value = value
			}
			return
		}
	}

	// Add new.
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	mn.Content = append(mn.Content, keyNode, valueNode)
}

func (f *Frontmatter) Unset(key string) {
	mn := f.mappingNode()

	for i := 0; i+1 < len(mn.Content); i += 2 {
		if mn.Content[i].Value == key {
			mn.Content = append(mn.Content[:i], mn.Content[i+2:]...)
			return
		}
	}
}

// Map returns all scalar key-value pairs as a map. Non-scalar values are
// represented by their string Value (which may be empty for sequences/mappings).
func (f *Frontmatter) Map() map[string]string {
	mn := f.mappingNode()
	m := make(map[string]string, len(mn.Content)/2)
	for i := 0; i+1 < len(mn.Content); i += 2 {
		m[mn.Content[i].Value] = mn.Content[i+1].Value
	}
	return m
}
