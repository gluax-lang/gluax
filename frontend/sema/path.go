package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func resolvePathGeneric[T any](a *Analysis, scope *Scope, path *ast.Path, leafResolver func(*Scope, string) *T) *T {
	idents := path.Idents
	if len(idents) == 0 {
		return nil
	}

	// aliasFirst := idents[0].Raw
	// if expansion, ok := a.UseAliases[aliasFirst]; ok {
	// 	newIdents := make([]ast.Ident, 0, len(expansion)+(len(idents)-1))
	// 	for _, seg := range expansion {
	// 		newIdents = append(newIdents, lexer.NewTokIdent(seg, path.Span()))
	// 	}
	// 	newIdents = append(newIdents, idents[1:]...)
	// 	path.Idents = newIdents
	// 	idents = path.Idents
	// }

	current := scope
	syms := make([]Symbol, 0, len(idents))
	for i, ident := range idents[:len(idents)-1] {
		name := ident.Raw
		imp := current.GetImport(name)
		if imp == nil {
			// nothing imported by that name -> bail out
			return nil
		}

		// only enforce public for imports after the first one, because first one is in our scope already
		if i > 0 && !current.IsSymbolPublic(name) {
			return nil
		}

		syms = append(syms, *current.GetSymbol(name))

		// drill into the imported package's scope
		current = imp.Analysis.(*Analysis).Scope
	}

	leafName := idents[len(idents)-1].Raw

	if len(idents) > 1 {
		if sym := current.GetSymbol(leafName); sym != nil && !sym.IsPublic() {
			return nil
		}
	}

	if sym := current.GetSymbol(leafName); sym != nil {
		syms = append(syms, *sym)
	}
	path.Symbols = syms

	return leafResolver(current, leafName)
}

func (a *Analysis) resolvePathType(scope *Scope, path *ast.Path) Type {
	t := resolvePathGeneric(a, scope, path, func(sc *Scope, name string) *Type {
		return sc.GetType(name)
	})
	if t == nil {
		a.Panic(fmt.Sprintf("Type `%s` not found", path.String()), path.Span())
	}
	return *t
}

func (a *Analysis) resolvePathValue(scope *Scope, path *ast.Path) Value {
	t := resolvePathGeneric(a, scope, path, func(sc *Scope, name string) *Value {
		return sc.GetValue(name)
	})
	if t == nil {
		a.Panic(fmt.Sprintf("Value `%s` not found", path.String()), path.Span())
	}
	return *t
}

func (a *Analysis) resolvePathSymbol(scope *Scope, path *ast.Path) Symbol {
	t := resolvePathGeneric(a, scope, path, func(sc *Scope, name string) *Symbol {
		return sc.GetSymbol(name)
	})
	if t == nil {
		a.Panic(fmt.Sprintf("Symbol `%s` not found", path.String()), path.Span())
	}
	return *t
}
