package gonotes

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// lspLog writes to stderr, which Neovim captures in its LSP log.
var lspLog = log.New(os.Stderr, "[gonotes-lsp] ", log.Ltime|log.Lshortfile)

// ---------- JSON-RPC 2.0 types ----------

type jsonrpcRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  any              `json:"result"`
	Error   *jsonrpcError    `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ---------- LSP protocol types ----------

type lspPosition struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspLocation struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type lspTextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type lspTextDocumentPositionParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
	Position     lspPosition               `json:"position"`
}

type lspReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type lspReferenceParams struct {
	lspTextDocumentPositionParams
	Context lspReferenceContext `json:"context"`
}

type lspWorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type lspInitializeParams struct {
	RootURI          string               `json:"rootUri"`
	RootPath         string               `json:"rootPath"`
	WorkspaceFolders []lspWorkspaceFolder `json:"workspaceFolders"`
}

type lspServerCapabilities struct {
	TextDocumentSync   *lspTextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	CompletionProvider *lspCompletionOptions       `json:"completionProvider,omitempty"`
	DefinitionProvider bool                        `json:"definitionProvider"`
	ReferencesProvider bool                        `json:"referencesProvider"`
}

type lspInitializeResult struct {
	Capabilities lspServerCapabilities `json:"capabilities"`
}

// ---------- Document sync types ----------

type lspTextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose"`
	Change    int  `json:"change"` // 1 = Full
}

type lspTextDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

type lspDidOpenTextDocumentParams struct {
	TextDocument lspTextDocumentItem `json:"textDocument"`
}

type lspDidChangeTextDocumentParams struct {
	TextDocument   lspVersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []lspTextDocumentContentChangeEvent `json:"contentChanges"`
}

type lspVersionedTextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type lspTextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type lspDidCloseTextDocumentParams struct {
	TextDocument lspTextDocumentIdentifier `json:"textDocument"`
}

// ---------- Completion types ----------

type lspCompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type lspCompletionParams struct {
	lspTextDocumentPositionParams
	Context *lspCompletionContext `json:"context,omitempty"`
}

type lspCompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

type lspCompletionItemLabelDetails struct {
	Description string `json:"description,omitempty"`
}

type lspCompletionItem struct {
	Label        string                         `json:"label"`
	Kind         int                            `json:"kind,omitempty"`
	Detail       string                         `json:"detail,omitempty"`
	LabelDetails *lspCompletionItemLabelDetails `json:"labelDetails,omitempty"`
	InsertText   string                         `json:"insertText,omitempty"`
	FilterText   string                         `json:"filterText,omitempty"`
}

type lspCompletionList struct {
	IsIncomplete bool                `json:"isIncomplete"`
	Items        []lspCompletionItem `json:"items"`
}

// ---------- File watching types ----------

type lspDidChangeWatchedFilesParams struct {
	Changes []lspFileEvent `json:"changes"`
}

type lspFileEvent struct {
	URI  string `json:"uri"`
	Type int    `json:"type"` // 1=Created, 2=Changed, 3=Deleted
}

// ---------- LSP server ----------

// LSPOptions configures the LSP server.
type LSPOptions struct {
	// FlatTags uses notes/by/tags/flat/ for tag references, resolving the
	// individual tag segment under the cursor. When false (default), uses
	// notes/by/tags/nested/ with the full tag path.
	FlatTags bool
}

// noteEntry holds a note's filename stem and tags for completion.
type noteEntry struct {
	stem string
	tags string
}

type lspServer struct {
	rootDir  string
	flatTags bool
	docs     map[string]string // URI -> content for open documents
	notes    []noteEntry       // cached notes from notes/by/id/
	files    []string          // cached file paths from files/
}

func (s *lspServer) handle(req jsonrpcRequest) (any, error) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params)
	case "initialized":
		return nil, nil
	case "textDocument/didOpen":
		return s.handleDidOpen(req.Params)
	case "textDocument/didChange":
		return s.handleDidChange(req.Params)
	case "textDocument/didClose":
		return s.handleDidClose(req.Params)
	case "textDocument/didSave":
		return nil, nil
	case "textDocument/completion":
		return s.handleCompletion(req.Params)
	case "textDocument/definition":
		return s.handleDefinition(req.Params)
	case "textDocument/references":
		return s.handleReferences(req.Params)
	case "workspace/didChangeWatchedFiles":
		return s.handleDidChangeWatchedFiles(req.Params)
	case "shutdown":
		return nil, nil
	case "exit":
		os.Exit(0)
	}
	return nil, nil
}

