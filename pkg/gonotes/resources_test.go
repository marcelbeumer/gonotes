package gonotes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoteNameFromUI(t *testing.T) {
	assert.Equal(t, "example", NoteNameFromURI("notes://example"))
	assert.Equal(t, "example", NoteNameFromURI("notes://example.md"))
	assert.Equal(t, "example", NoteNameFromURI("notes://example.foo.bar.md"))

	assert.Equal(t, "", NoteNameFromURI("notes//example"))
}

func TestIsURIRef(t *testing.T) {
	assert.Equal(t, true, IsURIRef("notes://example"))
	assert.Equal(t, true, IsURIRef("notes+notes://example"))

	assert.Equal(t, false, IsURIRef("notes:/example"))
	assert.Equal(t, false, IsURIRef("notes++notes://example"))
	assert.Equal(t, false, IsURIRef("notesnotes://example"))
}

func TestIsNamRef(t *testing.T) {
	assert.Equal(t, true, IsNameRef("example"))
	assert.Equal(t, true, IsNameRef("example"))

	assert.Equal(t, false, IsNameRef("notes://example"))
	assert.Equal(t, false, IsNameRef("notes:/example"))
}

func TestIsPathRef(t *testing.T) {
	assert.Equal(t, true, IsPathRef("/file/path/example"))
	assert.Equal(t, true, IsPathRef("/file/path/example.md"))
	assert.Equal(t, true, IsPathRef("example.md"))
	assert.Equal(t, true, IsPathRef("/example"))
	assert.Equal(t, true, IsPathRef("notes:/example")) // XXX Should it?

	assert.Equal(t, false, IsPathRef("notes://example"))
}

func TestRefToName(t *testing.T) {
	assert.Equal(t, "example", RefToName("/file/path/example.md"))
	assert.Equal(t, "example", RefToName("/file/path/example"))
	assert.Equal(t, "example", RefToName("example.md"))
	assert.Equal(t, "example", RefToName("notes://example"))
	assert.Equal(t, "example", RefToName("example"))
}
