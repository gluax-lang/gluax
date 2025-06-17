package sema

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
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
				a.panicf(ident.Span(), "`%s` is private", ident.Raw)
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
			a.panicf(name.Span(), "`%s` is private", name.Raw)
		}
		path.ResolvedSymbol = sym
		a.AddSpanSymbol(name.Span(), *sym)
		return sym.Type()
	})
	if t == nil {
		a.panicf(path.Span(), "type `%s` not found", path.String())
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
				a.panicf(name.Span(), "`%s` is private", raw)
			}
			path.ResolvedSymbol = sym
			a.AddSpanSymbol(name.Span(), *sym)
			return sym.Value()
		} else if sym.IsType() && sym.Type().IsClass() {
			st := sym.Type().Class()
			st = a.resolveClass(scope, st, path.Generics, name.Span())
			methods := a.FindClassOrTraitMethod(st, raw)
			var method SemFunction
			if len(methods) == 1 {
				method = methods[0]
			} else if len(methods) > 1 {
				a.panicf(path.Span(), "multiple methods found for `%s` in class `%s`", raw, st.Def.Name.Raw)
			} else {
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
		a.panicf(path.Span(), "value `%s` not found", path.String())
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
			a.panicf(name.Span(), "`%s` is private", raw)
		}
		path.ResolvedSymbol = sym
		a.AddSpanSymbol(name.Span(), *sym)
		return sym
	})
	if t == nil {
		a.panicf(path.Span(), "symbol `%s` not found", path.String())
	}
	return *t
}

func (a *Analysis) resolvePathTrait(scope *Scope, path *ast.Path) *ast.SemTrait {
	sym := a.resolvePathSymbol(scope, path)
	if !sym.IsTrait() {
		a.panicf(path.Span(), "expected trait type")
	}
	return sym.Trait()
}
