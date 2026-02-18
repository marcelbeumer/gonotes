package gonotes

import "io/fs"

func NoteFromFS(fs *fs.FS, filepath string) *Note { // maybe remove; let's follow use cases
	// Load file, get reader
	// Get ID from filepath
	// ReadNote
	return nil
}
