package sema

import (
	"slices"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func checkPairsIterFunc(a *Analysis, fun SemFunction) {
	if fun.HasVarargReturn() {
		a.Error(fun.Span(), "iterator function cannot have vararg return")
		return
	}

	for i, rt := range fun.ReturnTypes() {
		if !rt.IsNilable() {
			a.Errorf(rt.Span(), "iterator function return value %d must be nilable", i+1)
			return
		}
	}
}

var toCheckFuncs = map[string]func(*Analysis, *ast.SemClass, string){
	"__x_iter_pairs": func(a *Analysis, st *ast.SemClass, methodName string) {
		fun := a.FindClassMethod(st, methodName)
		if len(fun.Params) != 1 {
			a.Errorf(fun.Def.Span(), "method `%s` must have one parameter", methodName)
			return
		}

		if fun.Def.Params[0].Name.Raw != "self" {
			a.Error(fun.Def.Params[0].Type.Span(), "first parameter must be `self`")
			return
		}

		if fun.HasVarargReturn() {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
			return
		}

		if fun.ReturnCount() > 3 {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have more than 3 return values", methodName)
			return
		}

		firstReturn := fun.FirstReturnType()
		if !firstReturn.IsFunction() {
			a.Errorf(firstReturn.Span(), "first return value must be a function type")
			return
		}

		checkPairsIterFunc(a, firstReturn.Function())
	},
	"__x_iter_range": func(a *Analysis, st *ast.SemClass, methodName string) {
		fun := a.FindClassMethod(st, methodName)
		if len(fun.Params) != 2 {
			a.Errorf(fun.Def.Name.Span(), "method `%s` must have two parameters", methodName)
			return
		}

		if method := a.FindClassMethod(st, "__x_iter_range_bound"); method == nil {
			a.Errorf(fun.Def.Name.Span(), "class `%s` must implement method `__x_iter_range_bound` to use `%s`", st.Def.Name.Raw, methodName)
			return
		}

		if fun.Def.Params[0].Name.Raw != "self" {
			a.Errorf(fun.Def.Params[0].Type.Span(), "first parameter must be `self`")
			return
		}

		if !fun.Params[1].IsNumber() {
			a.Errorf(fun.Def.Params[1].Type.Span(), "second parameter must be a number type")
			return
		}

		if fun.HasVarargReturn() {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
			return
		}

		if fun.ReturnCount() != 1 {
			a.Errorf(fun.Return.Span(), "method `%s` must have exactly one return value", methodName)
			return
		}
	},
	"__x_iter_range_bound": func(a *Analysis, st *ast.SemClass, methodName string) {
		fun := a.FindClassMethod(st, methodName)
		if len(fun.Params) != 1 {
			a.Errorf(fun.Def.Name.Span(), "method `%s` must have one parameter", methodName)
			return
		}

		if method := a.FindClassMethod(st, "__x_iter_range"); method == nil {
			a.Errorf(fun.Def.Name.Span(), "class `%s` must implement method `__x_iter_range` to use `%s`", st.Def.Name.Raw, methodName)
			return
		}

		if fun.Def.Params[0].Name.Raw != "self" {
			a.Errorf(fun.Def.Params[0].Type.Span(), "first parameter must be `self`")
			return
		}

		if fun.HasVarargReturn() {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
			return
		}

		if fun.ReturnCount() != 1 {
			a.Errorf(fun.Return.Span(), "method `%s` must have exactly one return value", methodName)
			return
		}

		firstReturn := fun.FirstReturnType()
		if !firstReturn.IsNumber() {
			a.Errorf(firstReturn.Span(), "return value must be a number type")
			return
		}
	},
}

func (a *Analysis) checkClassMethods(st *ast.SemClass, methodName string) {
	if checkFunc, exists := toCheckFuncs[methodName]; exists {
		checkFunc(a, st, methodName)
	}
}

func (a *Analysis) handleUse(scope *Scope, it *ast.Use) {
	sym := a.resolvePathSymbol(scope, &it.Path)

	sym.SetPublic(it.Public)

	if err := scope.AddSymbol(it.NameIdent().Raw, sym); err != nil {
		a.Error(it.Span(), err.Error())
	}
}

func (a *Analysis) GetTraitMethods(trait *ast.SemTrait, name string) []ast.SemFunction {
	var found []ast.SemFunction
	for _, method := range trait.Methods {
		if method.Def.Name.Raw == name {
			found = append(found, method)
		}
	}
	for _, super := range trait.SuperTraits {
		found = append(found, a.GetTraitMethods(super, name)...)
	}
	return found
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

func traitImplements(trait *ast.SemTrait, target *ast.SemTrait) bool {
	if trait == target {
		return true
	}
	for _, super := range trait.SuperTraits {
		if traitImplements(super, target) {
			return true
		}
	}
	return false
}
