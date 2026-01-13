package gonotes

// Organize:
// Remove all symlinks.
// Read file by file.
// If something wrong with the file, write to stdout.
// Parse all metadata.
// Determine file path.
// Keep track of file paths, it should be unique.
// shouldWrite = serialized metadata or filepath different
// If shouldWrite, also read body
// If filepath changed, remove old (stdout).
// If shouldWrite, write new contents (stdout).
// Determine and write symlinks.
//
// Add:
// Create new metadata with date, title, tags, etc.
// Determine file path.
// Serialize and write.
// Stdout filepath.
//
// Other old features:
//
// Rename tag: do that with other tools like vim or on cli (add to README)
// Tree: never used it, but this is the symlink file tree (dirs or dirs+files)
// List: never used it
// Show: never used it
// Last: was useful but never mind
// Warnings and checks: now just rely on git
