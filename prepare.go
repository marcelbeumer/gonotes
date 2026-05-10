package gonotes

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const dateLayout = "2006-01-02 15:04:05"

// PrepareOptions holds explicitly provided values for preparing a note.
// A nil pointer means "not provided"; a non-nil pointer means "explicitly set"
// (even if the pointed-to string is empty).
type PrepareOptions struct {
	Title            *string
	Tags             *string
	Date             *string
	TagRewrites      []TagRewrite
	ExtraFrontmatter []FrontmatterField
	Now              func() time.Time // for testing; defaults to time.Now if nil
}

type TagRewrite struct {
	Match   string
	Replace string
}

type FrontmatterField struct {
	Key   string
	Value string
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

	if len(opts.TagRewrites) > 0 {
		tags, err := rewriteTags(note.Tags, opts.TagRewrites)
		if err != nil {
			return nil, fmt.Errorf("prepare: %w", err)
		}
		note.Tags = tags
		if len(tags) == 0 {
			note.Frontmatter.Unset("tags")
		} else {
			note.Frontmatter.Set("tags", FormatTags(tags))
		}
	}

	for _, f := range opts.ExtraFrontmatter {
		note.Frontmatter.Set(f.Key, f.Value)
	}

	return note, nil
}

func rewriteTags(tags []string, rewrites []TagRewrite) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	type compiledRewrite struct {
		re      *regexp.Regexp
		replace string
	}
	compiled := make([]compiledRewrite, len(rewrites))
	for i, rw := range rewrites {
		re, err := regexp.Compile(rw.Match)
		if err != nil {
			return nil, fmt.Errorf("invalid tag match regex at index %d (%q): %w", i, rw.Match, err)
		}
		compiled[i] = compiledRewrite{re: re, replace: rw.Replace}
	}

	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		updated := tag
		for _, rw := range compiled {
			updated = rw.re.ReplaceAllString(updated, rw.replace)
		}
		updated = strings.TrimSpace(updated)
		if updated == "" {
			continue
		}
		if _, ok := seen[updated]; ok {
			continue
		}
		seen[updated] = struct{}{}
		out = append(out, updated)
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

// StringPtr returns a pointer to s. Convenience for building PrepareOptions.
func StringPtr(s string) *string {
	return &s
}

func mergeTags(existing, fromFS []string) []string {
	fsSet := make(map[string]struct{}, len(fromFS))
	for _, t := range fromFS {
		fsSet[t] = struct{}{}
	}
	var result []string
	seen := make(map[string]struct{})
	for _, t := range existing {
		if _, ok := fsSet[t]; ok {
			if _, ok := seen[t]; !ok {
				result = append(result, t)
				seen[t] = struct{}{}
			}
		}
	}
	for _, t := range fromFS {
		if _, ok := seen[t]; !ok {
			result = append(result, t)
			seen[t] = struct{}{}
		}
	}
	return result
}

// FormatTags joins tags with ", " for use in frontmatter.
func FormatTags(tags []string) string {
	return strings.Join(tags, ", ")
}
