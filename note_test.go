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

func TestPrepare(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		opts    PrepareOptions
		wantFM  map[string]string
		wantErr bool
	}{
		{
			name:  "nil reader produces note with date default",
			input: "",
			opts:  PrepareOptions{Now: fixedNow},
			wantFM: map[string]string{
				"date": "2026-03-28 14:30:00",
			},
		},
		{
			name:  "nil reader with title and tags",
			input: "",
			opts: PrepareOptions{
				Title: StringPtr("New note"),
				Tags:  []string{"foo", "bar"},
				Now:   fixedNow,
			},
			wantFM: map[string]string{
				"title": "New note",
				"tags":  "foo, bar",
				"date":  "2026-03-28 14:30:00",
			},
		},
		{
			name: "explicit title overwrites existing",
			input: `---
title: Old title
date: 2026-01-01 00:00:00
---

Body.`,
			opts: PrepareOptions{
				Title: StringPtr("New title"),
				Now:   fixedNow,
			},
			wantFM: map[string]string{
				"title": "New title",
				"date":  "2026-01-01 00:00:00",
			},
		},
		{
			name: "tags are additive",
			input: `---
tags: old-tag
date: 2026-01-01 00:00:00
---`,
			opts: PrepareOptions{
				Tags: []string{"new/tag", "another"},
				Now:  fixedNow,
			},
			wantFM: map[string]string{
				"tags": "old-tag, new/tag, another",
				"date": "2026-01-01 00:00:00",
			},
		},
		{
			name: "tags additive with dedup",
			input: `---
tags: foo, bar
date: 2026-01-01 00:00:00
---`,
			opts: PrepareOptions{
				Tags: []string{"bar", "baz"},
				Now:  fixedNow,
			},
			wantFM: map[string]string{
				"tags": "foo, bar, baz",
				"date": "2026-01-01 00:00:00",
			},
		},
		{
			name: "default date does not overwrite existing",
			input: `---
date: 2026-01-01 00:00:00
---`,
			opts: PrepareOptions{Now: fixedNow},
			wantFM: map[string]string{
				"date": "2026-01-01 00:00:00",
			},
		},
		{
			name: "default date fills missing",
			input: `---
title: No date
---`,
			opts: PrepareOptions{Now: fixedNow},
			wantFM: map[string]string{
				"title": "No date",
				"date":  "2026-03-28 14:30:00",
			},
		},
		{
			name: "title not provided preserves existing",
			input: `---
title: Keep me
date: 2026-01-01 00:00:00
---`,
			opts: PrepareOptions{Now: fixedNow},
			wantFM: map[string]string{
				"title": "Keep me",
				"date":  "2026-01-01 00:00:00",
			},
		},
		{
			name: "tags not provided preserves existing",
			input: `---
tags: keep/me, also-me
date: 2026-01-01 00:00:00
---`,
			opts: PrepareOptions{Now: fixedNow},
			wantFM: map[string]string{
				"tags": "keep/me, also-me",
				"date": "2026-01-01 00:00:00",
			},
		},
		{
			name: "unrecognized frontmatter keys preserved",
			input: `---
title: Test
date: 2026-01-01 00:00:00
href: https://example.com
custom: value
---`,
			opts: PrepareOptions{Now: fixedNow},
			wantFM: map[string]string{
				"title":  "Test",
				"date":   "2026-01-01 00:00:00",
				"href":   "https://example.com",
				"custom": "value",
			},
		},
		{
			name:    "malformed yaml returns error",
			input:   "---\n\tbad: [broken\n---\n",
			opts:    PrepareOptions{Now: fixedNow},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r *strings.Reader
			if tt.input != "" {
				r = strings.NewReader(tt.input)
			}

			var note *Note
			var err error
			if r != nil {
				note, err = Prepare(r, tt.opts)
			} else {
				note, err = Prepare(nil, tt.opts)
			}

			if tt.wantErr {
				if err == nil {
					t.Fatal("Prepare() err = <nil>, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Prepare() err = %q", err)
			}

			for k, want := range tt.wantFM {
				got, ok := note.Frontmatter.Get(k)
				if !ok {
					t.Errorf("Frontmatter.Get(%q) not found, want %q", k, want)
				} else if got != want {
					t.Errorf("Frontmatter.Get(%q) = %q, want %q", k, got, want)
				}
			}
		})
	}
}

