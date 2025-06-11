package lsp

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/sema"
	protocol "github.com/gluax-lang/lsp"
)

func RunLSP() error {
	return NewHandler().Serve(context.Background())
}

type Handler struct {
	*protocol.Server
	fileCache        map[string]string
	mu               sync.Mutex
	workspace        string
	lastProjAnalysis *sema.ProjectAnalysis
}

func NewHandler() *Handler {
	h := &Handler{
		fileCache: make(map[string]string),
		mu:        sync.Mutex{},
	}
	h.Server = protocol.NewServer(os.Stdin, os.Stdout, h)
	return h
}

func (h *Handler) Initialize(p *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	if p.WorkspaceFolders == nil || len(*p.WorkspaceFolders) == 0 {
		return nil, fmt.Errorf("no workspace folder detected")
	}
	workspaceFolders := *p.WorkspaceFolders
	root, err := uriToFilePath(workspaceFolders[0].URI)
	if err != nil {
		fmt.Printf("invalid workspace folder: %v", err)
		return nil, err
	}
	log.Printf("root: %s", root)
	h.workspace = root
	return &protocol.InitializeResult{Capabilities: protocol.ServerCapabilities{
		HoverProvider: protocol.NewHoverProviderBool(true),
		TextDocumentSync: protocol.NewTextDocumentSyncOptions(protocol.TextDocumentSyncOptions{
			OpenClose: true,
			Change:    protocol.TextDocumentSyncKindFull,
			Save: &protocol.SaveOptions{
				IncludeText: true,
			},
		}),
		InlayHintProvider: protocol.NewInlayHintProviderOptions(protocol.InlayHintOptions{
			ResolveProvider: false,
			WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
				WorkDoneProgress: false,
			},
		}),
		DefinitionProvider: true,
	}}, nil
}

func (h *Handler) Initialized() error {
	log.Println("Initialized")
	return nil
}

func (h *Handler) Hover(p *protocol.HoverParams) (*protocol.Hover, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	uri := p.TextDocument.URI
	position := p.Position

	sym := h.findSymAtPos(uri, h.fileCache[uri], position)
	if sym == nil {
		return nil, nil
	}

	content := fmt.Sprintf("```gluax\n%s\n```\n", sym.AstString())

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  "markdown",
			Value: content,
		},
	}, nil
}

func (h *Handler) InlayHint(p *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	uri := p.TextDocument.URI
	text := h.fileCache[uri]
	if text == "" {
		return nil, nil
	}
	path, err := uriToFilePath(uri)
	if err != nil {
		return nil, nil
	}
	pAnalysis := h.lastProjAnalysis
	if pAnalysis == nil {
		return nil, nil
	}
	analysis := pAnalysis.Files()[pAnalysis.PathRelativeToWorkspace(path)]
	if analysis == nil {
		return nil, nil
	}
	return analysis.InlayHints, nil
}

func (h *Handler) DidOpen(p *protocol.DidOpenTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	uri := p.TextDocument.URI
	text := p.TextDocument.Text
	h.fileCache[uri] = text
	h.handleDiagnostics(uri, text)
	return nil
}

func (h *Handler) DidChange(p *protocol.DidChangeTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	uri := p.TextDocument.URI
	text := p.ContentChanges[0].Text
	h.fileCache[uri] = text
	h.handleDiagnostics(uri, text)
	return nil
}

func (h *Handler) DidClose(p *protocol.DidCloseTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	uri := p.TextDocument.URI
	delete(h.fileCache, uri)
	return nil
}

func (h *Handler) DidSave(p *protocol.DidSaveTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	uri := p.TextDocument.URI
	text := *p.Text
	h.fileCache[uri] = text
	h.handleDiagnostics(uri, text)
	return nil
}

