package lsp

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/lsp"
)

func (h *Handler) References(p *lsp.ReferenceParams) ([]lsp.Location, error) {
	var locations []lsp.Location

	dWR := h.findDeclAtPos(p.TextDocument.URI, p.Position)
	if dWR == nil {
		return nil, nil
	}

	if p.Context.IncludeDeclaration {
		locations = append(locations, dWR.Decl.Span().ToLocation())
	}

	for _, ref := range dWR.Refs {
		span := ref.Span()
		if ref, ok := ref.(ast.LSPRef); ok {
			span = ref.RefSpan()
		}

		locations = append(locations, span.ToLocation())
	}

	return locations, nil
}
