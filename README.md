# gonotes

A zettelkasten-like note-taking system backed by plain markdown files and symlinks.

## Note format

Notes are markdown files with optional YAML frontmatter. No fields are mandatory.

```
---
title: My Note
date: 2026-03-28 14:30:00
tags: programming/go, tools
---

Body text with [[20260328-2]] wiki-style internal links.
```

Recognized frontmatter fields:

- **title** -- used for the filename slug and symlinks
- **date** -- used for `notes/by/date/` symlinks
- **tags** -- comma-separated, may be nested (`foo/bar`); used for `notes/by/tags/` symlinks
- **ignore-links** -- comma-separated glob patterns; matching `[[link]]` targets
  are excluded from broken-link checking during `rebuild`. Patterns use
  `filepath.Match` syntax (`*` matches within a single path segment, `?` matches
  one character). Example: `ignore-links: 20260403-99, drafts/*`

All other frontmatter fields are preserved but ignored.

## ID format

IDs follow the format `yyyymmdd-N` (e.g. `20260328-1`). The date prefix groups
notes by day; the number is sequential but allows gaps. Once assigned, an ID
never changes and no date information is inferred from it.

## Directory structure

Source of truth is `notes/by/id/`. Symlinks are derived from frontmatter:

```
notes/by/id/20260328-1-my-note.md              # the actual file
notes/by/date/2026-03-28/20260328-1-my-note.md # symlink
notes/by/tags/nested/programming/go/20260328-1-my-note.md
notes/by/tags/flat/programming/20260328-1-my-note.md
notes/by/tags/flat/go/20260328-1-my-note.md
```

Files are stored in `files/` using ID-based folder names:

```
files/20260403-1-contract-pdfs/doc1.pdf
files/20260403-1-contract-pdfs/doc2.pdf
```

## Usage

```
gonotes <command> [flags]

Commands:
  id         Print the next available note ID
  new        Create a new note
  folder     Create a new folder for file storage
  prepare    Prepare a note: merge frontmatter fields, output to stdout
  rebuild    Scan notes, report issues, rename files, rebuild symlinks
```

**new** creates a note, writes it to `notes/by/id/`, and sets up symlinks:

```
gonotes new -t "My Note" -T "programming/go, tools"
gonotes new -f draft.md
cat draft.md | gonotes new -  # read note from stdin
gonotes new -n                # dry run
```

Flags: `-t` title, `-T` tags, `-d` date (default: now), `-i` id (default: generate),
`-f` file, `-` read from stdin, `-o md|json` (dry run output), `-n` dry run.

**prepare** reads a note, merges frontmatter fields, and writes to stdout:

```
gonotes prepare -t "New Title" -f note.md
gonotes prepare -T "new-tag" -o json -
```

**folder** creates a new directory under `files/` for storing files. The folder
name follows the same ID format as notes (`yyyymmdd-N-slug`):

```
gonotes folder -t "Contract PDFs"  # creates files/20260403-1-contract-pdfs/
gonotes folder                     # creates files/20260403-1/
```

Files can be referenced from notes using wiki-links:

```
See [[20260403-1-contract-pdfs/doc1.pdf]].
```

**rebuild** scans `notes/by/id/`, reports broken links and filename mismatches,
renames files, and rebuilds all symlinks. Link targets are checked against both
note IDs and files under `files/`:

```
gonotes rebuild     # interactive prompts
gonotes rebuild -y  # skip prompts
```