func (s *lspServer) handleInitialize(raw json.RawMessage) (any, error) {
	var params lspInitializeParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal initialize params: %w", err)
	}

	// Determine root directory: try rootUri, then workspaceFolders, then rootPath.
	switch {
	case params.RootURI != "":
		s.rootDir = uriToPath(params.RootURI)
	case len(params.WorkspaceFolders) > 0 && params.WorkspaceFolders[0].URI != "":
		s.rootDir = uriToPath(params.WorkspaceFolders[0].URI)
	case params.RootPath != "":
		s.rootDir = params.RootPath
	}

	// The editor may point into a subdirectory (e.g. notes/by/tags/flat).
	// Walk up to find the actual root that contains notes/by/id/.
	root, ok := findNotesRoot(s.rootDir)
	if !ok {
		lspLog.Printf("initialize: no notes/by/id/ found above %q, not a gonotes workspace", s.rootDir)
		return nil, fmt.Errorf("not a gonotes workspace: no notes/by/id/ directory found above %q", s.rootDir)
	}
	s.rootDir = root

	s.docs = make(map[string]string)
	s.refreshIndex()

	lspLog.Printf("initialize: rootDir=%q (rootUri=%q rootPath=%q workspaceFolders=%d) notes=%d files=%d",
		s.rootDir, params.RootURI, params.RootPath, len(params.WorkspaceFolders),
		len(s.notes), len(s.files))

	return lspInitializeResult{
		Capabilities: lspServerCapabilities{
			TextDocumentSync: &lspTextDocumentSyncOptions{
				OpenClose: true,
				Change:    1, // Full
			},
			CompletionProvider: &lspCompletionOptions{
				TriggerCharacters: []string{"["},
			},
			DefinitionProvider: true,
			ReferencesProvider: true,
		},
	}, nil
}

func (s *lspServer) handleDefinition(raw json.RawMessage) (any, error) {
	var params lspTextDocumentPositionParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal definition params: %w", err)
	}

	content, err := s.getDocumentContent(params.TextDocument.URI)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	target := wikiLinkAtPosition(content, int(params.Position.Line), int(params.Position.Character))
	lspLog.Printf("definition: uri=%q line=%d col=%d target=%q",
		params.TextDocument.URI, params.Position.Line, params.Position.Character, target)
	if target == "" {
		return nil, nil
	}

	targetID := linkTargetToID(target)
	idDir := filepath.Join(s.rootDir, "notes", "by", "id")

	filename, err := findFileForID(idDir, targetID)
	lspLog.Printf("definition: targetID=%q idDir=%q filename=%q err=%v",
		targetID, idDir, filename, err)
	if err != nil || filename == "" {
		return nil, err
	}

	return lspLocation{
		URI:   pathToURI(filepath.Join(idDir, filename)),
		Range: lspRange{},
	}, nil
}

