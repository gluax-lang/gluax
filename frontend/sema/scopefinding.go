package sema

import (
	"github.com/gluax-lang/lsp"
)

func (a *Analysis) FindScopeByPosition(pos lsp.Position, fPath string) *Scope {
	return a.findMostSpecificScopeInHierarchy(a.Scope, pos)
}

func (a *Analysis) findMostSpecificScopeInHierarchy(scope *Scope, pos lsp.Position) *Scope {
	var mostSpecific *Scope

	if a.scopeContainsPosition(scope, pos) {
		mostSpecific = scope
	}

	for _, child := range scope.Children {
		childResult := a.findMostSpecificScopeInHierarchy(child, pos)
		if childResult != nil {
			mostSpecific = childResult
		}
	}

	return mostSpecific
}

func (a *Analysis) scopeContainsPosition(scope *Scope, pos lsp.Position) bool {
	if scope.Span == nil {
		return false
	}
	scopeSpan := scope.Span.ToRange()
	return scopeSpan.Contains(pos)
}
