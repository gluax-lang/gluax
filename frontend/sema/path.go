package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
)

func getImportAnalysis(imp *ast.SemImport) *Analysis {
	return imp.Analysis.(*Analysis)
}

func getImportScope(imp *ast.SemImport) *Scope {
	if imp.Scope != nil {
		return imp.Scope.(*Scope)
	}
	return getImportAnalysis(imp).Scope
}

func resolvePathGeneric[T any](a *Analysis, scope *Scope, path *ast.Path, leafResolver func(*Symbol, ast.Ident) *T) *T {
	idents := path.Idents
	if len(idents) == 0 {
		return nil
	}

	var fakeImport ast.Import
	fakeSemImport := ast.NewSemImport(fakeImport, "", a)
	fakeSemImport.Scope = scope
	fakeSymbol := ast.NewSymbol("", &fakeSemImport, common.Span{}, false)
	currentSym := &fakeSymbol

	for i, ident := range idents[:len(idents)-1] {
		if currentSym.IsImport() {
			imp := currentSym.Import()
			currentSym = getImportScope(imp).GetSymbol(ident.Raw)
			if currentSym == nil {
				return nil
			}
			if i > 0 && !currentSym.IsPublic() {
				a.Panic(fmt.Sprintf("`%s` is private", ident.Raw), ident.Span())
			}
			if currentSym.IsImport() {
				imp := currentSym.Import()
				customSym := *currentSym
				customSym.Span = common.SpanSrc(getImportAnalysis(imp).Src)
				a.AddSpanSymbol(ident.Span(), customSym)
			} else {
				a.AddSpanSymbol(ident.Span(), *currentSym)
			}
		} else {
			return nil
		}
	}

	if currentSym == nil {
		return nil
	}

	leaf := idents[len(idents)-1]

	return leafResolver(currentSym, leaf)
}

func (a *Analysis) resolvePathType(scope *Scope, path *ast.Path) Type {
	t := resolvePathGeneric(a, scope, path, func(sym *Symbol, name ast.Ident) *Type {
		if !sym.IsImport() {
			return nil
		}
		sym = getImportScope(sym.Import()).GetSymbol(name.Raw)
		if sym == nil || !sym.IsType() {
			return nil
		}
		if len(path.Idents) > 1 && !sym.IsPublic() {
			a.Panic(fmt.Sprintf("`%s` is private", name.Raw), name.Span())
		}
		path.ResolvedSymbol = sym
		a.AddSpanSymbol(name.Span(), *sym)
		return sym.Type()
	})
	if t == nil {
		a.Panic(fmt.Sprintf("Type `%s` not found", path.String()), path.Span())
	}
	return *t
}

func (a *Analysis) resolvePathValue(scope *Scope, path *ast.Path) Value {
	t := resolvePathGeneric(a, scope, path, func(sym *Symbol, name ast.Ident) *Value {
		raw := name.Raw
		if sym.IsImport() {
			sym = getImportScope(sym.Import()).GetSymbol(raw)
			if sym == nil || !sym.IsValue() {
				return nil
			}
			if len(path.Idents) > 1 && !sym.IsPublic() {
				a.Panic(fmt.Sprintf("`%s` is private", raw), name.Span())
			}
			path.ResolvedSymbol = sym
			a.AddSpanSymbol(name.Span(), *sym)
			return sym.Value()
		} else if sym.IsType() && sym.Type().IsStruct() {
			st := sym.Type().Struct()
			st = a.resolveStruct(scope, st, path.Generics, name.Span())
			method, exists := a.GetStructMethod(st, raw)
			if !exists {
				return nil
			}
			val := ast.NewValue(method)
			sym := ast.NewSymbol(raw, &val, method.Def.Name.Span(), method.Def.Public)
			path.ResolvedSymbol = &sym
			a.AddSpanSymbol(name.Span(), sym)
			return &val
		}
		return nil
	})
	if t == nil {
		a.Panic(fmt.Sprintf("Value `%s` not found", path.String()), path.Span())
	}
	return *t
}

func (a *Analysis) resolvePathSymbol(scope *Scope, path *ast.Path) Symbol {
	t := resolvePathGeneric(a, scope, path, func(sym *Symbol, name ast.Ident) *Symbol {
		raw := name.Raw
		if !sym.IsImport() {
			return nil
		}
		sym = getImportScope(sym.Import()).GetSymbol(raw)
		if sym == nil {
			return nil
		}
		if len(path.Idents) > 1 && !sym.IsPublic() {
			a.Panic(fmt.Sprintf("`%s` is private", raw), name.Span())
		}
		path.ResolvedSymbol = sym
		a.AddSpanSymbol(name.Span(), *sym)
		return sym
	})
	if t == nil {
		a.Panic(fmt.Sprintf("Symbol `%s` not found", path.String()), path.Span())
	}
	return *t
}

func (a *Analysis) resolvePathTrait(scope *Scope, path *ast.Path) *ast.SemTrait {
	sym := a.resolvePathSymbol(scope, path)
	if !sym.IsTrait() {
		a.Panic(fmt.Sprintf("trait `%s` not found", path.String()), path.Span())
	}
	return sym.Trait()
}