func (h *Handler) compileProject(uri, code string) (*string, *sema.ProjectAnalysis) {
	relPath, err := uriToFilePath(uri)
	if err != nil {
		return nil, nil
	}
	overrides := map[string]string{
		relPath: code,
	}
	pAnalysis, err := sema.AnalyzeProject(h.workspace, overrides)
	if err != nil {
		fmt.Printf("error analyzing project: %v", err)
		return nil, nil
	}
	h.lastProjAnalysis = pAnalysis
	return &relPath, pAnalysis
}

// func (h *Handler) getFileAnalysis(uri, code string) *sema.Analysis {
// 	relPath, pAnalysis := h.compileProject(uri, code)
// 	if relPath == nil || pAnalysis == nil {
// 		return nil
// 	}
// 	analysis := pAnalysis.Files()[pAnalysis.PathRelativeToWorkspace(*relPath)]
// 	return analysis
// }

func (h *Handler) getServerFileAnalysis(uri, code string) *sema.Analysis {
	relPath, pAnalysis := h.compileProject(uri, code)
	if relPath == nil || pAnalysis == nil {
		return nil
	}
	analysis := pAnalysis.ServerFiles()[pAnalysis.PathRelativeToWorkspace(*relPath)]
	return analysis
}

func (h *Handler) getClientFileAnalysis(uri, code string) *sema.Analysis {
	relPath, pAnalysis := h.compileProject(uri, code)
	if relPath == nil || pAnalysis == nil {
		return nil
	}
	analysis := pAnalysis.ClientFiles()[pAnalysis.PathRelativeToWorkspace(*relPath)]
	return analysis
}

func (h *Handler) handleDiagnostics(uri, code string) {
	relPath, pAnalysis := h.compileProject(uri, code)
	if relPath == nil || pAnalysis == nil {
		return
	}
	for _, analysis := range pAnalysis.Files() {
		fileURI := common.FilePathToURI(analysis.Src)
		h.PublishDiagnostics(fileURI, analysis.Diags)
	}
}

// func (h *Handler) handleDiagnostics(uri, code string) {
// 	analysis := h.getFileAnalysis(uri, code)
// 	if analysis == nil {
// 		return
// 	}
// 	h.PublishDiagnostics(uri, analysis.Diags)
// }

func (h *Handler) Definition(p *protocol.DefinitionParams) ([]protocol.Location, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	uri := p.TextDocument.URI
	position := p.Position

	// Get the file analysis
	analysis := h.getServerFileAnalysis(uri, h.fileCache[uri])
	if analysis == nil {
		return nil, nil
	}

	// Find symbol at position using scopes
	symbol := h.findSymAtPos(uri, h.fileCache[uri], position)
	if symbol == nil {
		return nil, nil
	}
	// Convert symbol span to location
	return []protocol.Location{symbol.Span.ToLocation()}, nil
}

func (h *Handler) findSymAtPos(uri, code string, pos protocol.Position) *sema.Symbol {
	find := func(analysis *sema.Analysis) *sema.Symbol {
		for span, sym := range analysis.SpanSymbols {
			rng := span.ToRange()
			rng.End.Character++ // just to make sure it works if clicking on the last character
			if rng.Contains(pos) {
				return &sym
			}
		}
		return nil
	}
	if serverAnalysis := h.getServerFileAnalysis(uri, code); serverAnalysis != nil {
		if symbol := find(serverAnalysis); symbol != nil {
			return symbol
		}
	}
	if clientAnalysis := h.getClientFileAnalysis(uri, code); clientAnalysis != nil {
		if symbol := find(clientAnalysis); symbol != nil {
			return symbol
		}
	}
	return nil
}

// uriToFilePath converts a file:// URI into an **absolute** filesystem path.
func uriToFilePath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI: %w", err)
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("unsupported scheme %q (must be file)", u.Scheme)
	}

	// URL‐unescape the path (e.g. %20 -> space)
	p, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", fmt.Errorf("cannot unescape path: %w", err)
	}

	// On Windows, strip the leading slash before the drive letter
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
			p = p[1:]
		}
	}

	// Convert slashes to OS‐specific separators
	return filepath.FromSlash(p), nil
}
