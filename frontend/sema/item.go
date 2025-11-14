package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

var arithmeticCheck = func(a *Analysis, st *ast.SemClass, methodName string) {
	fun := a.FindClassMethod(st, methodName)

	if fun.IsStatic() {
		a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
		return
	}

	if len(fun.Params) != 2 {
		a.Errorf(fun.Def.Span(), "method `%s` must have 2 parameters", methodName)
		return
	}

	if fun.HasVarargReturn() {
		a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
		return
	}

	if fun.ReturnCount() > 1 {
		a.Errorf(fun.Return.Span(), "method `%s` cannot have more than 1 return value", methodName)
		return
	}
}

func checkPairsIterFunc(a *Analysis, fun *SemFunction) {
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

		if fun.IsStatic() {
			a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
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

		if fun.IsStatic() {
			a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
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

		if fun.IsStatic() {
			a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
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
	"__tostring": func(a *Analysis, st *ast.SemClass, methodName string) {
		fun := a.FindClassMethod(st, methodName)

		if fun.IsStatic() {
			a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
			return
		}

		if len(fun.Params) != 1 {
			a.Errorf(fun.Def.Span(), "method `%s` must have 1 parameter", methodName)
			return
		}

		if fun.HasVarargReturn() {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
			return
		}

		if fun.ReturnCount() > 1 {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have more than 1 return value", methodName)
			return
		}

		firstReturn := fun.FirstReturnType()
		if !firstReturn.IsString() {
			a.Errorf(firstReturn.Span(), "return value must be a string type")
			return
		}
	},
	"__unm": func(a *Analysis, st *ast.SemClass, methodName string) {
		fun := a.FindClassMethod(st, methodName)

		if fun.IsStatic() {
			a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
			return
		}

		if len(fun.Params) != 1 {
			a.Errorf(fun.Def.Span(), "method `%s` must have 1 parameter", methodName)
			return
		}

		if fun.HasVarargReturn() {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
			return
		}

		if fun.ReturnCount() > 1 {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have more than 1 return value", methodName)
			return
		}
	},
	"__eq": func(a *Analysis, st *ast.SemClass, methodName string) {
		fun := a.FindClassMethod(st, methodName)

		if fun.IsStatic() {
			a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
			return
		}

		if len(fun.Params) != 2 {
			a.Errorf(fun.Def.Span(), "method `%s` must have 2 parameters", methodName)
			return
		}

		if fun.HasVarargReturn() {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
			return
		}

		if fun.ReturnCount() > 1 {
			a.Errorf(fun.Return.Span(), "method `%s` cannot have more than 1 return value", methodName)
			return
		}

		a.Matches(fun.Params[0], fun.Params[1], fun.Params[1].Span())
		a.Matches(a.boolType(), fun.FirstReturnType(), fun.FirstReturnType().Span())
	},
}

func init() {
	for methodName := range ast.ArithmeticMetaMethods {
		toCheckFuncs[methodName] = arithmeticCheck
	}
}

func (a *Analysis) checkClassMethods(st *ast.SemClass, methodName string) {
	if checkFunc, exists := toCheckFuncs[methodName]; exists {
		checkFunc(a, st, methodName)
	}
}

func (a *Analysis) handleUse(scope *Scope, it *ast.Use) {
	sym := a.resolvePathSymbol(scope, &it.Path)

	// to not change the original symbol visibility
	symCopy := *sym
	symCopy.SetPublic(it.Public)

	if err := scope.AddSymbol(it.NameIdent().Raw, &symCopy); err != nil {
		a.Error(it.Span(), err.Error())
	}
}
