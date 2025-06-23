package lsp

import (
	"fmt"

	"github.com/gluax-lang/lsp"
)

func (h *Handler) Hover(p *lsp.HoverParams) (*lsp.Hover, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	uri := p.TextDocument.URI
	position := p.Position

	sym := h.findSymAtPos(uri, position, nil)
	if sym == nil {
		return nil, nil
	}

	content := fmt.Sprintf("```gluax\n%s\n```\n", (*sym).LSPString())

	return &lsp.Hover{
		Contents: lsp.MarkupContent{
			Kind:  "markdown",
			Value: content,
		},
	}, nil
}
