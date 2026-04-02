# gonotes

Personal notes system written in Go.

This is a rewrite of a gonotes tool in the master branch.

These are the things that I want to achieve:

- The notes system is zettelkasten inspired.
- The repository backbone and source of truth is: `<cwd>/notes/by/id/<id-with-title-slug>.md`. These are the hardcopies. Adding the title slug helps in the text editor.
- ID format is `yyyymmdd-<num>`, for example `20260328-1. The id is unique and never changes. The num tries to be sequential, but allows holes, its only so that the suffix stays simple.
- The ID format date based but once an ID is created it is just a UID: we **never** infer date information from it.
- The program has a `gonotes id` that returns a new id. It does a readdir (in
   go), in batches, so `os.Open` and then `f.ReadDir(<size>)`. It keeps the
   highest `<num>` on the current `yyyymmdd`, and then a +1. It is probably
   most efficient to keep a max int var and match on the specific file format,
   if it matches, parse it, set the max to that value. All with regex.
- It should always be possible to create a id by hand (honering the format).
- The note itself is markdown and starts with frontmatter document and then the note body.
- The frontmatter has no mandatory fields, but it recognizes a few: title, date, tags
  - title: is read to update the slug in the filename and used for the symlinks
  - date: is read to create directory and symlink structure notes/by/date
  - tags: is read to create directory and symlink structure notes/by/tags
- Tags in frontmatter are in format: `tags: my/nested/tag, another-tag, tag/nested/too`
- Internal links in the body are in wiki links format `[[id]]`
- The program has a `gonotes rebuild` that:
  - Read notes/by/id/* file by file, but keeps only one file in memory at once
  - Keeps list of ids collected so far
  - Keeps map of links ot other ids per id so it can check broken links in the end
  - Keeps a list of ids and the current filename
  - Keeps a list of ids and the appropriate filename based on title read
  - In the end gives a report about:
    - broken interna links, just to notify, can't fix them
    - renames of files on notes/by/id. Asks for permission to perform these unless --confirm/-y
    does it.
  - Asks for permission to rebuild symlink structures. If yes or --confirm/-y:
    - Deletes notes/by/date dir
    - Deletes notes/by/tags dir
    - Read notes/by/id/* file by file, keeping only one file in memory at once
    - For each file it symlinks:
      - `notes/by/date/<yyyy-dd-mm>/<id-with-title-slug>.md`
      - `notes/by/tags/nested/<tag>/<nested-tag>/<deeper-nested-tag>/<id-with-title-slug>.md`
      - `notes/by/tags/flat/<tag>/<id-with-title-slug>.md`
      - `notes/by/tags/flat/<nested-tag>/<id-with-title-slug>.md`
      - `notes/by/tags/flat/<deeper-nested-tag>/<id-with-title-slug>.md`
      - Does mkdir -p like behavior for each
- The program has a `gonotes new` that:
  - Takes from stdin with `gonotes new -`
  - Takes from a file with `--file/-f`
  - Takes a title with `--title/-t` otherwise none
  - Takes tags with `--tags/-T` otherwise none
  - Takes date with `--date/-d` otherwise takes "now"
  - Takes id with `--id/-i` otherwise generates it
  - If does not get note content it makes empty note
  - Updates frontmatter of note for title, date, tags:
    - always set what was explictly provided on cli
    - fill missing fields with either provided or defaults
    - defaults never overwrite existing frontmatter values!
  - Writes the file to notes/by/id
  - Rebuilds symlink structure for this file only, ends up on same codepath as
    `gonotes rebuild` for a single file.

Code structure:

- frontmatter.go does only that, with get, set, remove
- note.go parses the note into frontmatter, content, tags, id, title, slug, checks internal links in content (nice to have)
- fs.go reads from by/id/* and rebuilds symlink structures

Explicity:

- No more `.is_gonotes_root` file
- No more tree command, no search things, we have the fs.

Codestyle and general directions for refactor:

- Idiomatic Go code following Google Go Styleguide, a superset of Effective Go.
- The code in master is not always idiomatic and only serves as reference.
- Everything should be unit tested. It's best to have a few good integration
  tests, and then unit tests to check different edge cases (and normal cases
  too ofc)
- Entrypoint is cmd/gonotes. Library code lives in the root of the repo
  (note.go, frontmatter.go, fs.go, etc.).
- The general idea is to keep things simple, small and flat.
