package gonotes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWikiLinkAtPosition(t *testing.T) {
	tests := []struct {
		name    string
		content string
		line    int
		col     int
		want    string
	}{
		{
			name:    "cursor inside link",
			content: "some text [[20220601-1]] more text",
			line:    0,
			col:     15,
			want:    "20220601-1",
		},
		{
			name:    "cursor on opening brackets",
			content: "some text [[20220601-1]] more text",
			line:    0,
			col:     10,
			want:    "20220601-1",
		},
		{
			name:    "cursor on closing bracket",
			content: "some text [[20220601-1]] more text",
			line:    0,
			col:     22,
			want:    "20220601-1",
		},
		{
			name:    "cursor on last closing bracket",
			content: "some text [[20220601-1]] more text",
			line:    0,
			col:     23,
			want:    "20220601-1",
		},
		{
			name:    "cursor after closing brackets",
			content: "some text [[20220601-1]] more text",
			line:    0,
			col:     24,
			want:    "",
		},
		{
			name:    "cursor before link",
			content: "some text [[20220601-1]] more text",
			line:    0,
			col:     5,
			want:    "",
		},
		{
			name:    "link with slug",
			content: "see [[20220601-1-my-title]] here",
			line:    0,
			col:     10,
			want:    "20220601-1-my-title",
		},
		{
			name:    "multiple links, cursor on second",
			content: "[[20220601-1]] and [[20220601-2]]",
			line:    0,
			col:     22,
			want:    "20220601-2",
		},
		{
			name:    "multiline, cursor on second line",
			content: "first line\nsee [[20220601-1]] here\nthird line",
			line:    1,
			col:     8,
			want:    "20220601-1",
		},
		{
			name:    "line out of range",
			content: "one line",
			line:    5,
			col:     0,
			want:    "",
		},
		{
			name:    "file link with path",
			content: "see [[20220601-1-docs/file.pdf]]",
			line:    0,
			col:     10,
			want:    "20220601-1-docs/file.pdf",
		},
		{
			name:    "empty content",
			content: "",
			line:    0,
			col:     0,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wikiLinkAtPosition(tt.content, tt.line, tt.col)
			if got != tt.want {
				t.Errorf("wikiLinkAtPosition(..., %d, %d) = %q, want %q", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestTagAtPosition(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want string
	}{
		{
			name: "cursor on first tag",
			line: "tags: programming/go, tools",
			col:  8,
			want: "programming/go",
		},
		{
			name: "cursor on second tag",
			line: "tags: programming/go, tools",
			col:  24,
			want: "tools",
		},
		{
			name: "cursor on tag start",
			line: "tags: programming/go, tools",
			col:  6,
			want: "programming/go",
		},
		{
			name: "cursor between tags (on space after comma)",
			line: "tags: programming/go, tools",
			col:  21, // the space after the comma
			want: "",
		},
		{
			name: "single tag",
			line: "tags: mytag",
			col:  7,
			want: "mytag",
		},
		{
			name: "not a tags line",
			line: "title: My Title",
			col:  8,
			want: "",
		},
		{
			name: "cursor on key part",
			line: "tags: foo, bar",
			col:  2,
			want: "",
		},
		{
			name: "three tags, cursor on middle",
			line: "tags: alpha, beta, gamma",
			col:  14,
			want: "beta",
		},
		{
			name: "extra spaces around tags",
			line: "tags:  foo ,  bar",
			col:  7,
			want: "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagAtPosition(tt.line, tt.col)
			if got != tt.want {
				t.Errorf("tagAtPosition(%q, %d) = %q, want %q", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestTagSegmentAtPosition(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want string
	}{
		{
			name: "cursor on first segment",
			line: "tags: programming/go, tools",
			col:  8,
			want: "programming",
		},
		{
			name: "cursor on second segment",
			line: "tags: programming/go, tools",
			col:  19,
			want: "go",
		},
		{
			name: "single segment tag",
			line: "tags: tools",
			col:  7,
			want: "tools",
		},
		{
			name: "three segments, cursor on middle",
			line: "tags: a/b/c",
			col:  8,
			want: "b",
		},
		{
			name: "three segments, cursor on last",
			line: "tags: a/b/c",
			col:  10,
			want: "c",
		},
		{
			name: "cursor not on tag value",
			line: "tags: programming/go, tools",
			col:  2,
			want: "",
		},
		{
			name: "not a tags line",
			line: "title: My Title",
			col:  8,
			want: "",
		},
		{
			name: "second tag, nested, cursor on first segment",
			line: "tags: tools, programming/go",
			col:  14,
			want: "programming",
		},
		{
			name: "second tag, nested, cursor on second segment",
			line: "tags: tools, programming/go",
			col:  26,
			want: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagSegmentAtPosition(tt.line, tt.col)
			if got != tt.want {
				t.Errorf("tagSegmentAtPosition(%q, %d) = %q, want %q", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestLinkTargetToID(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		{"20220601-1", "20220601-1"},
		{"20220601-1-my-title", "20220601-1"},
		{"20220601-1-docs/file.pdf", "20220601-1-docs/file.pdf"},
		{"some-arbitrary-name", "some-arbitrary-name"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := linkTargetToID(tt.target)
			if got != tt.want {
				t.Errorf("linkTargetToID(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestIsInFrontmatter(t *testing.T) {
	content := "---\ntitle: Hello\ntags: foo\n---\nbody text"
	tests := []struct {
		line int
		want bool
	}{
		{0, false}, // the --- line itself
		{1, true},  // title: Hello
		{2, true},  // tags: foo
		{3, false}, // the closing ---
		{4, false}, // body text
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("line_%d", tt.line), func(t *testing.T) {
			got := isInFrontmatter(content, tt.line)
			if got != tt.want {
				t.Errorf("isInFrontmatter(..., %d) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFrontmatterKeyAtLine(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"title: My Title", "title"},
		{"tags: foo, bar", "tags"},
		{"date: 2026-01-01 10:00:00", "date"},
		{"ignore-links: foo*, bar*", "ignore-links"},
		{"no colon here", ""},
		{": no key", ""},
		{"  spaced key: value", ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := frontmatterKeyAtLine(tt.line)
			if got != tt.want {
				t.Errorf("frontmatterKeyAtLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestFindFileForID(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"20220601-1-my-title.md",
		"20220601-2.md",
		"20220602-1-other.md",
		"readme.txt",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		id   string
		want string
	}{
		{"20220601-1", "20220601-1-my-title.md"},
		{"20220601-2", "20220601-2.md"},
		{"20220602-1", "20220602-1-other.md"},
		{"20220603-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got, err := findFileForID(dir, tt.id)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("findFileForID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestFindFileForID_noDir(t *testing.T) {
	got, err := findFileForID(filepath.Join(t.TempDir(), "nonexistent"), "20220601-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty result for nonexistent dir, got %q", got)
	}
}

func TestFindReferencesToID(t *testing.T) {
	dir := t.TempDir()

	// Note that links to 20220601-1.
	writeTestFile(t, dir, "20220602-1-linker.md",
		"---\ntitle: Linker\n---\nSee [[20220601-1]] and [[20220601-1-my-title]].\n")

	// Note that links to something else.
	writeTestFile(t, dir, "20220602-2-other.md",
		"---\ntitle: Other\n---\nSee [[20220603-1]].\n")

	// Note with no links.
	writeTestFile(t, dir, "20220601-1-my-title.md",
		"---\ntitle: My Title\n---\nNo links here.\n")

	locs, err := findReferencesToID(dir, "20220601-1")
	if err != nil {
		t.Fatal(err)
	}

	// Should find 2 references in the linker file.
	if len(locs) != 2 {
		t.Fatalf("got %d locations, want 2", len(locs))
	}

	// Both should be from the linker file, on line 3.
	for _, loc := range locs {
		wantURI := pathToURI(filepath.Join(dir, "20220602-1-linker.md"))
		if loc.URI != wantURI {
			t.Errorf("loc.URI = %q, want %q", loc.URI, wantURI)
		}
		if loc.Range.Start.Line != 3 {
			t.Errorf("loc.Range.Start.Line = %d, want 3", loc.Range.Start.Line)
		}
	}
}

func TestFindReferencesToID_noDir(t *testing.T) {
	locs, err := findReferencesToID(filepath.Join(t.TempDir(), "nonexistent"), "20220601-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(locs) != 0 {
		t.Errorf("expected no results for nonexistent dir, got %d", len(locs))
	}
}

func TestURIConversion(t *testing.T) {
	tests := []struct {
		path string
	}{
		{"/home/user/notes/note.md"},
		{"/tmp/test.md"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			uri := pathToURI(tt.path)
			got := uriToPath(uri)
			if got != tt.path {
				t.Errorf("round-trip: pathToURI(%q) = %q, uriToPath(...) = %q", tt.path, uri, got)
			}
		})
	}
}

func TestServeLSP_InitializeAndDefinition(t *testing.T) {
	// Set up a notes directory.
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, idDir, "20220601-1-my-title.md",
		"---\ntitle: My Title\n---\nSome content.\n")

	writeTestFile(t, idDir, "20220602-1-linker.md",
		"---\ntitle: Linker\n---\nSee [[20220601-1]].\n")

	// Build JSON-RPC messages.
	var input bytes.Buffer

	// 1. initialize
	writeRPCMessage(t, &input, 1, "initialize", map[string]any{
		"rootUri": pathToURI(baseDir),
	})

	// 2. initialized notification
	writeRPCNotification(t, &input, "initialized", map[string]any{})

	// 3. textDocument/definition on the link in linker.md
	writeRPCMessage(t, &input, 2, "textDocument/definition", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(filepath.Join(idDir, "20220602-1-linker.md")),
		},
		"position": map[string]int{
			"line":      3,
			"character": 8,
		},
	})

	// 4. shutdown
	writeRPCMessage(t, &input, 3, "shutdown", nil)

	// 5. exit
	writeRPCNotification(t, &input, "exit", nil)

	// Run the server (exit will call os.Exit, so we skip it by not sending it
	// and instead relying on EOF from the input).
	// Actually, remove the exit notification and rely on EOF.
	input.Reset()
	writeRPCMessage(t, &input, 1, "initialize", map[string]any{
		"rootUri": pathToURI(baseDir),
	})
	writeRPCNotification(t, &input, "initialized", map[string]any{})
	writeRPCMessage(t, &input, 2, "textDocument/definition", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(filepath.Join(idDir, "20220602-1-linker.md")),
		},
		"position": map[string]int{
			"line":      3,
			"character": 8,
		},
	})
	writeRPCMessage(t, &input, 3, "shutdown", nil)

	var output bytes.Buffer
	err := ServeLSP(&input, &output, LSPOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Parse responses.
	responses := parseRPCResponses(t, output.Bytes())

	// Should have 3 responses: initialize, definition, shutdown.
	if len(responses) != 3 {
		t.Fatalf("got %d responses, want 3", len(responses))
	}

	// Check initialize response has capabilities.
	var initResult lspInitializeResult
	if err := json.Unmarshal(responses[0].Result, &initResult); err != nil {
		t.Fatal(err)
	}
	if !initResult.Capabilities.DefinitionProvider {
		t.Error("DefinitionProvider should be true")
	}
	if !initResult.Capabilities.ReferencesProvider {
		t.Error("ReferencesProvider should be true")
	}

	// Check definition response points to the target file.
	var defResult lspLocation
	if err := json.Unmarshal(responses[1].Result, &defResult); err != nil {
		t.Fatal(err)
	}
	wantURI := pathToURI(filepath.Join(idDir, "20220601-1-my-title.md"))
	if defResult.URI != wantURI {
		t.Errorf("definition URI = %q, want %q", defResult.URI, wantURI)
	}
}

func TestServeLSP_References(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	tagsDir := filepath.Join(baseDir, "notes", "by", "tags", "nested", "programming", "go")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tagsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, idDir, "20220601-1-target.md",
		"---\ntitle: Target\ntags: programming/go\n---\nTarget content.\n")

	writeTestFile(t, idDir, "20220602-1-linker.md",
		"---\ntitle: Linker\n---\nSee [[20220601-1]].\n")

	writeTestFile(t, idDir, "20220602-2-another.md",
		"---\ntitle: Another\n---\nAlso see [[20220601-1-target]].\n")

	// Create symlink for tag.
	relTarget, _ := filepath.Rel(tagsDir, filepath.Join(idDir, "20220601-1-target.md"))
	os.Symlink(relTarget, filepath.Join(tagsDir, "20220601-1-target.md"))

	// Test 1: references on a wiki-link.
	var input bytes.Buffer
	writeRPCMessage(t, &input, 1, "initialize", map[string]any{
		"rootUri": pathToURI(baseDir),
	})
	writeRPCNotification(t, &input, "initialized", map[string]any{})
	writeRPCMessage(t, &input, 2, "textDocument/references", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(filepath.Join(idDir, "20220602-1-linker.md")),
		},
		"position": map[string]int{
			"line":      3,
			"character": 8,
		},
		"context": map[string]bool{
			"includeDeclaration": false,
		},
	})

	// Test 2: references on a tag.
	writeRPCMessage(t, &input, 3, "textDocument/references", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(filepath.Join(idDir, "20220601-1-target.md")),
		},
		"position": map[string]int{
			"line":      2,
			"character": 8,
		},
		"context": map[string]bool{
			"includeDeclaration": false,
		},
	})

	// Test 3: references on a title.
	writeRPCMessage(t, &input, 4, "textDocument/references", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(filepath.Join(idDir, "20220601-1-target.md")),
		},
		"position": map[string]int{
			"line":      1,
			"character": 10,
		},
		"context": map[string]bool{
			"includeDeclaration": false,
		},
	})

	writeRPCMessage(t, &input, 5, "shutdown", nil)

	var output bytes.Buffer
	if err := ServeLSP(&input, &output, LSPOptions{}); err != nil {
		t.Fatal(err)
	}

	responses := parseRPCResponses(t, output.Bytes())
	if len(responses) != 5 {
		t.Fatalf("got %d responses, want 5", len(responses))
	}

	// Response 1 (id=2): references on wiki-link [[20220601-1]].
	// Should find references in both linker.md and another.md.
	var linkRefs []lspLocation
	if err := json.Unmarshal(responses[1].Result, &linkRefs); err != nil {
		t.Fatal(err)
	}
	if len(linkRefs) != 2 {
		t.Errorf("wiki-link references: got %d, want 2", len(linkRefs))
	}

	// Response 2 (id=3): references on tag "programming/go".
	// Should find the target file via symlink.
	var tagRefs []lspLocation
	if err := json.Unmarshal(responses[2].Result, &tagRefs); err != nil {
		t.Fatal(err)
	}
	if len(tagRefs) != 1 {
		t.Errorf("tag references: got %d, want 1", len(tagRefs))
	}

	// Response 3 (id=4): references on title of target.md.
	// Should find references in both linker.md and another.md.
	var titleRefs []lspLocation
	if err := json.Unmarshal(responses[3].Result, &titleRefs); err != nil {
		t.Fatal(err)
	}
	if len(titleRefs) != 2 {
		t.Errorf("title references: got %d, want 2", len(titleRefs))
	}
}

func TestServeLSP_ReferencesFlat(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	flatGoDir := filepath.Join(baseDir, "notes", "by", "tags", "flat", "go")
	flatProgDir := filepath.Join(baseDir, "notes", "by", "tags", "flat", "programming")
	for _, d := range []string{idDir, flatGoDir, flatProgDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, idDir, "20220601-1-target.md",
		"---\ntitle: Target\ntags: programming/go\n---\nTarget content.\n")

	writeTestFile(t, idDir, "20220602-1-also-go.md",
		"---\ntitle: Also Go\ntags: programming/go\n---\nMore go content.\n")

	// Create flat symlinks for "go" segment.
	for _, name := range []string{"20220601-1-target.md", "20220602-1-also-go.md"} {
		relTarget, _ := filepath.Rel(flatGoDir, filepath.Join(idDir, name))
		os.Symlink(relTarget, filepath.Join(flatGoDir, name))
		relTarget2, _ := filepath.Rel(flatProgDir, filepath.Join(idDir, name))
		os.Symlink(relTarget2, filepath.Join(flatProgDir, name))
	}

	// Request references with cursor on "go" in "tags: programming/go"
	//                                             0123456789012345678
	// "tags: programming/go" — "go" starts at col 19
	var input bytes.Buffer
	writeRPCMessage(t, &input, 1, "initialize", map[string]any{
		"rootUri": pathToURI(baseDir),
	})
	writeRPCNotification(t, &input, "initialized", map[string]any{})
	writeRPCMessage(t, &input, 2, "textDocument/references", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(filepath.Join(idDir, "20220601-1-target.md")),
		},
		"position": map[string]int{
			"line":      2,
			"character": 19,
		},
		"context": map[string]bool{
			"includeDeclaration": false,
		},
	})

	// Request references with cursor on "programming" in "tags: programming/go"
	// "programming" starts at col 6
	writeRPCMessage(t, &input, 3, "textDocument/references", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(filepath.Join(idDir, "20220601-1-target.md")),
		},
		"position": map[string]int{
			"line":      2,
			"character": 8,
		},
		"context": map[string]bool{
			"includeDeclaration": false,
		},
	})

	writeRPCMessage(t, &input, 4, "shutdown", nil)

	var output bytes.Buffer
	if err := ServeLSP(&input, &output, LSPOptions{FlatTags: true}); err != nil {
		t.Fatal(err)
	}

	responses := parseRPCResponses(t, output.Bytes())
	if len(responses) != 4 {
		t.Fatalf("got %d responses, want 4", len(responses))
	}

	// Response for id=2: cursor on "go" → flat/go/ → 2 notes.
	var goRefs []lspLocation
	if err := json.Unmarshal(responses[1].Result, &goRefs); err != nil {
		t.Fatal(err)
	}
	if len(goRefs) != 2 {
		t.Errorf("flat 'go' references: got %d, want 2", len(goRefs))
	}

	// Response for id=3: cursor on "programming" → flat/programming/ → 2 notes.
	var progRefs []lspLocation
	if err := json.Unmarshal(responses[2].Result, &progRefs); err != nil {
		t.Fatal(err)
	}
	if len(progRefs) != 2 {
		t.Errorf("flat 'programming' references: got %d, want 2", len(progRefs))
	}
}

func TestExtractFrontmatterTags(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "tags present",
			content: "---\ntitle: Hello\ntags: programming/go, tools\n---\nBody text.\n",
			want:    "programming/go, tools",
		},
		{
			name:    "no tags",
			content: "---\ntitle: Hello\n---\nBody text.\n",
			want:    "",
		},
		{
			name:    "no frontmatter",
			content: "Just body text.\n",
			want:    "",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "single tag",
			content: "---\ntags: mytag\n---\n",
			want:    "mytag",
		},
		{
			name:    "tags with extra spaces",
			content: "---\ntags:  foo , bar \n---\n",
			want:    "foo , bar",
		},
		{
			name:    "tags not in frontmatter",
			content: "---\ntitle: Hello\n---\ntags: not-frontmatter\n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFrontmatterTags(tt.content)
			if got != tt.want {
				t.Errorf("extractFrontmatterTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsInsideWikiLink(t *testing.T) {
	tests := []struct {
		name    string
		content string
		line    int
		col     int
		want    bool
	}{
		{
			name:    "cursor right after [[",
			content: "text [[",
			line:    0,
			col:     7,
			want:    true,
		},
		{
			name:    "cursor typing inside [[",
			content: "text [[20220",
			line:    0,
			col:     12,
			want:    true,
		},
		{
			name:    "cursor inside closed link",
			content: "text [[20220601-1]]",
			line:    0,
			col:     10,
			want:    true,
		},
		{
			name:    "cursor after closed link",
			content: "text [[20220601-1]] more",
			line:    0,
			col:     22,
			want:    false,
		},
		{
			name:    "single bracket only",
			content: "text [",
			line:    0,
			col:     6,
			want:    false,
		},
		{
			name:    "no brackets",
			content: "plain text",
			line:    0,
			col:     5,
			want:    false,
		},
		{
			name:    "second unclosed link on same line",
			content: "text [[20220601-1]] more [[",
			line:    0,
			col:     27,
			want:    true,
		},
		{
			name:    "multiline, cursor on second line",
			content: "first line\ntext [[note",
			line:    1,
			col:     11,
			want:    true,
		},
		{
			name:    "cursor at col 0",
			content: "text [[note",
			line:    0,
			col:     0,
			want:    false,
		},
		{
			name:    "empty content",
			content: "",
			line:    0,
			col:     0,
			want:    false,
		},
		{
			name:    "line out of range",
			content: "one line",
			line:    5,
			col:     0,
			want:    false,
		},
		{
			name:    "col beyond line length",
			content: "text [[",
			line:    0,
			col:     100,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInsideWikiLink(tt.content, tt.line, tt.col)
			if got != tt.want {
				t.Errorf("isInsideWikiLink(%q, %d, %d) = %v, want %v",
					tt.content, tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestGetDocumentContent(t *testing.T) {
	dir := t.TempDir()
	idDir := filepath.Join(dir, "notes", "by", "id")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, idDir, "20220601-1-test.md", "disk content")
	filePath := filepath.Join(idDir, "20220601-1-test.md")
	uri := pathToURI(filePath)

	srv := &lspServer{
		rootDir: dir,
		docs:    make(map[string]string),
	}

	// Without didOpen, should read from disk.
	content, err := srv.getDocumentContent(uri)
	if err != nil {
		t.Fatal(err)
	}
	if content != "disk content" {
		t.Errorf("got %q, want %q", content, "disk content")
	}

	// After didOpen, should return buffer content.
	srv.docs[uri] = "buffer content"
	content, err = srv.getDocumentContent(uri)
	if err != nil {
		t.Fatal(err)
	}
	if content != "buffer content" {
		t.Errorf("got %q, want %q", content, "buffer content")
	}

	// After didClose, should fall back to disk.
	delete(srv.docs, uri)
	content, err = srv.getDocumentContent(uri)
	if err != nil {
		t.Fatal(err)
	}
	if content != "disk content" {
		t.Errorf("got %q, want %q", content, "disk content")
	}
}

func TestServeLSP_Completion(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	filesDir := filepath.Join(baseDir, "files", "20220601-3-docs")
	for _, d := range []string{idDir, filesDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, idDir, "20220601-1-alpha.md",
		"---\ntitle: Alpha\ntags: programming/go, tools\n---\nAlpha content.\n")
	writeTestFile(t, idDir, "20220601-2-beta.md",
		"---\ntitle: Beta\n---\nBeta content.\n")
	writeTestFile(t, filesDir, "contract.pdf", "pdf bytes")

	editorFile := filepath.Join(idDir, "20220602-1-editor.md")
	writeTestFile(t, idDir, "20220602-1-editor.md",
		"---\ntitle: Editor\n---\nSee [[")

	var input bytes.Buffer

	// 1. initialize
	writeRPCMessage(t, &input, 1, "initialize", map[string]any{
		"rootUri": pathToURI(baseDir),
	})

	// 2. initialized
	writeRPCNotification(t, &input, "initialized", map[string]any{})

	// 3. didOpen with content containing [[
	writeRPCNotification(t, &input, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        pathToURI(editorFile),
			"languageId": "markdown",
			"version":    1,
			"text":       "---\ntitle: Editor\n---\nSee [[",
		},
	})

	// 4. completion after [[
	writeRPCMessage(t, &input, 2, "textDocument/completion", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(editorFile),
		},
		"position": map[string]int{
			"line":      3,
			"character": 7, // after "See [["
		},
		"context": map[string]any{
			"triggerKind":      2,
			"triggerCharacter": "[",
		},
	})

	// 5. completion NOT inside [[ (cursor on line 1)
	writeRPCMessage(t, &input, 3, "textDocument/completion", map[string]any{
		"textDocument": map[string]string{
			"uri": pathToURI(editorFile),
		},
		"position": map[string]int{
			"line":      1,
			"character": 5,
		},
	})

	// 6. shutdown
	writeRPCMessage(t, &input, 4, "shutdown", nil)

	var output bytes.Buffer
	if err := ServeLSP(&input, &output, LSPOptions{}); err != nil {
		t.Fatal(err)
	}

	responses := parseRPCResponses(t, output.Bytes())
	if len(responses) != 4 {
		t.Fatalf("got %d responses, want 4", len(responses))
	}

	// Response 0 (id=1): initialize - check capabilities.
	var initResult lspInitializeResult
	if err := json.Unmarshal(responses[0].Result, &initResult); err != nil {
		t.Fatal(err)
	}
	if initResult.Capabilities.CompletionProvider == nil {
		t.Fatal("CompletionProvider should not be nil")
	}
	if initResult.Capabilities.TextDocumentSync == nil {
		t.Fatal("TextDocumentSync should not be nil")
	}

	// Response 1 (id=2): completion inside [[ - should have items.
	var completionResult lspCompletionList
	if err := json.Unmarshal(responses[1].Result, &completionResult); err != nil {
		t.Fatal(err)
	}
	// Should have 3 notes (alpha, beta, editor) + 1 file.
	if len(completionResult.Items) != 4 {
		t.Errorf("completion items: got %d, want 4", len(completionResult.Items))
		for _, item := range completionResult.Items {
			t.Logf("  item: %q", item.Label)
		}
	}

	// Verify we have both note stems and the file path.
	items := make(map[string]lspCompletionItem)
	for _, item := range completionResult.Items {
		items[item.Label] = item
		if item.Kind != 17 {
			t.Errorf("item %q kind = %d, want 17", item.Label, item.Kind)
		}
	}
	for _, want := range []string{"20220601-1-alpha", "20220601-2-beta", "20220601-3-docs/contract.pdf"} {
		if _, ok := items[want]; !ok {
			t.Errorf("missing expected completion item %q", want)
		}
	}

	// Verify Detail and LabelDetails (tags) on note items.
	if got := items["20220601-1-alpha"].Detail; got != "programming/go, tools" {
		t.Errorf("alpha detail = %q, want %q", got, "programming/go, tools")
	}
	if items["20220601-1-alpha"].LabelDetails == nil {
		t.Error("alpha labelDetails should not be nil")
	} else if got := items["20220601-1-alpha"].LabelDetails.Description; got != "programming/go, tools" {
		t.Errorf("alpha labelDetails.description = %q, want %q", got, "programming/go, tools")
	}
	if got := items["20220601-2-beta"].Detail; got != "" {
		t.Errorf("beta detail = %q, want empty", got)
	}
	if items["20220601-2-beta"].LabelDetails != nil {
		t.Errorf("beta labelDetails should be nil, got %+v", items["20220601-2-beta"].LabelDetails)
	}
	if got := items["20220601-3-docs/contract.pdf"].Detail; got != "" {
		t.Errorf("file detail = %q, want empty", got)
	}

	// Response 2 (id=3): completion NOT inside [[ - should be null.
	if string(responses[2].Result) != "null" {
		t.Errorf("expected null for completion outside [[, got %s", responses[2].Result)
	}
}

func TestServeLSP_CompletionIndexUpdate(t *testing.T) {
	baseDir := t.TempDir()
	idDir := filepath.Join(baseDir, "notes", "by", "id")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, idDir, "20220601-1-alpha.md",
		"---\ntitle: Alpha\n---\nAlpha content.\n")

	// Use a separate tmp file outside notes/by/id/ for the editor buffer,
	// so we can precisely control the index count.
	editorContent := "---\ntitle: Editor\n---\nSee [["
	editorURI := "file:///tmp/editor-buffer.md"

	var input bytes.Buffer

	// 1. initialize - at this point only alpha.md is in notes/by/id/.
	writeRPCMessage(t, &input, 1, "initialize", map[string]any{
		"rootUri": pathToURI(baseDir),
	})
	writeRPCNotification(t, &input, "initialized", map[string]any{})

	// 2. didOpen with buffer content containing [[
	writeRPCNotification(t, &input, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        editorURI,
			"languageId": "markdown",
			"version":    1,
			"text":       editorContent,
		},
	})

	// 3. completion - should have 1 note (alpha).
	writeRPCMessage(t, &input, 2, "textDocument/completion", map[string]any{
		"textDocument": map[string]string{
			"uri": editorURI,
		},
		"position": map[string]int{
			"line":      3,
			"character": 7,
		},
	})

	// 4. Notify via didChangeWatchedFiles that a new note was created.
	// The file doesn't need to exist on disk — indexAddFile only parses the URI.
	newNoteURI := pathToURI(filepath.Join(idDir, "20220601-2-beta.md"))

	writeRPCNotification(t, &input, "workspace/didChangeWatchedFiles", map[string]any{
		"changes": []map[string]any{
			{"uri": newNoteURI, "type": 1}, // Created
		},
	})

	// 5. completion again - should now have 2 notes (alpha + beta).
	writeRPCMessage(t, &input, 3, "textDocument/completion", map[string]any{
		"textDocument": map[string]string{
			"uri": editorURI,
		},
		"position": map[string]int{
			"line":      3,
			"character": 7,
		},
	})

	// 6. Delete alpha via didChangeWatchedFiles.
	writeRPCNotification(t, &input, "workspace/didChangeWatchedFiles", map[string]any{
		"changes": []map[string]any{
			{"uri": pathToURI(filepath.Join(idDir, "20220601-1-alpha.md")), "type": 3}, // Deleted
		},
	})

	// 7. completion again - should now have 1 note (beta).
	writeRPCMessage(t, &input, 4, "textDocument/completion", map[string]any{
		"textDocument": map[string]string{
			"uri": editorURI,
		},
		"position": map[string]int{
			"line":      3,
			"character": 7,
		},
	})

	// 8. shutdown
	writeRPCMessage(t, &input, 5, "shutdown", nil)

	var output bytes.Buffer
	if err := ServeLSP(&input, &output, LSPOptions{}); err != nil {
		t.Fatal(err)
	}

	responses := parseRPCResponses(t, output.Bytes())
	if len(responses) != 5 {
		t.Fatalf("got %d responses, want 5", len(responses))
	}

	// Response 1 (id=2): initial completion - 1 note (alpha).
	var comp1 lspCompletionList
	if err := json.Unmarshal(responses[1].Result, &comp1); err != nil {
		t.Fatal(err)
	}
	if len(comp1.Items) != 1 {
		t.Errorf("initial completion: got %d items, want 1", len(comp1.Items))
	}

	// Response 2 (id=3): after create - 2 notes (alpha + beta).
	var comp2 lspCompletionList
	if err := json.Unmarshal(responses[2].Result, &comp2); err != nil {
		t.Fatal(err)
	}
	if len(comp2.Items) != 2 {
		t.Errorf("after create: got %d items, want 2", len(comp2.Items))
	}

	// Verify the new note is in the list.
	found := false
	for _, item := range comp2.Items {
		if item.Label == "20220601-2-beta" {
			found = true
			break
		}
	}
	if !found {
		t.Error("new note 20220601-2-beta not found in completion after create")
	}

	// Response 3 (id=4): after delete - 1 note (beta).
	var comp3 lspCompletionList
	if err := json.Unmarshal(responses[3].Result, &comp3); err != nil {
		t.Fatal(err)
	}
	if len(comp3.Items) != 1 {
		t.Errorf("after delete: got %d items, want 1", len(comp3.Items))
	}

	// Verify alpha is gone.
	for _, item := range comp3.Items {
		if item.Label == "20220601-1-alpha" {
			t.Error("deleted note 20220601-1-alpha should not be in completion after delete")
		}
	}
}

// ---------- test helpers ----------

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeRPCMessage(t *testing.T, buf *bytes.Buffer, id int, method string, params any) {
	t.Helper()
	rawID := json.RawMessage(fmt.Sprintf("%d", id))
	msg := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      &rawID,
		Method:  method,
	}
	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		msg.Params = p
	}
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(buf, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

func writeRPCNotification(t *testing.T, buf *bytes.Buffer, method string, params any) {
	t.Helper()
	msg := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
	}
	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		msg.Params = p
	}
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(buf, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

type rawResponse struct {
	ID     *json.RawMessage `json:"id"`
	Result json.RawMessage  `json:"result"`
	Error  *jsonrpcError    `json:"error"`
}

func parseRPCResponses(t *testing.T, data []byte) []rawResponse {
	t.Helper()
	var responses []rawResponse

	for len(data) > 0 {
		// Find Content-Length header.
		idx := bytes.Index(data, []byte("Content-Length: "))
		if idx < 0 {
			break
		}
		data = data[idx+len("Content-Length: "):]
		endIdx := bytes.Index(data, []byte("\r\n\r\n"))
		if endIdx < 0 {
			break
		}
		var length int
		fmt.Sscanf(string(data[:endIdx]), "%d", &length)
		data = data[endIdx+4:]
		if len(data) < length {
			break
		}
		body := data[:length]
		data = data[length:]

		var resp rawResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("unmarshal response: %v\nbody: %s", err, body)
		}
		responses = append(responses, resp)
	}
	return responses
}