func (s *lspServer) handleReferences(raw json.RawMessage) (any, error) {
	var params lspReferenceParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal reference params: %w", err)
	}

	content, err := s.getDocumentContent(params.TextDocument.URI)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	path := uriToPath(params.TextDocument.URI)
	lines := strings.Split(content, "\n")
	line := int(params.Position.Line)
	col := int(params.Position.Character)

	lspLog.Printf("references: path=%q line=%d col=%d", path, line, col)

	// Mode 1: cursor on a wiki-link.
	if target := wikiLinkAtPosition(content, line, col); target != "" {
		targetID := linkTargetToID(target)
		idDir := filepath.Join(s.rootDir, "notes", "by", "id")
		lspLog.Printf("references: mode=wiki-link target=%q targetID=%q idDir=%q", target, targetID, idDir)
		locs, err := findReferencesToID(idDir, targetID)
		lspLog.Printf("references: found %d locations, err=%v", len(locs), err)
		return locs, err
	}

	// Mode 2 & 3: cursor in frontmatter.
	inFM := line < len(lines) && isInFrontmatter(content, line)
	lspLog.Printf("references: inFrontmatter=%v", inFM)
	if inFM {
		lineText := lines[line]

		// Mode 2: cursor on a tag.
		if s.flatTags {
			if seg := tagSegmentAtPosition(lineText, col); seg != "" {
				lspLog.Printf("references: mode=tag-flat segment=%q", seg)
				return s.findNotesByTagFlat(seg)
			}
		} else {
			if tag := tagAtPosition(lineText, col); tag != "" {
				lspLog.Printf("references: mode=tag-nested tag=%q", tag)
				return s.findNotesByTag(tag)
			}
		}

		// Mode 3: cursor on title line — find references to this file.
		if key := frontmatterKeyAtLine(lineText); key == "title" {
			id, _ := IDFromFilename(filepath.Base(path))
			lspLog.Printf("references: mode=title id=%q", id)
			if id != "" {
				idDir := filepath.Join(s.rootDir, "notes", "by", "id")
				return findReferencesToID(idDir, id)
			}
		}
	}

	lspLog.Printf("references: no mode matched")
	return nil, nil
}

func (s *lspServer) findNotesByTag(tag string) ([]lspLocation, error) {
	tagDir := filepath.Join(s.rootDir, "notes", "by", "tags", "nested", tag)

	entries, err := os.ReadDir(tagDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tag dir: %w", err)
	}

	var locs []lspLocation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// Resolve the symlink to get the real path.
		linkPath := filepath.Join(tagDir, e.Name())
		realPath, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			continue
		}
		locs = append(locs, lspLocation{
			URI:   pathToURI(realPath),
			Range: lspRange{},
		})
	}
	return locs, nil
}

func (s *lspServer) findNotesByTagFlat(segment string) ([]lspLocation, error) {
	tagDir := filepath.Join(s.rootDir, "notes", "by", "tags", "flat", segment)

	entries, err := os.ReadDir(tagDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tag dir: %w", err)
	}

	var locs []lspLocation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		linkPath := filepath.Join(tagDir, e.Name())
		realPath, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			continue
		}
		locs = append(locs, lspLocation{
			URI:   pathToURI(realPath),
			Range: lspRange{},
		})
	}
	return locs, nil
}

// ---------- Document store ----------

func (s *lspServer) handleDidOpen(raw json.RawMessage) (any, error) {
	var params lspDidOpenTextDocumentParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal didOpen params: %w", err)
	}
	s.docs[params.TextDocument.URI] = params.TextDocument.Text
	return nil, nil
}

func (s *lspServer) handleDidChange(raw json.RawMessage) (any, error) {
	var params lspDidChangeTextDocumentParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal didChange params: %w", err)
	}
	if len(params.ContentChanges) > 0 {
		// Full sync: last change contains the entire document.
		s.docs[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
	}
	return nil, nil
}

func (s *lspServer) handleDidClose(raw json.RawMessage) (any, error) {
	var params lspDidCloseTextDocumentParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal didClose params: %w", err)
	}
	delete(s.docs, params.TextDocument.URI)
	return nil, nil
}

// getDocumentContent returns the content of a document. If the document is
// open in the editor (tracked via didOpen/didChange), the in-memory version
// is returned. Otherwise the file is read from disk.
func (s *lspServer) getDocumentContent(uri string) (string, error) {
	if content, ok := s.docs[uri]; ok {
		return content, nil
	}
	data, err := os.ReadFile(uriToPath(uri))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ---------- Completion index ----------

// refreshIndex scans notes/by/id/ for .md files (extracting stems and tags)
// and walks files/ for all file paths. Called once during initialize.
func (s *lspServer) refreshIndex() {
	s.notes = nil
	s.files = nil

	idDir := filepath.Join(s.rootDir, "notes", "by", "id")
	if entries, err := os.ReadDir(idDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			stem := strings.TrimSuffix(e.Name(), ".md")
			var tags string
			if data, err := os.ReadFile(filepath.Join(idDir, e.Name())); err == nil {
				tags = extractFrontmatterTags(string(data))
			}
			s.notes = append(s.notes, noteEntry{stem: stem, tags: tags})
		}
	}

	filesDir := filepath.Join(s.rootDir, "files")
	filepath.WalkDir(filesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filesDir, path)
		if err != nil {
			return nil
		}
		s.files = append(s.files, rel)
		return nil
	})
}

