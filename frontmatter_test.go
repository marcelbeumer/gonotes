package gonotes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestFrontmatterUnmarshalValues(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantMap map[string]string
	}{
		{
			name: "basics",
			input: `title: Hello world
date: 2026-01-15 20:43:09
tags: foo/bar, this/that
href: https://example.com
version: 2 # comment
meta: $$%%%`,
			wantMap: map[string]string{
				"title":   "Hello world",
				"date":    "2026-01-15 20:43:09",
				"tags":    "foo/bar, this/that",
				"href":    "https://example.com",
				"version": "2",
				"meta":    "$$%%%",
			},
		},
		{
			name: "with document node",
			input: `---
title: Hello world
version: 2
---`,
			wantMap: map[string]string{
				"title":   "Hello world",
				"version": "2",
			},
		},
		{
			name:  "unknown value",
			input: `title: Hello world`,
			wantMap: map[string]string{
				"unknown": "",
			},
		},
		{
			name:  "whitespace",
			input: `title:  Hello world   `,
			wantMap: map[string]string{
				"title": "Hello world",
			},
		},
		{
			name:    "parse error",
			input:   `	k: v`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFrontmatter()
			err := yaml.Unmarshal([]byte(tt.input), f)

			if !tt.wantErr && err != nil {
				t.Errorf("Unmarshal() err = %q", err)
			}

			if tt.wantErr && err == nil {
				t.Error("Unmarshal() err = <nil>")
			}

			for k, v := range tt.wantMap {
				got, _ := f.Get(k)
				want := v
				if got != want {
					t.Errorf("Got %q, want %q", got, want)
				}
			}
		})
	}
}

func TestFrontmatterMap(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name: "all scalar fields",
			input: `title: Hello world
date: 2026-01-15 20:43:09
tags: foo/bar, this/that
href: https://example.com`,
			want: map[string]string{
				"title": "Hello world",
				"date":  "2026-01-15 20:43:09",
				"tags":  "foo/bar, this/that",
				"href":  "https://example.com",
			},
		},
		{
			name:  "empty frontmatter",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "single field",
			input: `title: Test`,
			want:  map[string]string{"title": "Test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFrontmatter()
			if tt.input != "" {
				if err := yaml.Unmarshal([]byte(tt.input), f); err != nil {
					t.Fatalf("Unmarshal() err = %q", err)
				}
			}
			got := f.Map()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Map() diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestFrontmatterUnmarshalIdempotent(t *testing.T) {
	basics := `title: Hello world
date: 2026-01-15 20:43:09
href: https://example.com
version: 2 # comment
meta: $$%%%
`

	withObject := `title: Hello world
a:
    - b
    - dd
`

	title := `title: Hello world
`
	titleNoNewline := `title: Hello world`
	titleWithDoc := `---
title: Hello world
---`

	tests := []struct {
		name       string
		input      string
		wantOutput string
	}{
		{
			name:       "basics",
			input:      basics,
			wantOutput: basics,
		},
		{
			name:       "with document node",
			input:      titleWithDoc,
			wantOutput: title,
		},
		{
			name:       "without document node",
			input:      title,
			wantOutput: title,
		},
		{
			name:       "object",
			input:      withObject,
			wantOutput: withObject,
		},
		{
			name:       "adds newline",
			input:      titleNoNewline,
			wantOutput: title,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFrontmatter()
			if err := yaml.Unmarshal([]byte(tt.input), f); err != nil {
				t.Errorf("Unmarshal() err = %q", err)
			}

			b, err := yaml.Marshal(&f)
			if err != nil {
				t.Errorf("Marshal() err = %q", err)
			}

			if diff := cmp.Diff(tt.wantOutput, string(b)); diff != "" {
				t.Errorf("Marshall() vs Unmarshal() diff (-want, +got):\n%s", diff)
			}
		})
	}
}
