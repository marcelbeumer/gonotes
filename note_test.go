package gonotes

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestReadNote(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		input   string
		wantErr bool
		want    *Note
		wantFM  map[string]string
	}{
		{
			name: "full note with all fields",
			id:   "20260328-1",
			input: `---
title: Hello world
date: 2026-03-28 10:00:00
tags: foo/bar, this/that, plain
href: https://example.com
---

Some body content here.`,
			want: &Note{
				ID:    "20260328-1",
				Title: "Hello world",
				Slug:  "hello-world",
				Tags:  []string{"foo/bar", "this/that", "plain"},
				Body:  "\nSome body content here.",
			},
			wantFM: map[string]string{
				"title": "Hello world",
				"date":  "2026-03-28 10:00:00",
				"tags":  "foo/bar, this/that, plain",
				"href":  "https://example.com",
			},
		},
		{
			name:  "no frontmatter, body only",
			id:    "20260328-2",
			input: "Just some plain text\nwith multiple lines.",
			want: &Note{
				ID:   "20260328-2",
				Body: "Just some plain text\nwith multiple lines.",
			},
		},
		{
			name:  "empty input",
			id:    "20260328-3",
			input: "",
			want: &Note{
				ID: "20260328-3",
			},
		},
		{
			name: "frontmatter only, no body",
			id:   "20260328-4",
			input: `---
title: No body
---`,
			want: &Note{
				ID:    "20260328-4",
				Title: "No body",
				Slug:  "no-body",
			},
			wantFM: map[string]string{
				"title": "No body",
			},
		},
		{
			name: "multiple --- in body",
			id:   "20260328-5",
			input: `---
title: Multiple separators
---

Some content

---

More content after separator

---`,
			want: &Note{
				ID:    "20260328-5",
				Title: "Multiple separators",
				Slug:  "multiple-separators",
				Body:  "\nSome content\n\n---\n\nMore content after separator\n\n---",
			},
		},
		{
			name: "wiki links in body",
			id:   "20260328-6",
			input: `---
title: Links
---

See [[20260101-1]] and also [[20260102-3]] for details.`,
			want: &Note{
				ID:            "20260328-6",
				Title:         "Links",
				Slug:          "links",
				Body:          "\nSee [[20260101-1]] and also [[20260102-3]] for details.",
				InternalLinks: []string{"20260101-1", "20260102-3"},
			},
		},
		{
			name:  "wiki links without frontmatter",
			id:    "20260328-7",
			input: "Check [[20260101-1]] here.",
			want: &Note{
				ID:            "20260328-7",
				Body:          "Check [[20260101-1]] here.",
				InternalLinks: []string{"20260101-1"},
			},
		},
		{
			name: "no wiki links",
			id:   "20260328-8",
			input: `---
title: No links
---

Plain text without links.`,
			want: &Note{
				ID:    "20260328-8",
				Title: "No links",
				Slug:  "no-links",
				Body:  "\nPlain text without links.",
			},
		},
		{
			name: "tags with extra whitespace",
			id:   "20260328-9",
			input: `---
tags:  foo ,  bar/baz ,  ,  qux
---`,
			want: &Note{
				ID:   "20260328-9",
				Tags: []string{"foo", "bar/baz", "qux"},
			},
		},
		{
			name: "tags space separated",
			id:   "20260328-10",
			input: `---
tags: foo bar/baz qux
---`,
			want: &Note{
				ID:   "20260328-10",
				Tags: []string{"foo", "bar/baz", "qux"},
			},
		},
		{
			name: "no title means empty slug",
			id:   "20260328-11",
			input: `---
date: 2026-03-28 10:00:00
---

Body without title in frontmatter.`,
			want: &Note{
				ID:   "20260328-11",
				Body: "\nBody without title in frontmatter.",
			},
			wantFM: map[string]string{
				"date": "2026-03-28 10:00:00",
			},
		},
		{
			name:    "malformed yaml returns error",
			id:      "20260328-12",
			input:   "---\n\t bad yaml: [unterminated\n---\n",
			wantErr: true,
		},
		{
			name: "single tag no comma",
			id:   "20260328-13",
			input: `---
tags: single-tag
---`,
			want: &Note{
				ID:   "20260328-13",
				Tags: []string{"single-tag"},
			},
		},
		{
			name: "nested tag preserved",
			id:   "20260328-14",
			input: `---
tags: bookmark/npm/request
---`,
			want: &Note{
				ID:   "20260328-14",
				Tags: []string{"bookmark/npm/request"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadNote(tt.id, strings.NewReader(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("ReadNote() err = <nil>, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("ReadNote() err = %q", err)
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.want.ID)
			}
			if got.Title != tt.want.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.want.Title)
			}
			if got.Slug != tt.want.Slug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.want.Slug)
			}
			if got.Body != tt.want.Body {
				t.Errorf("Body diff (-want, +got):\n%s", cmp.Diff(tt.want.Body, got.Body))
			}
			if diff := cmp.Diff(tt.want.Tags, got.Tags); diff != "" {
				t.Errorf("Tags diff (-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.InternalLinks, got.InternalLinks); diff != "" {
				t.Errorf("InternalLinks diff (-want, +got):\n%s", diff)
			}

			for k, v := range tt.wantFM {
				gotV, ok := got.Frontmatter.Get(k)
				if !ok {
					t.Errorf("Frontmatter.Get(%q) not found, want %q", k, v)
				} else if gotV != v {
					t.Errorf("Frontmatter.Get(%q) = %q, want %q", k, gotV, v)
				}
			}
		})
	}
}