func (s *lspServer) indexAddFile(uri string) {
	path := uriToPath(uri)
	idDir := filepath.Join(s.rootDir, "notes", "by", "id")
	filesDir := filepath.Join(s.rootDir, "files")

	if strings.HasPrefix(path, idDir+string(filepath.Separator)) {
		name := filepath.Base(path)
		if strings.HasSuffix(name, ".md") {
			stem := strings.TrimSuffix(name, ".md")
			var tags string
			if data, err := os.ReadFile(path); err == nil {
				tags = extractFrontmatterTags(string(data))
			}
			s.notes = append(s.notes, noteEntry{stem: stem, tags: tags})
		}
	} else if strings.HasPrefix(path, filesDir+string(filepath.Separator)) {
		rel, err := filepath.Rel(filesDir, path)
		if err == nil {
			s.files = append(s.files, rel)
		}
	}
}

func (s *lspServer) indexRemoveFile(uri string) {
	path := uriToPath(uri)
	idDir := filepath.Join(s.rootDir, "notes", "by", "id")
	filesDir := filepath.Join(s.rootDir, "files")

	if strings.HasPrefix(path, idDir+string(filepath.Separator)) {
		name := filepath.Base(path)
		if strings.HasSuffix(name, ".md") {
			stem := strings.TrimSuffix(name, ".md")
			for i, n := range s.notes {
				if n.stem == stem {
					s.notes = append(s.notes[:i], s.notes[i+1:]...)
					break
				}
			}
		}
	} else if strings.HasPrefix(path, filesDir+string(filepath.Separator)) {
		rel, err := filepath.Rel(filesDir, path)
		if err == nil {
			for i, f := range s.files {
				if f == rel {
					s.files = append(s.files[:i], s.files[i+1:]...)
					break
				}
			}
		}
	}
}

// ---------- Completion ----------

func (s *lspServer) handleCompletion(raw json.RawMessage) (any, error) {
	var params lspCompletionParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal completion params: %w", err)
	}

	content, err := s.getDocumentContent(params.TextDocument.URI)
	if err != nil {
		return nil, nil // gracefully return no completions
	}

	if !isInsideWikiLink(content, int(params.Position.Line), int(params.Position.Character)) {
		return nil, nil
	}

	items := make([]lspCompletionItem, 0, len(s.notes)+len(s.files))
	for _, n := range s.notes {
		item := lspCompletionItem{
			Label:      n.stem,
			Kind:       17, // File
			Detail:     n.tags,
			InsertText: n.stem,
			FilterText: n.stem,
		}
		if n.tags != "" {
			item.LabelDetails = &lspCompletionItemLabelDetails{Description: n.tags}
		}
		items = append(items, item)
	}
	for _, name := range s.files {
		items = append(items, lspCompletionItem{
			Label:      name,
			Kind:       17, // File
			InsertText: name,
			FilterText: name,
		})
	}

	return lspCompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// ---------- File watching ----------

func (s *lspServer) handleDidChangeWatchedFiles(raw json.RawMessage) (any, error) {
	var params lspDidChangeWatchedFilesParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal didChangeWatchedFiles params: %w", err)
	}
	for _, change := range params.Changes {
		switch change.Type {
		case 1: // Created
			s.indexAddFile(change.URI)
		case 3: // Deleted
			s.indexRemoveFile(change.URI)
		}
	}
	return nil, nil
}

// ---------- Helpers ----------

// wikiLinkAtPosition returns the target inside a [[...]] wiki-link if the
// cursor at (line, col) falls within one, or "" otherwise.
func wikiLinkAtPosition(content string, line, col int) string {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	lineText := lines[line]

	for _, m := range reWikiLink.FindAllStringSubmatchIndex(lineText, -1) {
		// m[0]:m[1] is the full match [[target]]
		// m[2]:m[3] is the captured group (target)
		linkStart := m[0] // position of first [
		linkEnd := m[1]   // position after last ]
		if col >= linkStart && col < linkEnd {
			return lineText[m[2]:m[3]]
		}
	}
	return ""
}

