package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

type methodCheck struct {
	params     int // exact required param count
	maxReturns int // 0 = no limit
	extra      func(a *Analysis, st *SemClass, fun *SemFunction)
}

var methodChecks = map[string]methodCheck{
	"__tostring": {params: 1, maxReturns: 1, extra: func(a *Analysis, _ *SemClass, fun *SemFunction) {
		if !fun.FirstReturnType().IsString() {
			a.Errorf(fun.FirstReturnType().Span(), "return value must be a string type")
		}
	}},
	"__unm": {params: 1, maxReturns: 1},
	"__eq": {params: 2, maxReturns: 1, extra: func(a *Analysis, _ *SemClass, fun *SemFunction) {
		a.Matches(fun.Params[0], fun.Params[1], fun.Params[1].Span())
		a.Matches(a.boolType(), fun.FirstReturnType(), fun.FirstReturnType().Span())
	}},
	"__x_iter_pairs": {params: 1, maxReturns: 3, extra: func(a *Analysis, _ *SemClass, fun *SemFunction) {
		first := fun.FirstReturnType()
		if !first.IsFunction() {
			a.Errorf(first.Span(), "first return value must be a function type")
			return
		}
		checkPairsIterFunc(a, first.Function())
	}},
	"__x_iter_range": {params: 2, maxReturns: 1, extra: func(a *Analysis, st *SemClass, fun *SemFunction) {
		if st.GetMethod("__x_iter_range_bound", true) == nil {
			a.Errorf(fun.Def.Name.Span(), "class `%s` must implement method `__x_iter_range_bound` to use `__x_iter_range`", st.Def.Name.Raw)
			return
		}
		if !fun.Params[1].IsNumber() {
			a.Errorf(fun.Def.Params[1].Type.Span(), "second parameter must be a number type")
		}
	}},
	"__x_iter_range_bound": {params: 1, maxReturns: 1, extra: func(a *Analysis, st *SemClass, fun *SemFunction) {
		if st.GetMethod("__x_iter_range", true) == nil {
			a.Errorf(fun.Def.Name.Span(), "class `%s` must implement method `__x_iter_range` to use `__x_iter_range_bound`", st.Def.Name.Raw)
			return
		}
		if !fun.FirstReturnType().IsNumber() {
			a.Errorf(fun.FirstReturnType().Span(), "return value must be a number type")
		}
	}},
}

func init() {
	// Arithmetic meta-methods all share the same shape: 2 params, 1 return.
	for name := range ast.ArithmeticMetaMethods {
		methodChecks[name] = methodCheck{params: 2, maxReturns: 1}
	}
}

func (a *Analysis) checkClassMethods(st *SemClass, methodName string) {
	check, ok := methodChecks[methodName]
	if !ok {
		return
	}
	fun := st.GetMethod(methodName, true)
	if fun.IsStatic() {
		a.Errorf(fun.Span(), "method `%s` cannot be static", methodName)
		return
	}
	if len(fun.Params) != check.params {
		a.Errorf(fun.Def.Span(), "method `%s` must have %d parameter(s)", methodName, check.params)
		return
	}
	if fun.HasVarargReturn() {
		a.Errorf(fun.Return.Span(), "method `%s` cannot have vararg return", methodName)
		return
	}
	if check.maxReturns > 0 && fun.ReturnCount() > check.maxReturns {
		a.Errorf(fun.Return.Span(), "method `%s` cannot have more than %d return value(s)", methodName, check.maxReturns)
		return
	}
	if check.extra != nil {
		check.extra(a, st, fun)
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

func (a *Analysis) handleUse(scope *Scope, it *ast.Use) {
	sym := a.resolvePathSymbol(scope, &it.Path)

	// to not change the original symbol visibility
	symCopy := *sym
	symCopy.SetPublic(it.Public)

	if err := scope.AddSymbol(it.NameIdent().Raw, &symCopy); err != nil {
		a.Error(it.Span(), err.Error())
	}
}
