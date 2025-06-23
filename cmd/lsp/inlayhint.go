package lsp

import "github.com/gluax-lang/lsp"

func (h *Handler) InlayHint(p *lsp.InlayHintParams) ([]lsp.InlayHint, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	uri := p.TextDocument.URI
	path, err := uriToFilePath(uri)
	if err != nil {
		return nil, nil
	}
	text := h.fileCache[path]
	if text == "" {
		return nil, nil
	}
	pAnalysis := h.lastProjAnalysis
	if pAnalysis == nil {
		return nil, nil
	}
	analysis := pAnalysis.Files()[path]
	if analysis == nil {
		return nil, nil
	}
	return analysis.InlayHints, nil
}
