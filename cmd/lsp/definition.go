package lsp

import "github.com/gluax-lang/lsp"

func (h *Handler) Definition(p *lsp.DefinitionParams) ([]lsp.Location, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	uri := p.TextDocument.URI
	position := p.Position

	// Find symbol at position using scopes
	symbol := h.findSymAtPos(uri, position, nil)
	if symbol == nil {
		return nil, nil
	}
	// Convert symbol span to location
	return []lsp.Location{(*symbol).Span().ToLocation()}, nil
}