// isInsideWikiLink reports whether the cursor at (line, col) is inside an
// unclosed [[ wiki-link. It scans backwards from the cursor on the same line
// looking for [[ without a closing ]] between it and the cursor.
func isInsideWikiLink(content string, line, col int) bool {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return false
	}
	lineText := lines[line]
	if col > len(lineText) {
		col = len(lineText)
	}
	prefix := lineText[:col]

	// Find the last [[ in the prefix.
	openIdx := strings.LastIndex(prefix, "[[")
	if openIdx < 0 {
		return false
	}
	// Check there is no ]] between the [[ and the cursor.
	between := prefix[openIdx+2:]
	return !strings.Contains(between, "]]")
}

// extractFrontmatterTags does a lightweight scan of a note's content to
// extract the tags value from the YAML frontmatter without full parsing.
// Returns the raw value string (e.g. "programming/go, tools") or "".
func extractFrontmatterTags(content string) string {
	lines := strings.Split(content, "\n")
	inFM := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == frontmatterSep {
			if inFM {
				return "" // closing --- reached without finding tags
			}
			inFM = true
			continue
		}
		if !inFM {
			continue
		}
		if key := frontmatterKeyAtLine(l); key == "tags" {
			idx := strings.Index(l, ":")
			return strings.TrimSpace(l[idx+1:])
		}
	}
	return ""
}

// tagAtPosition returns the tag under the cursor on a "tags: ..." frontmatter
// line, or "" if the cursor is not on a tag value.
func tagAtPosition(lineText string, col int) string {
	key := frontmatterKeyAtLine(lineText)
	if key != "tags" {
		return ""
	}
	// Find the start of the value (after "tags:").
	idx := strings.Index(lineText, ":")
	if idx < 0 {
		return ""
	}
	valueStart := idx + 1
	value := lineText[valueStart:]

	parts := strings.Split(value, ",")
	offset := valueStart
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			offset += len(part) + 1 // +1 for comma
			continue
		}
		// Find the tag within the part (accounting for leading whitespace).
		tagOffset := offset + strings.Index(part, tag)
		tagEnd := tagOffset + len(tag)
		if col >= tagOffset && col < tagEnd {
			return tag
		}
		offset += len(part) + 1 // +1 for comma
	}
	return ""
}

// tagSegmentAtPosition returns the slash-delimited segment of a tag under the
// cursor on a "tags: ..." frontmatter line. For example, with line
// "tags: programming/go, tools" and cursor on "go", it returns "go".
// Returns "" if the cursor is not on a tag value.
func tagSegmentAtPosition(lineText string, col int) string {
	tag := tagAtPosition(lineText, col)
	if tag == "" {
		return ""
	}
	if !strings.Contains(tag, "/") {
		return tag
	}

	// Find where this tag starts in the line so we can map col to
	// a position within the tag string.
	key := frontmatterKeyAtLine(lineText)
	if key != "tags" {
		return ""
	}
	idx := strings.Index(lineText, ":")
	if idx < 0 {
		return ""
	}
	valueStart := idx + 1
	value := lineText[valueStart:]

	// Walk comma-separated parts to find the offset of this tag.
	parts := strings.Split(value, ",")
	offset := valueStart
	tagOffset := -1
	for _, part := range parts {
		t := strings.TrimSpace(part)
		if t == tag {
			tagOffset = offset + strings.Index(part, t)
			break
		}
		offset += len(part) + 1
	}
	if tagOffset < 0 {
		return tag
	}

	// col is now relative to the tag start.
	posInTag := col - tagOffset
	if posInTag < 0 || posInTag >= len(tag) {
		return tag
	}

	// Find which segment the cursor falls in.
	segments := strings.Split(tag, "/")
	segStart := 0
	for _, seg := range segments {
		segEnd := segStart + len(seg)
		if posInTag >= segStart && posInTag < segEnd {
			return seg
		}
		segStart = segEnd + 1 // +1 for the "/"
	}
	return tag
}

// linkTargetToID normalizes a wiki-link target to a note ID by stripping
// the slug suffix. For targets with path separators (file references),
// the target is returned as-is.
func linkTargetToID(target string) string {
	if strings.Contains(target, "/") {
		return target
	}
	if m := reIDPrefix.FindStringSubmatch(target); m != nil {
		return m[1] + "-" + m[2]
	}
	return target
}

