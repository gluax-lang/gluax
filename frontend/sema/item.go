package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (a *Analysis) handleItems(items []ast.Item) {
	// TODO: handle recursion if a let statement calls a function that uses the let statement

	for _, item := range items {
		switch it := item.(type) {
		case *ast.Import:
			a.handleImport(a.Scope, it)
		}
	}

	for _, item := range items {
		switch it := item.(type) {
		case *ast.Use:
			a.handleUse(a.Scope, it)
		}
	}

	{
		fakeScope := NewScope(a.Scope)
		var fakeSymbol ast.Symbol
		for _, item := range items {
			var name lexer.TokIdent
			switch it := item.(type) {
			case *ast.Let:
				for _, name := range it.Names {
					err := fakeScope.AddSymbol(name.Raw, fakeSymbol)
					if err != nil {
						a.Panic(err.Error(), name.Span())
					}
				}
				continue
			case *ast.Struct:
				name = it.Name
			case *ast.Function:
				name = *it.Name
			case *ast.Trait:
				name = it.Name
			default:
				continue
			}
			if err := fakeScope.AddSymbol(name.Raw, fakeSymbol); err != nil {
				a.Panic(err.Error(), name.Span())
			}
		}
	}

	// struct names with their generics phase
	for _, item := range items {
		switch it := item.(type) {
		case *ast.Struct:
			it.Scope = a.Scope
			st := a.setupStruct(it, nil)

			SelfSt := a.setupStruct(it, nil)
			for i, g := range SelfSt.Generics.Params {
				SelfSt.Generics.Params[i] = ast.NewSemGenericType(g.Generic().Ident, true)
			}
			SelfStTy := ast.NewSemType(SelfSt, it.Span())
			SelfStScope := SelfSt.Scope.(*Scope)
			SelfStScope.ForceAddType("Self", SelfStTy)
			stScope := st.Scope.(*Scope)
			stScope.ForceAddType("Self", SelfStTy)

			stSem := ast.NewSemType(st, it.Name.Span())
			a.AddTypeVisibility(a.Scope, it.Name.Raw, stSem, it.Public)
		case *ast.Trait:
			it.Scope = a.Scope
			trait := ast.NewSemTrait(it)
			trait.Scope = a.Scope.Child(false)
			_ = a.Scope.AddTrait(it.Name.Raw, trait, it.Span(), it.Public)
		}
	}

	// struct fields and methods signature phase
	for _, item := range items {
		switch it := item.(type) {
		case *ast.Struct:
			st := a.State.GetStruct(it, nil)
			stScope := st.Scope.(*Scope)
			SelfSt := stScope.GetType("Self").Struct()
			a.collectStructFields(SelfSt)
			a.collectStructFields(st)
		case *ast.Function:
			funcSem := a.handleFunctionSignature(a.Scope, it)
			it.SetSem(&funcSem)
			a.AddValueVisibility(a.Scope, it.Name.Raw, ast.NewValue(funcSem), it.Name.Span(), it.Public)
		case *ast.ImplStruct:
			it.Scope = a.Scope
			genericsScope := NewScope(a.Scope)
			for _, g := range it.Generics.Params {
				binding := ast.NewSemGenericType(g.Name, true)
				a.AddType(genericsScope, g.Name.Raw, binding)
			}
			stTy := a.resolveType(genericsScope, it.Struct)
			if !stTy.IsStruct() {
				a.Panic(fmt.Sprintf("expected struct type, got: %s", stTy.String()), it.Struct.Span())
			}
			if err := genericsScope.AddType("Self", stTy); err != nil {
				a.Error(err.Error(), it.Struct.Span())
			}
			st := stTy.Struct()
			if st.Def.Attributes.Has("no_impl") {
				a.Panic(fmt.Sprintf("struct `%s` cannot implement methods", st.Def.Name.Raw), it.Span())
			}
			for _, method := range it.Methods {
				funcTy := a.handleFunctionSignature(genericsScope, &method)
				funcTy.ImplStruct = it
				a.addStructMethod(st, funcTy)
			}
			it.GenericsScope = genericsScope
		case *ast.Trait:
			trait := a.Scope.GetTrait(it.Name.Raw)
			scope := trait.Scope.(*Scope)
			for _, method := range it.Methods {
				funcTy := a.handleFunctionSignature(scope, &method)
				trait.Methods[method.Name.Raw] = funcTy
			}
		}
	}

	for _, item := range items {
		switch it := item.(type) {
		case *ast.ImplTraitForStruct:
			traitPath := a.resolvePathSymbol(a.Scope, &it.Trait)
			if !traitPath.IsTrait() {
				a.Panic("expected trait", it.Trait.Span())
			}
			trait := traitPath.Trait()

			stTy := a.resolveType(a.Scope, it.Struct)
			if !stTy.IsStruct() {
				a.Panic("expected struct", it.Struct.Span())
			}
			st := stTy.Struct()

			if trait.Def.Attributes.Has("requires_metatable") && st.Def.Attributes.Has("no_metatable") {
				a.Panic(fmt.Sprintf("struct `%s` must have a metatable to implement trait `%s`", st.Def.Name.Raw, trait.Def.Name.Raw), it.Span())
			}

			for name, method := range trait.Methods {
				stMethod, exists := a.getStructMethod(st, name)
				if !exists {
					if method.Def.Body != nil {

					} else {
						a.Panic(fmt.Sprintf("struct `%s` does not implement trait `%s` method `%s`", st.Def.Name.Raw, trait.Def.Name.Raw, name), it.Span())
					}
				}
				stMethodTy := ast.NewSemType(stMethod, st.Def.Name.Span())
				if !method.StrictMatches(stMethodTy) {
					a.Panic(fmt.Sprintf("method `%s` doesn't match trait `%s`: expected %s, got %s", name, trait.Def.Name.Raw, method.String(), stMethodTy.String()), it.Span())
				}
			}
		}
	}

	// let statements phase
	for _, item := range items {
		switch it := item.(type) {
		case *ast.Let:
			a.handleLet(a.Scope, it)
		}
	}

	// struct methods phase
	for _, item := range items {
		switch it := item.(type) {
		case *ast.Function:
			a.handleFunction(a.Scope, it)
		case *ast.ImplStruct:
			for _, method := range it.Methods {
				_ = a.handleFunction(it.GenericsScope.(*Scope), &method)
			}
		case *ast.Trait:
			trait := a.Scope.GetTrait(it.Name.Raw)
			scope := trait.Scope.(*Scope)
			for _, method := range it.Methods {
				funcTy := a.handleFunction(scope, &method)
				trait.Methods[method.Name.Raw] = funcTy
			}
		}
	}
}

func (a *Analysis) handleUse(scope *Scope, it *ast.Use) {
	sym := a.resolvePathSymbol(scope, &it.Path)

	sym.SetPublic(it.Public)

	if err := scope.AddSymbol(it.NameIdent().Raw, sym); err != nil {
		a.Error(err.Error(), it.Span())
	}
}
