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
	"github.com/gluax-lang/lsp"
)

func RunLSP() error {
	return NewHandler().Serve(context.Background())
}

type FileAnalysis struct {
	server *sema.Analysis // Analysis for server files
	client *sema.Analysis // Analysis for client files
}

type Handler struct {
	*lsp.Server
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
	h.Server = lsp.NewServer(os.Stdin, os.Stdout, h)
	return h
}

func (h *Handler) Initialize(p *lsp.InitializeParams) (*lsp.InitializeResult, error) {
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
	return &lsp.InitializeResult{Capabilities: lsp.ServerCapabilities{
		HoverProvider: lsp.NewHoverProviderBool(true),
		TextDocumentSync: lsp.NewTextDocumentSyncOptions(lsp.TextDocumentSyncOptions{
			OpenClose: true,
			Change:    lsp.TextDocumentSyncKindFull,
			Save: &lsp.SaveOptions{
				IncludeText: true,
			},
		}),
		InlayHintProvider: lsp.NewInlayHintProviderOptions(lsp.InlayHintOptions{
			ResolveProvider: false,
			WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{
				WorkDoneProgress: false,
			},
		}),
		DefinitionProvider: true,
		ReferencesProvider: true,
		CompletionProvider: lsp.CompletionOptions{
			TriggerCharacters: []string{"."},
		},
	}}, nil
}

func (h *Handler) Initialized() error {
	log.Println("Initialized")
	return nil
}

func (h *Handler) compileProject() *sema.ProjectAnalysis {
	overrides := h.fileCache
	options := sema.CompileOptions{
		Workspace:    h.workspace,
		VirtualFiles: overrides,
	}
	pAnalysis, err := sema.AnalyzeProject(options)
	if err != nil {
		log.Printf("error analyzing project: %v", err)
		return nil
	}
	h.lastProjAnalysis = pAnalysis
	return pAnalysis
}

func (h *Handler) getStatesAnalysis(uri string) (*sema.Analysis, *sema.Analysis) {
	pAnalysis := h.compileProject()
	if pAnalysis == nil {
		return nil, nil
	}
	relPath, err := uriToFilePath(uri)
	if err != nil {
		return nil, nil
	}
	serverAnalysis := pAnalysis.ServerFiles()[relPath]
	clientAnalysis := pAnalysis.ClientFiles()[relPath]
	return serverAnalysis, clientAnalysis
}

func (h *Handler) handleDiagnostics() {
	pAnalysis := h.compileProject()
	if pAnalysis == nil {
		return
	}
	for _, analysis := range pAnalysis.Files() {
		fileURI := common.FilePathToURI(analysis.Src)
		h.PublishDiagnostics(fileURI, analysis.Diags)
	}
}

func (h *Handler) findSymAtPos(uri string, pos lsp.Position, pA *sema.ProjectAnalysis) *sema.LSPSymbol {
	if pA == nil {
		pA = h.compileProject()
		if pA == nil {
			return nil
		}
	}
	fPath, err := uriToFilePath(uri)
	if err != nil {
		return nil
	}
	fPath = common.FilePathClean(fPath)
	if serverAnalysis := pA.ServerFiles()[fPath]; serverAnalysis != nil {
		if symbol := serverAnalysis.GetSymbolAtPosition(pos, fPath); symbol != nil {
			return symbol
		}
	}
	if clientAnalysis := pA.ClientFiles()[fPath]; clientAnalysis != nil {
		if symbol := clientAnalysis.GetSymbolAtPosition(pos, fPath); symbol != nil {
			return symbol
		}
	}
	return nil
}

func (h *Handler) findDeclAtPos(uri string, pos lsp.Position) *sema.DeclWithRef {
	fPath, err := uriToFilePath(uri)
	if err != nil {
		return nil
	}
	fPath = common.FilePathClean(fPath)
	serverAnalysis, clientAnalysis := h.getStatesAnalysis(uri)
	if serverAnalysis != nil {
		if symbol := serverAnalysis.GetDeclAtPosition(pos, fPath); symbol != nil {
			return symbol
		}
	}
	if clientAnalysis != nil {
		if symbol := clientAnalysis.GetDeclAtPosition(pos, fPath); symbol != nil {
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
	return common.FilePathClean(filepath.FromSlash(p)), nil
}