func TestReadNotePreservesFrontmatter(t *testing.T) {
	input := `---
title: Test
date: 2026-03-28 10:00:00
tags: a, b
href: https://example.com
custom: value
---

Body.`

	note, err := ReadNote("20260328-1", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadNote() err = %q", err)
	}

	for _, kv := range []struct{ k, v string }{
		{"title", "Test"},
		{"date", "2026-03-28 10:00:00"},
		{"tags", "a, b"},
		{"href", "https://example.com"},
		{"custom", "value"},
	} {
		got, ok := note.Frontmatter.Get(kv.k)
		if !ok {
			t.Errorf("Frontmatter.Get(%q) not found", kv.k)
		} else if got != kv.v {
			t.Errorf("Frontmatter.Get(%q) = %q, want %q", kv.k, got, kv.v)
		}
	}
}

func TestMarkdown(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want string
	}{
		{
			name: "full note",
			note: func() *Note {
				n, _ := ReadNote("20260328-1", strings.NewReader(`---
title: Hello
tags: a, b
---

Body here.`))
				return n
			}(),
			want: `---
title: Hello
tags: a, b
---

Body here.`,
		},
		{
			name: "no frontmatter",
			note: &Note{
				Frontmatter: NewFrontmatter(),
				Body:        "Just body.",
			},
			want: "Just body.",
		},
		{
			name: "empty note",
			note: &Note{
				Frontmatter: NewFrontmatter(),
			},
			want: "",
		},
		{
			name: "frontmatter only",
			note: func() *Note {
				n, _ := ReadNote("20260328-1", strings.NewReader(`---
title: No body
---`))
				return n
			}(),
			want: "---\ntitle: No body\n---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.note.Markdown()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Markdown() diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestMarkdownRoundTrip(t *testing.T) {
	input := `---
title: Round trip
date: 2026-03-28 10:00:00
tags: foo/bar, baz
---

Body with [[20260101-1]] link.`

	note1, err := ReadNote("20260328-1", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadNote() err = %q", err)
	}

	md := note1.Markdown()

	note2, err := ReadNote("20260328-1", strings.NewReader(md))
	if err != nil {
		t.Fatalf("ReadNote() round trip err = %q", err)
	}

	if note1.ID != note2.ID {
		t.Errorf("ID: %q != %q", note1.ID, note2.ID)
	}
	if note1.Title != note2.Title {
		t.Errorf("Title: %q != %q", note1.Title, note2.Title)
	}
	if note1.Slug != note2.Slug {
		t.Errorf("Slug: %q != %q", note1.Slug, note2.Slug)
	}
	if diff := cmp.Diff(note1.Tags, note2.Tags); diff != "" {
		t.Errorf("Tags diff:\n%s", diff)
	}
	if diff := cmp.Diff(note1.InternalLinks, note2.InternalLinks); diff != "" {
		t.Errorf("InternalLinks diff:\n%s", diff)
	}
	if note1.Body != note2.Body {
		t.Errorf("Body diff:\n%s", cmp.Diff(note1.Body, note2.Body))
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"foo, bar, baz", []string{"foo", "bar", "baz"}},
		{"foo/bar, this/that", []string{"foo/bar", "this/that"}},
		{"single", []string{"single"}},
		{" foo , bar ", []string{"foo", "bar"}},
		{",,,", nil},
		{"", nil},
		{"a,,b", []string{"a", "b"}},
		{"foo bar baz", []string{"foo", "bar", "baz"}},
		{"foo/bar  this/that", []string{"foo/bar", "this/that"}},
		{"  foo   bar  ", []string{"foo", "bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseTags(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ParseTags(%q) diff (-want, +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestParseInternalLinks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single link", "See [[20260101-1]].", []string{"20260101-1"}},
		{"multiple links", "[[a]] and [[b]].", []string{"a", "b"}},
		{"no links", "No links here.", nil},
		{"empty", "", nil},
		{"link with title slug", "[[20260101-1-some-title]]", []string{"20260101-1-some-title"}},
		{"adjacent links", "[[a]][[b]]", []string{"a", "b"}},
		{"link in multiline", "line1\n[[a]]\nline2\n[[b]]", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInternalLinks(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("parseInternalLinks() diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestReadNoteDateParsing(t *testing.T) {
	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: Hello
date: 2026-03-28 14:30:00
---

Body.`))
	if err != nil {
		t.Fatalf("ReadNote() err = %q", err)
	}

	want := time.Date(2026, 3, 28, 14, 30, 0, 0, time.UTC)
	if !note.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", note.Date, want)
	}
}

func TestReadNoteDateZero(t *testing.T) {
	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: No Date
---

Body.`))
	if err != nil {
		t.Fatalf("ReadNote() err = %q", err)
	}

	if !note.Date.IsZero() {
		t.Errorf("Date = %v, want zero", note.Date)
	}
}

func TestReadNoteDateInvalid(t *testing.T) {
	note, err := ReadNote("20260328-1", strings.NewReader(`---
title: Bad Date
date: not-a-date
---`))
	if err != nil {
		t.Fatalf("ReadNote() err = %q", err)
	}
	if !note.Date.IsZero() {
		t.Errorf("Date = %v, want zero for unparseable date", note.Date)
	}
}

func TestReadNoteIgnoreLinks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "single pattern",
			input: `---
ignore-links: 20260403-99
---

See [[20260403-99]].`,
			want: []string{"20260403-99"},
		},
		{
			name: "multiple patterns",
			input: `---
ignore-links: 20260403-99, some-folder/*
---

See [[20260403-99]] and [[some-folder/doc.pdf]].`,
			want: []string{"20260403-99", "some-folder/*"},
		},
		{
			name:  "no ignore-links",
			input: "Just body.",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := ReadNote("test", strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("ReadNote() err = %q", err)
			}
			if diff := cmp.Diff(tt.want, note.IgnoreLinks); diff != "" {
				t.Errorf("IgnoreLinks mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDedupStrings(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, nil},
		{"empty", []string{}, nil},
		{"no dups", []string{"a", "b"}, []string{"a", "b"}},
		{"dups", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupStrings(tt.in)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("dedupStrings() diff (-want, +got):\n%s", diff)
			}
		})
	}
}
