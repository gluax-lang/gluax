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
				span := common.SpanSrc(getImportAnalysis(imp).Src)
				customSym.SetSpan(span)
				a.AddRef(customSym, ident.Span())
			} else {
				a.AddRef(*currentSym, ident.Span())
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
		a.AddRef(*sym, name.Span())
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
			a.AddRef(*sym, name.Span())
			return sym.Value()
		} else if sym.IsType() {
			baseTy := sym.Type()
			var resolvedTy Type

			if baseTy.IsClass() {
				st := a.resolveClass(scope, baseTy.Class(), path.Generics, name.Span())
				resolvedTy = ast.NewSemType(st, baseTy.Span())
			} else {
				if len(path.Generics) > 0 {
					a.panicf(path.Span(), "cannot specify generics for non-class type `%s`", baseTy.String())
				}
				resolvedTy = *baseTy
			}

			methods := a.findMethodsOnType(scope, resolvedTy, raw)
			if len(methods) == 0 {
				return nil // Not found
			}
			if len(methods) > 1 {
				a.panicf(path.Span(), "ambiguous method `%s` on type `%s`", raw, resolvedTy.String())
			}

			method := methods[0]

			if resolvedTy.IsGeneric() {
				childScope := NewScope(method.Scope.(*Scope))
				if err := childScope.AddType("Self", resolvedTy); err != nil {
					a.Error(resolvedTy.Span(), err.Error())
				}
				methodScope := method.Scope
				method = a.handleFunctionSignature(childScope, &method.Def)
				method.Scope = methodScope
			}

			val := ast.NewValue(method)
			valSym := ast.NewSymbol(raw, &val, method.Def.Name.Span(), method.Def.Public)
			path.ResolvedSymbol = &valSym
			a.AddRef(valSym, name.Span())
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
		a.AddRef(*sym, name.Span())
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
