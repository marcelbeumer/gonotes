package gonotes

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const dateLayout = "2006-01-02 15:04:05"

// PrepareOptions holds explicitly provided values for preparing a note.
// A nil pointer means "not provided"; a non-nil pointer means "explicitly set"
// (even if the pointed-to string is empty).
type PrepareOptions struct {
	Title *string
	Tags  *string
	Date  *string
	Now   func() time.Time // for testing; defaults to time.Now if nil
}

// Prepare reads a note from r, merges frontmatter fields per opts, and returns
// the updated note. If r is nil, an empty note is used as the starting point.
//
// Merge semantics:
//   - Explicitly provided values always overwrite.
//   - Defaults fill in missing fields but never overwrite existing values.
//   - The only default is date, which defaults to "now" when absent.
func Prepare(r io.Reader, opts PrepareOptions) (*Note, error) {
	var note *Note
	var err error

	if r != nil {
		note, err = ReadNote("", r)
		if err != nil {
			return nil, fmt.Errorf("prepare: %w", err)
		}
	} else {
		note = NewNote()
	}

	// Explicit values always win.
	if opts.Title != nil {
		note.Frontmatter.Set("title", *opts.Title)
	}
	if opts.Tags != nil {
		note.Frontmatter.Set("tags", *opts.Tags)
	}
	if opts.Date != nil {
		note.Frontmatter.Set("date", *opts.Date)
	}

	// Default: date fills in when missing.
	if _, ok := note.Frontmatter.Get("date"); !ok {
		now := time.Now
		if opts.Now != nil {
			now = opts.Now
		}
		note.Frontmatter.Set("date", now().Local().Format(dateLayout))
	}

	// Re-derive computed fields from (possibly updated) frontmatter.
	if title, ok := note.Frontmatter.Get("title"); ok {
		note.Title = title
		note.Slug = slugify(title)
	} else {
		note.Title = ""
		note.Slug = ""
	}

	if tags, ok := note.Frontmatter.Get("tags"); ok {
		note.Tags = parseTags(tags)
	} else {
		note.Tags = nil
	}

	return note, nil
}

// StringPtr returns a pointer to s. Convenience for building PrepareOptions.
func StringPtr(s string) *string {
	return &s
}

// FormatTags joins tags with ", " for use in frontmatter.
func FormatTags(tags []string) string {
	return strings.Join(tags, ", ")
}