// findFileForID scans idDir for a markdown file whose ID matches the given id.
func findFileForID(idDir, id string) (string, error) {
	entries, err := os.ReadDir(idDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read id dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fileID, parsed := IDFromFilename(e.Name())
		if parsed && fileID == id {
			return e.Name(), nil
		}
	}
	return "", nil
}

// findReferencesToID scans all notes in idDir and returns the locations of
// wiki-links whose normalized target matches targetID.
func findReferencesToID(idDir, targetID string) ([]lspLocation, error) {
	entries, err := os.ReadDir(idDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read id dir: %w", err)
	}

	var locs []lspLocation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(idDir, e.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, lineText := range lines {
			for _, m := range reWikiLink.FindAllStringSubmatchIndex(lineText, -1) {
				linkTarget := lineText[m[2]:m[3]]
				if linkTargetToID(linkTarget) == targetID {
					locs = append(locs, lspLocation{
						URI: pathToURI(path),
						Range: lspRange{
							Start: lspPosition{Line: uint32(lineNum), Character: uint32(m[0])},
							End:   lspPosition{Line: uint32(lineNum), Character: uint32(m[1])},
						},
					})
				}
			}
		}
	}
	return locs, nil
}

// isInFrontmatter reports whether the given 0-indexed line number falls
// between the opening and closing --- frontmatter delimiters.
func isInFrontmatter(content string, line int) bool {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return false
	}

	sepCount := 0
	for i, l := range lines {
		if strings.TrimSpace(l) == frontmatterSep {
			sepCount++
			if sepCount == 2 {
				// Line is inside frontmatter if it's after the first ---
				// and before the second ---.
				return line > 0 && line < i
			}
		}
	}
	return false
}

// frontmatterKeyAtLine returns the frontmatter key name if the line looks
// like "key: value", or "" otherwise.
func frontmatterKeyAtLine(line string) string {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return ""
	}
	key := strings.TrimSpace(line[:idx])
	if key == "" || strings.ContainsAny(key, " \t") {
		return ""
	}
	return key
}

// findNotesRoot walks up from dir looking for a directory that contains
// notes/by/id/. If found, returns that ancestor. Otherwise returns dir as-is.
// findNotesRoot walks up from dir looking for a directory that contains
// notes/by/id/. Returns the path and true if found, or "" and false otherwise.
func findNotesRoot(dir string) (string, bool) {
	cur := dir
	for {
		candidate := filepath.Join(cur, "notes", "by", "id")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return cur, true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false
		}
		cur = parent
	}
}

// ---------- URI conversion ----------

func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		// Best effort: strip the scheme prefix.
		return strings.TrimPrefix(uri, "file://")
	}
	return u.Path
}

func pathToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	return "file://" + absPath
}

// ---------- Transport ----------

// ServeLSP runs the LSP server, reading JSON-RPC messages from r and writing
// responses to w. It blocks until r is closed or an error occurs.
func ServeLSP(r io.Reader, w io.Writer, opts LSPOptions) error {
	srv := &lspServer{flatTags: opts.FlatTags}
	reader := bufio.NewReader(r)

	for {
		// Read headers until blank line.
		var contentLen int
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("read header: %w", err)
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				n, err := strconv.Atoi(strings.TrimSpace(line[len("Content-Length:"):]))
				if err == nil {
					contentLen = n
				}
			}
		}

		if contentLen == 0 {
			continue
		}

		body := make([]byte, contentLen)
		if _, err := io.ReadFull(reader, body); err != nil {
			return fmt.Errorf("read body: %w", err)
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}

		result, err := srv.handle(req)

		// Notifications (no ID) get no response.
		if req.ID == nil {
			continue
		}

		resp := jsonrpcResponse{JSONRPC: "2.0", ID: req.ID}
		if err != nil {
			resp.Error = &jsonrpcError{Code: -32603, Message: err.Error()}
		} else {
			resp.Result = result
		}

		respBody, err := json.Marshal(resp)
		if err != nil {
			return fmt.Errorf("marshal response: %w", err)
		}
		if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(respBody), respBody); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}
}
