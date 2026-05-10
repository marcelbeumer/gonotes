package gonotes

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var fixedNow = func() time.Time {
	return time.Date(2026, 3, 28, 14, 30, 0, 0, time.Local)
}

func TestPrepare(t *testing.T) {
	tests := []struct {
		name    string
		input   string // empty string means nil reader
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
			name:  "nil reader with all flags",
			input: "",
			opts: PrepareOptions{
				Title: StringPtr("New note"),
				Tags:  StringPtr("foo, bar"),
				Date:  StringPtr("2026-01-01 00:00:00"),
				Now:   fixedNow,
			},
			wantFM: map[string]string{
				"title": "New note",
				"tags":  "foo, bar",
				"date":  "2026-01-01 00:00:00",
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
			name: "explicit tags overwrites existing",
			input: `---
tags: old-tag
date: 2026-01-01 00:00:00
---`,
			opts: PrepareOptions{
				Tags: StringPtr("new/tag, another"),
				Now:  fixedNow,
			},
			wantFM: map[string]string{
				"tags": "new/tag, another",
				"date": "2026-01-01 00:00:00",
			},
		},
		{
			name: "explicit date overwrites existing",
			input: `---
date: 2026-01-01 00:00:00
---`,
			opts: PrepareOptions{
				Date: StringPtr("2026-06-15 12:00:00"),
				Now:  fixedNow,
			},
			wantFM: map[string]string{
				"date": "2026-06-15 12:00:00",
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
			name: "all flags on note with all fields",
			input: `---
title: Old
date: 2026-01-01 00:00:00
tags: old
---

Old body.`,
			opts: PrepareOptions{
				Title: StringPtr("New"),
				Tags:  StringPtr("new/tag"),
				Date:  StringPtr("2026-12-25 00:00:00"),
				Now:   fixedNow,
			},
			wantFM: map[string]string{
				"title": "New",
				"tags":  "new/tag",
				"date":  "2026-12-25 00:00:00",
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
			Tags: StringPtr("foo/bar, baz"),
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

func TestPrepareTagRewrites(t *testing.T) {
	note, err := Prepare(strings.NewReader(`---
tags: programming/go, code/go, keep
---
`), PrepareOptions{
		TagRewrites: []TagRewrite{
			{Match: `^(programming|code)/go$`, Replace: `$1/golang`},
		},
		Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("Prepare() err = %q", err)
	}

	want := []string{"programming/golang", "code/golang", "keep"}
	if diff := cmp.Diff(want, note.Tags); diff != "" {
		t.Errorf("tags diff (-want, +got):\n%s", diff)
	}

	got, ok := note.Frontmatter.Get("tags")
	if !ok {
		t.Fatal("frontmatter tags missing")
	}
	if got != "programming/golang, code/golang, keep" {
		t.Errorf("frontmatter tags = %q", got)
	}
}

func TestPrepareTagRewritesInvalidRegex(t *testing.T) {
	_, err := Prepare(strings.NewReader("---\ntags: foo\n---\n"), PrepareOptions{
		TagRewrites: []TagRewrite{{Match: "(", Replace: "x"}},
		Now:         fixedNow,
	})
	if err == nil {
		t.Fatal("Prepare() err = <nil>, want error")
	}
}

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		fromFS   []string
		want     []string
	}{
		{
			name:     "both empty",
			existing: nil,
			fromFS:   nil,
			want:     nil,
		},
		{
			name:     "existing only",
			existing: []string{"a", "b"},
			fromFS:   []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "new tags from fs",
			existing: []string{"a"},
			fromFS:   []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "tags removed from fs",
			existing: []string{"a", "b"},
			fromFS:   []string{"a"},
			want:     []string{"a"},
		},
		{
			name:     "order preserved existing first new appended",
			existing: []string{"a"},
			fromFS:   []string{"b", "a"},
			want:     []string{"a", "b"},
		},
		{
			name:     "duplicates in existing and fs deduped",
			existing: []string{"a", "a"},
			fromFS:   []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "all tags removed",
			existing: []string{"a", "b"},
			fromFS:   nil,
			want:     nil,
		},
		{
			name:     "all tags new from fs",
			existing: nil,
			fromFS:   []string{"x", "y"},
			want:     []string{"x", "y"},
		},
		{
			name:     "duplicates in fs deduped",
			existing: []string{"a"},
			fromFS:   []string{"b", "a"},
			want:     []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeTags(tt.existing, tt.fromFS)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeTags() diff (-want, +got):\n%s", diff)
			}
		})
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
				Tags: StringPtr("foo, bar"),
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
