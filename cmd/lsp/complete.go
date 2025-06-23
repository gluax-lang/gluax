package lsp

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/lsp"
)

func (h *Handler) Complete(p *lsp.CompletionParams) (*lsp.CompletionList, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	uri := p.TextDocument.URI
	fPath, err := uriToFilePath(uri)
	if err != nil {
		println("COMPLETION CANCELED")
		return nil, nil
	}

	pA := h.lastProjAnalysis
	if pA == nil {
		return nil, nil
	}
	sA := pA.ServerFiles()[fPath]
	if sA == nil {
		return nil, nil
	}

	scope := sA.FindScopeByPosition(p.Position, fPath)
	if scope == nil {
		return nil, nil
	}

	text := h.fileCache[fPath]
	if text == "" {
		return nil, nil
	}

	runeIndex := BuildRuneIndex(text)

	// Dot completion logic
	if r := runeIndex.GetRuneBefore(p.Position); r != nil {
		var toIndex *ast.Expr
		isCall := false
		for i := len(sA.Exprs) - 1; i >= 0; i-- {
			expr := sA.Exprs[i]
			if expr.Span().ToRange().Contains(r.Position) {
				if expr.Kind() == ast.ExprKindPostfix {
					toIndex = &expr.Postfix().Left
					_, isCall = expr.Postfix().Op.(*ast.Call)
				}
				break
			}
		}
		if toIndex == nil {
			goto outDotCompletion
		}

		toIndexTy := toIndex.Type()

		var list []lsp.CompletionItem
		if !isCall {
			if toIndexTy.IsClass() {
				clss := toIndexTy.Class()
				for _, field := range clss.Fields {
					list = append(list, lsp.CompletionItem{
						Label:  field.Def.Name.Raw,
						Kind:   lsp.CompletionItemKindField,
						Detail: field.LSPString(),
					})
				}
			}
		}
		methods := sA.FindMethodsOnType(scope, toIndexTy, "")

		added := make(map[string]struct{})
		for _, method := range methods {
			name := method.Def.Name.Raw
			if _, exists := added[name]; exists {
				continue
			}
			if !method.IsFirstParamSelf() {
				continue
			}
			added[name] = struct{}{} // Mark this method as added
			list = append(list, lsp.CompletionItem{
				Label:            method.Def.Name.Raw,
				Kind:             lsp.CompletionItemKindMethod,
				Detail:           method.LSPString(),
				InsertText:       method.Def.Name.Raw + "($0)", // Insert with parens and snippet cursor
				InsertTextFormat: lsp.InsertTextFormatSnippet,
			})
		}

		return &lsp.CompletionList{
			IsIncomplete: false,
			Items:        list,
		}, nil

	}
outDotCompletion:

	// println("SEARCHING SCOPE")

	var list []lsp.CompletionItem
	for _, symSlice := range scope.Symbols {
		for _, sym := range symSlice {
			if sym.IsImport() || sym.IsTrait() || sym.IsType() {
				continue // Skip imports, traits, and types
			}
			list = append(list, lsp.CompletionItem{
				Label:  sym.Name,
				Kind:   lsp.CompletionItemKindVariable,
				Detail: sym.LSPString(),
			})
		}
	}

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        list,
	}, nil
}
