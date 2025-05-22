package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleItem(scope *Scope, item ast.Item) {
	switch it := item.(type) {
	case *ast.Let:
		a.handleLet(scope, it)
	case *ast.Struct:
		// handled in Analysis.handleAst
	case *ast.Import:
		// handled in Analysis.handleAst
	case *ast.Use:
		a.handleUse(scope, it)
	case *ast.Function:
		funcSem := a.handleFunction(scope, it)
		it.SetSem(&funcSem)
		a.AddValue(scope, it.Name.Raw, ast.NewValue(funcSem), it.Name.Span())
	default:
		panic("TODO ITEM")
	}
}

func (a *Analysis) handleUse(scope *Scope, it *ast.Use) {
	sym := a.resolvePathSymbol(scope, &it.Path)
	var asName string
	if it.As != nil {
		asName = it.As.Raw
	} else {
		asName = it.Path.Idents[len(it.Path.Idents)-1].Raw
	}

	// var pathSegments []string
	// for _, id := range it.Path.Idents {
	// 	pathSegments = append(pathSegments, id.Raw)
	// }
	// a.UseAliases[asName] = pathSegments

	sym.SetPublic(it.Public)
	sym.SetIsUse(true)

	if err := scope.AddSymbol(asName, sym, it.Span()); err != nil {
		a.Error(err.Error(), it.Span())
	}
}
