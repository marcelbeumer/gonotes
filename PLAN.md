# PLAN.md

## Current task

I want to support storing not only notes but also files.

Currently the system supports medialinks like `[[id]]` and it will check for
broken links too.

I want to implement that medialinks also support links to files with
`[[file]]`. 

For checking broken links in `gonotes rebuild`, I think it makes sense to read
also the entire list files so it can check first if there is a note what id,
and when not, if there is a file with that filename.

For organizing files I am imagining using a similar mechanmism as for note ids,
but then generate directories, so the files themselves can keep their original
filenames and I do not need to worry about renaming.

Effectively this means my file links will always have a slash (/) in them, but
I do not want the code to make assumptions about that, like written above, a
[[link]] is the same as a [[path/to/file]] or [[file]] in the logic of `gonotes
rebuild`.

Subfolders should be supported.

This is the usage I imagine:

- I have a file I want to add to my note.
- I run `gonotes folder -t my-title`, and the program will create a directory
  (mkdir -p style) in format `files/yyyymmdd-<nu>-<title>/` very similar to
  note ids, for example `20260403-1-contract-pdfs`. The title is optional, just
  like with a note.
- I drop the file in the folder using the OS.
- I type text in the note: `[[20260403-1-contract-pdfs/doc1.pdf]]`.

As I test I can then

- Change the link to: `[[20260403-1-contract-pdfs/bad.pdf]]`.
- Run `gonotes rebuild`. This will do what it does now but also read all the
  files recursively and report a broken link because this file does not exist.

## Code structure

The fs.go file will now need to also read all files and use that in the rebuild
logic to detect broken links.

There will need to be a function in fs.go that implements creating a new folder
according so that main.go in cmd/gonotes can call it for `gonotes folder` (with
optional -t)`

Maybe it makes sense to have a files.go, but then again, maybe it's all small
enough to just fit it in fs.go, such separation may only confuse...

## Plan

Done.

1. Added `FolderName(id, slug)` in fs.go -- returns `id-slug` directory name.
2. Added `CreateFolder(baseDir, title, now)` in fs.go -- generates next ID from
   `files/` dir, creates `files/yyyymmdd-N-slug/`.
3. Changed `ScanNotes(idDir)` to `ScanNotes(baseDir)` -- derives `idDir` and
   `filesDir` internally.
4. Extended broken-link checking -- after failing note ID lookup, does
   `os.Stat(files/<target>)`. Targets with `/` skip note ID normalization.
5. Added `gonotes folder [-t title]` command in main.go.
6. Tests: `TestFolderName`, `TestCreateFolder`, `TestCreateFolderSequentialIDs`,
   `TestCreateFolderNoTitle`, `TestScanNotesFileLinks`,
   `TestScanNotesFileLinksNoFilesDir`, `TestScanNotesFileLinkAndNoteLink`.
7. Updated README.md with folder command docs and file storage docs.
