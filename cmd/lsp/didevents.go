package lsp

import "github.com/gluax-lang/lsp"

func (h *Handler) DidOpen(p *lsp.DidOpenTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	path, err := uriToFilePath(p.TextDocument.URI)
	if err != nil {
		return nil
	}
	text := p.TextDocument.Text
	h.fileCache[path] = text
	h.handleDiagnostics()
	return nil
}

func (h *Handler) DidChange(p *lsp.DidChangeTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	path, err := uriToFilePath(p.TextDocument.URI)
	if err != nil {
		return nil
	}
	text := p.ContentChanges[0].Text
	h.fileCache[path] = text
	h.handleDiagnostics()
	return nil
}

func (h *Handler) DidClose(p *lsp.DidCloseTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	uri := p.TextDocument.URI
	path, err := uriToFilePath(uri)
	if err != nil {
		return nil
	}
	delete(h.fileCache, path)
	return nil
}

func (h *Handler) DidSave(p *lsp.DidSaveTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	path, err := uriToFilePath(p.TextDocument.URI)
	if err != nil {
		return nil
	}
	text := *p.Text
	h.fileCache[path] = text
	h.handleDiagnostics()
	return nil
}