func TestPrepareComputedFields(t *testing.T) {
	t.Run("title sets slug", func(t *testing.T) {
		note, err := Prepare(nil, PrepareOptions{
			Title: StringPtr("Hello World"),
			Now:   fixedNow,
		})
		if err != nil {
			t.Fatalf("Prepare() err = %q", err)
		}
		if note.Title != "Hello World" {
			t.Errorf("Title = %q, want %q", note.Title, "Hello World")
		}
		if note.Slug != "hello-world" {
			t.Errorf("Slug = %q, want %q", note.Slug, "hello-world")
		}
	})

	t.Run("title overwrites updates slug", func(t *testing.T) {
		input := "---\ntitle: Old Title\ndate: 2026-01-01 00:00:00\n---\n"
		note, err := Prepare(strings.NewReader(input), PrepareOptions{
			Title: StringPtr("New Title"),
			Now:   fixedNow,
		})
		if err != nil {
			t.Fatalf("Prepare() err = %q", err)
		}
		if note.Slug != "new-title" {
			t.Errorf("Slug = %q, want %q", note.Slug, "new-title")
		}
	})

	t.Run("tags parsed into slice", func(t *testing.T) {
		note, err := Prepare(nil, PrepareOptions{
			Tags: []string{"foo/bar", "baz"},
			Now:  fixedNow,
		})
		if err != nil {
			t.Fatalf("Prepare() err = %q", err)
		}
		want := []string{"foo/bar", "baz"}
		if diff := cmp.Diff(want, note.Tags); diff != "" {
			t.Errorf("Tags diff (-want, +got):\n%s", diff)
		}
	})

	t.Run("no title means empty slug", func(t *testing.T) {
		note, err := Prepare(nil, PrepareOptions{Now: fixedNow})
		if err != nil {
			t.Fatalf("Prepare() err = %q", err)
		}
		if note.Title != "" {
			t.Errorf("Title = %q, want empty", note.Title)
		}
		if note.Slug != "" {
			t.Errorf("Slug = %q, want empty", note.Slug)
		}
	})
}

func TestPreparePreservesBody(t *testing.T) {
	input := `---
title: Test
date: 2026-01-01 00:00:00
---

Body with [[20260101-1]] link.

More content.`

	note, err := Prepare(strings.NewReader(input), PrepareOptions{
		Title: StringPtr("Updated"),
		Now:   fixedNow,
	})
	if err != nil {
		t.Fatalf("Prepare() err = %q", err)
	}

	wantBody := "\nBody with [[20260101-1]] link.\n\nMore content."
	if note.Body != wantBody {
		t.Errorf("Body diff:\n%s", cmp.Diff(wantBody, note.Body))
	}

	wantLinks := []string{"20260101-1"}
	if diff := cmp.Diff(wantLinks, note.InternalLinks); diff != "" {
		t.Errorf("InternalLinks diff:\n%s", diff)
	}
}

func TestPrepareExtraFrontmatter(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		opts   PrepareOptions
		wantFM map[string]string
	}{
		{
			name:  "sets custom fields",
			input: "---\ntitle: Test\ndate: 2026-01-01 00:00:00\n---\n",
			opts: PrepareOptions{
				ExtraFrontmatter: []FrontmatterField{
					{Key: "href", Value: "https://example.com"},
					{Key: "author", Value: "Alice"},
				},
				Now: fixedNow,
			},
			wantFM: map[string]string{
				"title":  "Test",
				"date":   "2026-01-01 00:00:00",
				"href":   "https://example.com",
				"author": "Alice",
			},
		},
		{
			name:  "on empty note",
			input: "",
			opts: PrepareOptions{
				ExtraFrontmatter: []FrontmatterField{
					{Key: "custom", Value: "value"},
				},
				Now: fixedNow,
			},
			wantFM: map[string]string{
				"date":   "2026-03-28 14:30:00",
				"custom": "value",
			},
		},
		{
			name: "overwrites existing value",
			input: `---
title: Old
date: 2026-01-01 00:00:00
author: Bob
---
`,
			opts: PrepareOptions{
				ExtraFrontmatter: []FrontmatterField{
					{Key: "author", Value: "Alice"},
				},
				Now: fixedNow,
			},
			wantFM: map[string]string{
				"title":  "Old",
				"date":   "2026-01-01 00:00:00",
				"author": "Alice",
			},
		},
		{
			name:  "combinable with tags",
			input: "---\ndate: 2026-01-01 00:00:00\n---\n",
			opts: PrepareOptions{
				Tags: []string{"foo", "bar"},
				ExtraFrontmatter: []FrontmatterField{
					{Key: "status", Value: "draft"},
				},
				Now: fixedNow,
			},
			wantFM: map[string]string{
				"date":   "2026-01-01 00:00:00",
				"tags":   "foo, bar",
				"status": "draft",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r *strings.Reader
			if tt.input != "" {
				r = strings.NewReader(tt.input)
			}

			var note *Note
			var err error
			if r != nil {
				note, err = Prepare(r, tt.opts)
			} else {
				note, err = Prepare(nil, tt.opts)
			}
			if err != nil {
				t.Fatalf("Prepare() err = %q", err)
			}

			for k, want := range tt.wantFM {
				got, ok := note.Frontmatter.Get(k)
				if !ok {
					t.Errorf("Frontmatter.Get(%q) not found, want %q", k, want)
				} else if got != want {
					t.Errorf("Frontmatter.Get(%q) = %q, want %q", k, got, want)
				}
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
