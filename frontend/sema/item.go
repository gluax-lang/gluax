package sema

import (
	"slices"

	"github.com/gluax-lang/gluax/frontend/ast"
)

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

		iterFunc := firstReturn.Function()
		if iterFunc.HasVarargReturn() {
			a.Errorf(iterFunc.Return.Span(), "method `%s` iterator function cannot have vararg return", methodName)
			return
		}
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

func (a *Analysis) handleItems(astD *ast.Ast) {
	// TODO: handle recursion if a let statement calls a function that uses the let statement

	for _, imp := range astD.Imports {
		a.handleImport(a.Scope, imp)
	}

	for _, use := range astD.Uses {
		a.handleUse(a.Scope, use)
	}

	for _, traitDef := range astD.Traits {
		traitDef.Scope = a.Scope
		trait := ast.NewSemTrait(traitDef)
		trait.Scope = a.Scope.Child(false)
		if err := a.Scope.AddTrait(traitDef.Name.Raw, &trait, traitDef.Span(), traitDef.Public); err != nil {
			a.Error(traitDef.Span(), err.Error())
		}
		traitDef.Sem = &trait
	}

	for _, traitDef := range astD.Traits {
		for _, super := range traitDef.SuperTraits {
			superDef := a.resolvePathSymbol(a.Scope, &super)
			if !superDef.IsTrait() {
				a.panic(super.Span(), "expected trait")
			}
			trait := traitDef.Sem
			superTrait := superDef.Trait()
			if causesTraitCycle(trait, superTrait) {
				a.panicf(super.Span(), "cyclic supertrait: trait `%s` is (directly or indirectly) a supertrait of itself", trait.Def.Name.Raw)
			}
			trait.SuperTraits = append(trait.SuperTraits, superTrait)
		}
	}

	for _, stDef := range astD.Classes {
		stDef.Scope = a.Scope
		st := a.setupClass(stDef, nil)

		SelfSt := a.setupClass(stDef, nil)
		for i, g := range SelfSt.Def.Generics.Params {
			traits := getGenericParamTraits(g)
			SelfSt.Generics.Params[i] = ast.NewSemGenericType(g.Name, traits, true)
		}
		SelfStTy := ast.NewSemType(SelfSt, stDef.Span())
		SelfStScope := SelfSt.Scope.(*Scope)
		SelfStScope.ForceAddType("Self", SelfStTy)
		stScope := st.Scope.(*Scope)
		stScope.ForceAddType("Self", SelfStTy)

		stSem := ast.NewSemType(st, stDef.Name.Span())
		a.AddTypeVisibility(a.Scope, stDef.Name.Raw, stSem, stDef.Public)
	}

	for _, stDef := range astD.Classes {
		superDef := stDef.Super
		if superDef == nil {
			continue
		}
		st := a.GetClass(stDef, nil)
		stScope := st.Scope.(*Scope)

		superT := a.resolveType(stScope, *superDef)
		if !superT.IsClass() {
			a.panicf((*superDef).Span(), "expected class type, got: %s", superT.String())
		}

		st.Super = superT.Class()
	}

	for _, stDef := range astD.Classes {
		st := a.GetClass(stDef, nil)
		stScope := st.Scope.(*Scope)
		SelfSt := stScope.GetType("Self").Class()
		a.collectClassFields(SelfSt)
		a.collectClassFields(st)
	}

	for _, funcDef := range astD.Funcs {
		funcSem := a.handleFunctionSignature(a.Scope, funcDef)
		funcDef.SetSem(&funcSem)
		a.AddValueVisibility(a.Scope, funcDef.Name.Raw, ast.NewValue(funcSem), funcDef.Name.Span(), funcDef.Public)
	}

	var implClassesChecks []func()
	for _, impl := range astD.ImplClasses {
		impl.Scope = a.Scope
		genericsScope := a.setupTypeGenerics(a.Scope, impl.Generics, nil)
		stTy := a.resolveType(genericsScope, impl.Class)
		if !stTy.IsClass() {
			a.panicf(impl.Class.Span(), "expected class type, got: %s", stTy.String())
		}
		if err := genericsScope.AddType("Self", stTy); err != nil {
			a.Error(impl.Class.Span(), err.Error())
		}
		st := stTy.Class()
		if st.Def.Attributes.Has("no_impl") {
			a.panicf(impl.Span(), "class `%s` cannot implement methods", st.Def.Name.Raw)
		}
		for _, method := range impl.Methods {
			funcTy := a.handleFunctionSignature(genericsScope, &method)
			funcTy.Scope = a.Scope
			funcTy.Generics = impl.Generics
			methodName := method.Name.Raw
			a.RegisterClassMethod(st, funcTy)
			implClassesChecks = append(implClassesChecks, func() {
				// this hack is needed, so something like `__x_iter_range` can check if `__x_iter_range_bound` exists or not
				a.checkClassMethods(st, methodName)
			})
		}
		impl.GenericsScope = genericsScope
	}

	for _, runCheck := range implClassesChecks {
		runCheck()
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
				a.panicf(method.Name.Span(), "trait `%s` method `%s` must have a `self` parameter as the first parameter", traitDef.Name.Raw, method.Name.Raw)
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
				if _, exists := a.GetTraitMethod(superTrait, name); exists {
					a.panicf(method.Def.Name.Span(), "cannot redefine method `%s`: already defined in supertrait `%s`", name, superTrait.Def.Name.Raw)
				}
			}
		}
	}

	var checks []func()
	for _, implTrait := range astD.ImplTraits {
		traitPath := a.resolvePathSymbol(a.Scope, &implTrait.Trait)
		if !traitPath.IsTrait() {
			a.panic(implTrait.Trait.Span(), "expected trait")
		}
		trait := traitPath.Trait()

		genericsScope := a.setupTypeGenerics(a.Scope, implTrait.Generics, nil)

		stTy := a.resolveType(genericsScope, implTrait.Class)
		if !stTy.IsClass() {
			a.panic(implTrait.Class.Span(), "expected class")
		}
		st := stTy.Class()

		if trait.Def.Attributes.Has("requires_metatable") && st.Def.Attributes.Has("no_metatable") {
			a.panicf(implTrait.Span(), "class `%s` cannot implement trait `%s` because it has no metatable", st.Def.Name.Raw, trait.Def.Name.Raw)
		}

		checks = append(checks, func() {
			for _, superTrait := range trait.SuperTraits {
				if !a.ClassImplementsTrait(st, superTrait) {
					a.panicf(implTrait.Span(), "class `%s` must implement supertrait `%s`", st.Def.Name.Raw, superTrait.Def.Name.Raw)
				}
			}
		})

		var methods = make(map[string]ast.SemFunction, len(trait.Methods))
		for name, method := range trait.Methods {
			stMethod := a.FindClassMethod(st, name)
			if stMethod == nil {
				if method.Def.Body != nil {
					// a.RegisterStructMethod(st, method)
					methods[name] = method
					continue
				} else {
					methods[name] = method
					continue
					a.panicf(implTrait.Span(), "class `%s` does not implement trait `%s` method `%s`", st.Def.Name.Raw, trait.Def.Name.Raw, name)
				}
			}
			params := stMethod.Def.Params
			if len(params) < 1 || params[0].Name.Raw != "self" {
				a.panicf(implTrait.Span(), "class `%s` method `%s` must have a `self` parameter as the first parameter", st.Def.Name.Raw, name)
			}

			methodCopy := method
			methodCopy.Params = append([]Type{}, method.Params[1:]...)

			stMethodCopy := *stMethod
			stMethodCopy.Params = append([]Type{}, stMethod.Params[1:]...)

			stMethodTy := ast.NewSemType(stMethodCopy, st.Def.Name.Span())
			if !a.matchFunctionType(methodCopy, stMethodTy) {
				a.panicf(implTrait.Span(), "method `%s` doesn't match trait `%s`: expected %s, got %s", name, trait.Def.Name.Raw, method.String(), stMethodTy.String())
			}
		}

		a.RegisterClassTraitImplementation(st, trait, methods, implTrait.Span())
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

	for _, implClass := range astD.ImplClasses {
		for _, method := range implClass.Methods {
			_ = a.handleFunction(implClass.GenericsScope.(*Scope), &method)
		}
	}

	for _, traitCheck := range traitsChecks {
		traitCheck()
	}

	a.CheckConflictingMethodImplementations()
	a.CheckConflictingTraitImplementations()
}

func (a *Analysis) handleUse(scope *Scope, it *ast.Use) {
	sym := a.resolvePathSymbol(scope, &it.Path)

	sym.SetPublic(it.Public)

	if err := scope.AddSymbol(it.NameIdent().Raw, sym); err != nil {
		a.Error(it.Span(), err.Error())
	}
}

func (a *Analysis) GetTraitMethod(trait *ast.SemTrait, name string) (ast.SemFunction, bool) {
	method, exists := trait.Methods[name]
	if exists {
		return method, true
	}
	for _, super := range trait.SuperTraits {
		if superMethod, exists := a.GetTraitMethod(super, name); exists {
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
