# gonotes

Personal notes system written in Go.

## Features

- Add notes, optionally with title, tags and/or href meta. (`gonotes new`)
- Rename tags across all notes and update folder structures. (`gonotes rename-tag`)
- Update filenames and folder structures after editing notes. (`gonotes sync`)
- Print file path of last note created. (`gonotes last`)
- List all notes in the repository. (`gonotes list`)
- Print all tags with note count in tree format. (`gonotes tree`)
- Print note contents, which equals file contents. (`gonotes show`)

Upcoming features:

- Asset management (images/pdfs/etc).
- Database as alternative storage.
- GraphQL server.

## Install

```bash
go install github.com/marcelbeumer/gonotes@v2.0.0
```

## Setup

```bash
mkdir mynotes
cd mynotes
touch .is_gonotes_root
gonotes
```
