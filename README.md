# gonotes

Personal notes system written in Go.

## Features

- Always possible to create note by hand
- Create a new note via CLI `gonotes new` with optional -id, -title, -tags
- Parse markdown and get JSON back (nice to have)
- Parse JSON and get markdown back (nice to have)
- Repo is `/notes/by/id/<id>`, `/notes/by/date/../<symlink-id-title-slug>` `/notes/by/tag/.../<symlink-id-title-slug>`
- Add note via file or stdin (like kubectl), add -id=<id> or -infer-id
  - Error when id already exists (user can delete manually)
- Sync repo with note contents `gonotes sync`, does file by file (to be elegant)
- Internal links are checked `[[id]]`
- ID format is `yyyymmdd-num`
- CLI tool can give new id `gonotes id`
- Readme with examples how to change tags with CLI tools like sed
- No longer .is_gonotes_root, can just check notes/by/id existance or create that.
- No tree command, no search things, we have the fs.

## Code structure

- frontmatter.go does only that, with get, set, remove (support JSON; nice to have)
- note.go parses the note into frontmatter, content, tags, id, title, slug (and supports JSON/MD in&out; nice to have), checks internal links in content (nice to have)
- sync.go reads from by/id/* and rebuilds symlink structures
