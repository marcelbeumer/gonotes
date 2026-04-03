package gonotes

import (
	"encoding/json"
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
		want    *Note // checked field by field, Frontmatter checked via Get
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
			name: "no title means empty slug",
			id:   "20260328-10",
			input: `---
date: 2026-03-28 10:00:00
---

Body without title in frontmatter.`,
			want: &Note{
				ID:   "20260328-10",
				Body: "\nBody without title in frontmatter.",
			},
			wantFM: map[string]string{
				"date": "2026-03-28 10:00:00",
			},
		},
		{
			name:    "malformed yaml returns error",
			id:      "20260328-11",
			input:   "---\n\t bad yaml: [unterminated\n---\n",
			wantErr: true,
		},
		{
			name: "single tag no comma",
			id:   "20260328-12",
			input: `---
tags: single-tag
---`,
			want: &Note{
				ID:   "20260328-12",
				Tags: []string{"single-tag"},
			},
		},
		{
			name: "nested tag preserved",
			id:   "20260328-13",
			input: `---
tags: bookmark/npm/request
---`,
			want: &Note{
				ID:   "20260328-13",
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

			// Check specific frontmatter values if provided.
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
	// Frontmatter should preserve all keys, not just the recognized ones.
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
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTags(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("parseTags(%q) diff (-want, +got):\n%s", tt.input, diff)
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

func TestJSON(t *testing.T) {
	input := `---
title: Hello world
date: 2026-03-28 10:00:00
tags: foo/bar, baz
href: https://example.com
---

Body with [[20260101-1]] and [[20260102-3]].`

	note, err := ReadNote("20260328-1", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadNote() err = %q", err)
	}

	b, err := note.JSON()
	if err != nil {
		t.Fatalf("JSON() err = %q", err)
	}

	// Unmarshal into a generic structure to verify fields.
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal() err = %q", err)
	}

	if got["id"] != "20260328-1" {
		t.Errorf("id = %v, want %q", got["id"], "20260328-1")
	}
	if got["title"] != "Hello world" {
		t.Errorf("title = %v, want %q", got["title"], "Hello world")
	}
	if got["slug"] != "hello-world" {
		t.Errorf("slug = %v, want %q", got["slug"], "hello-world")
	}

	tags, ok := got["tags"].([]any)
	if !ok {
		t.Fatalf("tags is not an array: %T", got["tags"])
	}
	wantTags := []any{"foo/bar", "baz"}
	if diff := cmp.Diff(wantTags, tags); diff != "" {
		t.Errorf("tags diff (-want, +got):\n%s", diff)
	}

	links, ok := got["internalLinks"].([]any)
	if !ok {
		t.Fatalf("internalLinks is not an array: %T", got["internalLinks"])
	}
	wantLinks := []any{"20260101-1", "20260102-3"}
	if diff := cmp.Diff(wantLinks, links); diff != "" {
		t.Errorf("internalLinks diff (-want, +got):\n%s", diff)
	}

	fm, ok := got["frontmatter"].(map[string]any)
	if !ok {
		t.Fatalf("frontmatter is not an object: %T", got["frontmatter"])
	}
	if fm["title"] != "Hello world" {
		t.Errorf("frontmatter.title = %v, want %q", fm["title"], "Hello world")
	}
	if fm["href"] != "https://example.com" {
		t.Errorf("frontmatter.href = %v, want %q", fm["href"], "https://example.com")
	}
	if fm["date"] != "2026-03-28 10:00:00" {
		t.Errorf("frontmatter.date = %v, want %q", fm["date"], "2026-03-28 10:00:00")
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
	_, err := ReadNote("20260328-1", strings.NewReader(`---
title: Bad Date
date: not-a-date
---`))
	if err == nil {
		t.Fatal("ReadNote() err = <nil>, want error for invalid date")
	}
	if !strings.Contains(err.Error(), "parse date") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "parse date")
	}
}

func TestJSONOmitsEmptyFields(t *testing.T) {
	note := NewNote()
	note.Frontmatter.Set("date", "2026-01-01 00:00:00")

	b, err := note.JSON()
	if err != nil {
		t.Fatalf("JSON() err = %q", err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal() err = %q", err)
	}

	// These should be omitted (omitempty).
	for _, key := range []string{"id", "title", "slug", "tags", "body", "internalLinks"} {
		if _, ok := got[key]; ok {
			t.Errorf("expected %q to be omitted, but it was present: %v", key, got[key])
		}
	}

	// Frontmatter should still be present.
	if _, ok := got["frontmatter"]; !ok {
		t.Error("expected frontmatter to be present")
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
