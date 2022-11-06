package gonotes

import (
	"fmt"
	"io"
	"sort"
)

type Node struct {
	Notes     map[string]*Note
	Nodes     map[string]*Node
	CountDesc int
}

type Tree struct {
	AllNotes map[string]*Note
	Nodes    map[string]*Node
}

func (t *Tree) AddNote(n *Note, name string) error {
	if _, ok := t.AllNotes[name]; !ok {
		t.AllNotes[name] = n
	} else {
		return fmt.Errorf(`note "%s" already exists`, name)
	}

	for _, tag := range n.Meta.Tags {
		nodes := t.Nodes
		passed := []*Node{}
		parts := SplitTags(tag)
		added := false

		for i, part := range parts {
			node, ok := nodes[part]
			if !ok {
				newNode := Node{
					Notes:     map[string]*Note{},
					Nodes:     map[string]*Node{},
					CountDesc: 0,
				}
				nodes[part] = &newNode
				node = &newNode
			}
			if i == len(parts)-1 {
				if _, ok := node.Notes[name]; !ok {
					node.Notes[name] = n
					added = true
				}
			}
			passed = append(passed, node)
			nodes = node.Nodes
		}

		if added {
			for _, node := range passed {
				node.CountDesc += 1
			}
		}
	}

	return nil
}

func printTree(nodes map[string]*Node, out io.Writer, chain []bool) error {
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	if len(chain) == 0 {
		if _, err := io.WriteString(out, ".\n"); err != nil {
			return err
		}
	}

	for i, key := range keys {
		isLast := len(keys)-1 == i
		value := nodes[key]

		line := []byte{}
		for _, last := range chain {
			var char string
			if last {
				char = "    "
			} else {
				char = "│   "
			}
			line = append(line, char...)
		}
		var char string
		if isLast {
			char = "└── "
		} else {
			char = "├── "
		}
		line = append(line, char...)
		line = append(line, fmt.Sprintf("%s [%d]\n", key, value.CountDesc)...)

		if _, err := out.Write(line); err != nil {
			return err
		}

		if err := printTree(value.Nodes, out, append(chain, isLast)); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tree) Print(out io.Writer) error {
	if err := printTree(t.Nodes, out, []bool{}); err != nil {
		return err
	}
	return nil
}

func NewTree(notes map[string]*Note) *Tree {
	tree := Tree{
		AllNotes: map[string]*Note{},
		Nodes:    map[string]*Node{},
	}
	for name, note := range notes {
		tree.AddNote(note, name)
	}
	return &tree
}
