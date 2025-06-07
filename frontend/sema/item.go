package sema

import (
	"fmt"
	"slices"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleItems(astD *ast.Ast) {
	// TODO: handle recursion if a let statement calls a function that uses the let statement

	for _, imp := range astD.Imports {
		a.handleImport(a.Scope, imp)
	}

	for _, use := range astD.Uses {
		a.handleUse(a.Scope, use)
	}

	for _, stDef := range astD.Structs {
		stDef.Scope = a.Scope
		st := a.setupStruct(stDef, nil)

		SelfSt := a.setupStruct(stDef, nil)
		for i, g := range SelfSt.Generics.Params {
			SelfSt.Generics.Params[i] = ast.NewSemGenericType(g.Generic().Ident, true)
		}
		SelfStTy := ast.NewSemType(SelfSt, stDef.Span())
		SelfStScope := SelfSt.Scope.(*Scope)
		SelfStScope.ForceAddType("Self", SelfStTy)
		stScope := st.Scope.(*Scope)
		stScope.ForceAddType("Self", SelfStTy)

		stSem := ast.NewSemType(st, stDef.Name.Span())
		a.AddTypeVisibility(a.Scope, stDef.Name.Raw, stSem, stDef.Public)
	}

	for _, traitDef := range astD.Traits {
		traitDef.Scope = a.Scope
		trait := ast.NewSemTrait(traitDef)
		trait.Scope = a.Scope.Child(false)
		if err := a.Scope.AddTrait(traitDef.Name.Raw, &trait, traitDef.Span(), traitDef.Public); err != nil {
			a.Error(err.Error(), traitDef.Span())
		}
		traitDef.Sem = &trait
	}

	for _, traitDef := range astD.Traits {
		for _, super := range traitDef.SuperTraits {
			superDef := a.resolvePathSymbol(a.Scope, &super)
			if !superDef.IsTrait() {
				a.Panic("expected trait", super.Span())
			}
			trait := traitDef.Sem
			superTrait := superDef.Trait()
			if causesTraitCycle(trait, superTrait) {
				a.Panic(fmt.Sprintf("cyclic supertrait: trait `%s` is (directly or indirectly) a supertrait of itself", trait.Def.Name.Raw), super.Span())
			}
			trait.SuperTraits = append(trait.SuperTraits, superTrait)
		}
	}

	for _, stDef := range astD.Structs {
		st := a.State.GetStruct(stDef, nil)
		stScope := st.Scope.(*Scope)
		SelfSt := stScope.GetType("Self").Struct()
		a.collectStructFields(SelfSt)
		a.collectStructFields(st)
	}

	for _, funcDef := range astD.Funcs {
		funcSem := a.handleFunctionSignature(a.Scope, funcDef)
		funcDef.SetSem(&funcSem)
		a.AddValueVisibility(a.Scope, funcDef.Name.Raw, ast.NewValue(funcSem), funcDef.Name.Span(), funcDef.Public)
	}

	for _, impl := range astD.ImplStructs {
		impl.Scope = a.Scope
		genericsScope := a.setupTypeGenerics(a.Scope, impl.Generics, nil)
		stTy := a.resolveType(genericsScope, impl.Struct)
		if !stTy.IsStruct() {
			a.Panic(fmt.Sprintf("expected struct type, got: %s", stTy.String()), impl.Struct.Span())
		}
		if err := genericsScope.AddType("Self", stTy); err != nil {
			a.Error(err.Error(), impl.Struct.Span())
		}
		st := stTy.Struct()
		if st.Def.Attributes.Has("no_impl") {
			a.Panic(fmt.Sprintf("struct `%s` cannot implement methods", st.Def.Name.Raw), impl.Span())
		}
		for _, method := range impl.Methods {
			funcTy := a.handleFunctionSignature(genericsScope, &method)
			funcTy.Scope = a.Scope
			funcTy.Generics = impl.Generics
			a.addStructMethod(st, funcTy)
		}
		impl.GenericsScope = genericsScope
	}

	var traitsChecks []func()
	for _, traitDef := range astD.Traits {
		trait := traitDef.Sem
		scope := trait.Scope.(*Scope)
		SelfScope := scope.Child(false)
		SelfScope.ForceAddType("Self", ast.NewSemDynTrait(trait, traitDef.Name.Span()))
		for _, method := range traitDef.Methods {
			params := method.Params
			if len(params) < 1 || params[0].Name.Raw != "self" {
				a.Panic(fmt.Sprintf("trait `%s` method `%s` must have a `self` parameter as the first parameter", traitDef.Name.Raw, method.Name.Raw), method.Name.Span())
			}
			funcTy := a.handleFunctionSignature(SelfScope, &method)
			funcTy.Scope = scope
			trait.Methods[method.Name.Raw] = funcTy
			traitsChecks = append(traitsChecks, func() {
				trait.Methods[method.Name.Raw] = a.handleFunction(SelfScope, &method)
			})
		}
	}

	for _, traitDef := range astD.Traits {
		trait := traitDef.Sem
		for name, method := range trait.Methods {
			for _, superTrait := range trait.SuperTraits {
				if _, exists := a.getTraitMethod(superTrait, name); exists {
					a.Panic(fmt.Sprintf(
						"cannot redefine method `%s`: already defined in supertrait `%s`",
						name, superTrait.Def.Name.Raw,
					), method.Def.Name.Span())
				}
			}
		}
	}

	var checks []func()
	for _, implTrait := range astD.ImplTraits {
		traitPath := a.resolvePathSymbol(a.Scope, &implTrait.Trait)
		if !traitPath.IsTrait() {
			a.Panic("expected trait", implTrait.Trait.Span())
		}
		trait := traitPath.Trait()

		genericsScope := a.setupTypeGenerics(a.Scope, implTrait.Generics, nil)

		stTy := a.resolveType(genericsScope, implTrait.Struct)
		if !stTy.IsStruct() {
			a.Panic("expected struct", implTrait.Struct.Span())
		}
		st := stTy.Struct()

		if trait.Def.Attributes.Has("requires_metatable") && st.Def.Attributes.Has("no_metatable") {
			a.Panic(fmt.Sprintf("struct `%s` must have a metatable to implement trait `%s`", st.Def.Name.Raw, trait.Def.Name.Raw), implTrait.Span())
		}

		checks = append(checks, func() {
			for _, superTrait := range trait.SuperTraits {
				if !a.structHasTrait(st, superTrait) {
					a.Panic(fmt.Sprintf("struct `%s` must implement supertrait `%s`", st.Def.Name.Raw, superTrait.Def.Name.Raw), implTrait.Span())
				}
			}
		})

		for name, method := range trait.Methods {
			stMethod, exists := a.getStructMethod(st, name)
			if !exists {
				if method.Def.Body != nil {
					a.addStructMethod(st, method)
					continue
				} else {
					a.Panic(fmt.Sprintf("struct `%s` does not implement trait `%s` method `%s`", st.Def.Name.Raw, trait.Def.Name.Raw, name), implTrait.Span())
				}
			}
			params := stMethod.Def.Params
			if len(params) < 1 || params[0].Name.Raw != "self" {
				a.Panic(fmt.Sprintf("struct `%s` method `%s` must have a `self` parameter as the first parameter", st.Def.Name.Raw, name), implTrait.Span())
			}

			methodCopy := method
			methodCopy.Params = append([]Type{}, method.Params[1:]...)

			stMethodCopy := stMethod
			stMethodCopy.Params = append([]Type{}, stMethod.Params[1:]...)

			stMethodTy := ast.NewSemType(stMethodCopy, st.Def.Name.Span())
			if !methodCopy.StrictMatches(stMethodTy) {
				a.Panic(fmt.Sprintf("method `%s` doesn't match trait `%s`: expected %s, got %s", name, trait.Def.Name.Raw, method.String(), stMethodTy.String()), implTrait.Span())
			}
		}

		a.addStructTrait(st, trait, implTrait.Span())
	}

	for _, check := range checks {
		check()
	}

	for _, let := range astD.Lets {
		a.handleLet(a.Scope, let)
	}

	for _, funcDef := range astD.Funcs {
		a.handleFunction(a.Scope, funcDef)
	}

	for _, implStruct := range astD.ImplStructs {
		for _, method := range implStruct.Methods {
			_ = a.handleFunction(implStruct.GenericsScope.(*Scope), &method)
		}
	}

	for _, traitCheck := range traitsChecks {
		traitCheck()
	}
}

func (a *Analysis) handleUse(scope *Scope, it *ast.Use) {
	sym := a.resolvePathSymbol(scope, &it.Path)

	sym.SetPublic(it.Public)

	if err := scope.AddSymbol(it.NameIdent().Raw, sym); err != nil {
		a.Error(err.Error(), it.Span())
	}
}

func (a *Analysis) getTraitMethod(trait *ast.SemTrait, name string) (ast.SemFunction, bool) {
	method, exists := trait.Methods[name]
	if exists {
		return method, true
	}
	for _, super := range trait.SuperTraits {
		if superMethod, exists := a.getTraitMethod(super, name); exists {
			return superMethod, true
		}
	}
	return ast.SemFunction{}, false
}

func causesTraitCycle(trait *ast.SemTrait, super *ast.SemTrait) bool {
	if trait == super {
		return true
	}
	visited := map[*ast.SemTrait]struct{}{}
	var dfs func(t *ast.SemTrait) bool
	dfs = func(t *ast.SemTrait) bool {
		if t == trait {
			return true
		}
		if _, ok := visited[t]; ok {
			return false
		}
		visited[t] = struct{}{}
		return slices.ContainsFunc(t.SuperTraits, dfs)
	}
	return dfs(super)
}
