package sema

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
	protocol "github.com/gluax-lang/lsp"
)

// don't expose actual paths to the code generation :]
func (pa *ProjectAnalysis) PathRelativeToWorkspace(path string) string {
	ws := pa.workspace
	ws, path = common.FilePathClean(ws), common.FilePathClean(path)
	return common.FilePathClean(filepath.Join(pa.CurrentPackage(), strings.TrimPrefix(path, ws)))
}

func (pa *ProjectAnalysis) StartsWithWorkspace(path string) bool {
	ws := common.FilePathClean(pa.workspace)
	path = common.FilePathClean(path)
	return strings.HasPrefix(path, ws)
}

func (pa *ProjectAnalysis) StripWorkspace(path string) string {
	ws := common.FilePathClean(pa.workspace) + "/"
	path = common.FilePathClean(path)
	rel := strings.TrimPrefix(path, ws)
	return rel
}

type Analysis struct {
	Src                   string // source file name
	Workspace             string // workspace root
	Scope                 *Scope // root scope
	Diags                 []Diagnostic
	InlayHints            []InlayHint
	TempIdx               *int
	Project               *ProjectAnalysis
	Ast                   *ast.Ast
	State                 *State // current state of the analysis
	currentClassSetupSpan *Span  // used to track the span of the current class setup
}

func (a *Analysis) SetClassSetupSpan(span Span) bool {
	if a.currentClassSetupSpan == nil {
		a.currentClassSetupSpan = &span
		return true
	}
	return false
}

func (a *Analysis) ClearClassSetupSpan() {
	a.currentClassSetupSpan = nil
}

func (a *Analysis) GetClassSetupSpan(def Span) Span {
	if a.currentClassSetupSpan == nil {
		return def
	}
	return *a.currentClassSetupSpan
}

func (a *Analysis) Error(span Span, msg string) {
	// println("\n-------------------------------------")
	// println(msg)
	// debug.PrintStack()
	// println("-------------------------------------\n")
	a.Diags = append(a.Diags, *common.ErrorDiag(msg, span))
}

func (a *Analysis) Errorf(span Span, format string, args ...any) {
	a.Error(span, fmt.Sprintf(format, args...))
}

func (a *Analysis) Warning(span Span, msg string) {
	a.Diags = append(a.Diags, *common.WarningDiag(msg, span))
}

func (a *Analysis) panic(span Span, msg string) {
	a.Error(span, msg)
	panic("")
}

func (a *Analysis) panicf(span Span, format string, args ...any) {
	a.Errorf(span, format, args...)
	panic("")
}

func (a *Analysis) AddType(scope *Scope, name string, ty Type) {
	if err := scope.AddType(name, ty); err != nil {
		a.Errorf(ty.Span(), "%s", err.Error())
	}
}

func (a *Analysis) AddTypeVisibility(scope *Scope, name string, ty Type, public bool) {
	if err := scope.AddTypeVisibility(name, ty, public); err != nil {
		a.Errorf(ty.Span(), "%s", err.Error())
	}
}

func (a *Analysis) AddValue(scope *Scope, name string, val Value, span Span) {
	if err := scope.AddValue(name, val, span); err != nil {
		a.Errorf(span, "%s", err.Error())
	}
}

func (a *Analysis) AddValueVisibility(scope *Scope, name string, val Value, span Span, public bool) {
	if err := scope.AddValueVisibility(name, val, span, public); err != nil {
		a.Errorf(span, "%s", err.Error())
	}
}

func (a *Analysis) AddLabel(scope *Scope, label *ast.Ident) {
	if err := scope.AddLabel(label.Raw); err != nil {
		a.Errorf(label.Span(), "%s", err.Error())
	}
}

func (a *Analysis) InlayHintType(label string, span Span) {
	kind := protocol.InlayHintKindType
	a.InlayHints = append(a.InlayHints, protocol.InlayHint{
		Position: protocol.Position{
			Line:      span.LineStart - 1,
			Character: span.ColumnEnd,
		},
		Label: []protocol.InlayHintLabelPart{
			{Value: label},
		},
		Kind: &kind,
	})
}

func (a *Analysis) getBuiltinType(name string) Type {
	scope := a.Scope
	ty := scope.GetType(name)
	if ty == nil {
		a.panicf(common.SpanDefault(), "unknown type: %s", name)
	}
	if ty.Kind() != ast.SemClassKind {
		a.panicf(common.SpanDefault(), "expected class type, got: %s", ty.Kind())
	}
	return *ty
}

func (a *Analysis) nilType() Type {
	return a.getBuiltinType("nil")
}

func (a *Analysis) boolType() Type {
	return a.getBuiltinType("bool")
}

func (a *Analysis) numberType() Type {
	return a.getBuiltinType("number")
}

func (a *Analysis) stringType() Type {
	return a.getBuiltinType("string")
}

func (a *Analysis) anyType() Type {
	return a.getBuiltinType("any")
}

func (a *Analysis) instantiateBuiltinClass(name string, span Span, params ...Type) Type {
	if a.SetClassSetupSpan(span) {
		defer a.ClearClassSetupSpan()
	}
	builtin := a.getBuiltinType(name)
	st := builtin.Class()
	newSt := a.instantiateClass(st.Def, params)
	return ast.NewSemType(newSt, span)
}

func (a *Analysis) vecType(t Type, span Span) Type {
	return a.instantiateBuiltinClass("vec", span, t)
}

func (a *Analysis) mapType(key, value Type, span Span) Type {
	return a.instantiateBuiltinClass("map", span, key, value)
}

func (a *Analysis) nilableType(t Type, span Span) Type {
	return a.instantiateBuiltinClass("nilable", span, t)
}

func (a *Analysis) tupleType(span Span, ty Type, other ...Type) Type {
	return ast.NewSemType(
		ast.SemTuple{Elems: append([]Type{ty}, other...)},
		span,
	)
}

func (a *Analysis) Matches(ty, other Type, span Span) {
	if !a.matchTypes(ty, other) {
		a.Errorf(span, "mismatched types, expected `%s`, got `%s`", ty.String(), other.String())
	}
}

func (a *Analysis) StrictMatches(ty, other Type, span Span) {
	if !a.MatchTypesStrict(ty, other) {
		a.Errorf(span, "mismatched types, expected `%s`, got `%s`", ty.String(), other.String())
	}
}

func (a *Analysis) MatchesPanic(ty, other Type, span Span) {
	if !a.matchTypes(ty, other) {
		a.panicf(span, "mismatched types, expected `%s`, got `%s`", ty.String(), other.String())
	}
}

func (a *Analysis) populateDeclarations() {
	astD := a.Ast
	for _, traitDef := range astD.Traits {
		traitDef.Scope = a.Scope
		trait := ast.NewSemTrait(traitDef)
		trait.Scope = a.Scope.Child(false)
		if err := a.Scope.AddTrait(traitDef.Name.Raw, &trait, traitDef.Span(), traitDef.Public); err != nil {
			a.Error(traitDef.Span(), err.Error())
		}
		traitDef.Sem = &trait
		a.AddDecl(trait)
	}

	for _, stDef := range astD.Classes {
		stDef.Scope = a.Scope
		st := a.setupClass(stDef, nil, false)
		stSem := ast.NewSemType(st, stDef.Name.Span())
		a.AddTypeVisibility(a.Scope, stDef.Name.Raw, stSem, stDef.Public)
		a.AddDecl(stSem)
	}

	for _, funcDef := range astD.Funcs {
		funcSem := a.handleFunctionSignature(a.Scope, funcDef)
		funcDef.SetSem(&funcSem)
		a.AddValueVisibility(a.Scope, funcDef.Name.Raw, ast.NewValue(funcSem), funcDef.Name.Span(), funcDef.Public)
		a.AddDecl(funcSem)
	}

	for _, letDef := range astD.Lets {
		for i, ident := range letDef.Names {
			if ident.Raw == "_" {
				a.panic(ident.Span(), "cannot use `_` in top level let binding")
			}

			ty := a.resolveType(a.Scope, *letDef.Types[i])
			variable := ast.NewVariable(*letDef, i, ty)
			val := ast.NewValue(variable)
			a.AddValueVisibility(a.Scope, ident.Raw, val, ident.Span(), letDef.Public)
		}
	}
}

func (a *Analysis) resolveImplementations() {
	for _, use := range a.Ast.Uses {
		a.handleUse(a.Scope, use)
	}

	for _, traitDef := range a.Ast.Traits {
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

	for _, stDef := range a.Ast.Classes {
		st := a.GetClass(stDef, nil)
		a.buildGenericsTable(st.Scope.(*Scope), st, nil)

		SelfSt := a.setupClass(stDef, nil, true)
		for i, g := range SelfSt.Def.Generics.Params {
			traits := getGenericParamTraits(g)
			SelfSt.Generics.Params[i] = ast.NewSemGenericType(g.Name, traits, true)
		}
		SelfStTy := ast.NewSemType(SelfSt, stDef.Span())
		SelfStScope := SelfSt.Scope.(*Scope)
		SelfStScope.ForceAddType("Self", SelfStTy)
		stScope := st.Scope.(*Scope)
		stScope.ForceAddType("Self", SelfStTy)
	}

	for _, stDef := range a.Ast.Classes {
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

		superClass := superT.Class()
		if superClass.Def.Attributes.Has("sealed") {
			a.panicf((*superDef).Span(), "cannot inherit from sealed class `%s`", superClass.Def.Name.Raw)
		}

		st.Super = superClass
	}

	for _, stDef := range a.Ast.Classes {
		st := a.GetClass(stDef, nil)
		stScope := st.Scope.(*Scope)
		SelfSt := stScope.GetType("Self").Class()
		a.collectClassFields(SelfSt)
		a.collectClassFields(st)

		for _, field := range st.Fields {
			a.AddDecl(field)
		}
	}

	for _, traitDef := range a.Ast.Traits {
		trait := traitDef.Sem
		scope := trait.Scope.(*Scope)
		SelfScope := scope.Child(false)
		SelfScope.ForceAddType("Self", ast.NewSemDynTrait(trait, traitDef.Name.Span()))
		for _, method := range traitDef.Methods {
			name := method.Name.Raw
			if _, exists := trait.Methods[name]; exists {
				a.panicf(method.Name.Span(), "duplicate method `%s` in trait `%s`", name, traitDef.Name.Raw)
			}
			params := method.Params
			if len(params) < 1 || params[0].Name.Raw != "self" {
				a.panicf(method.Name.Span(), "trait `%s` method `%s` must have a `self` parameter as the first parameter", traitDef.Name.Raw, method.Name.Raw)
			}
			funcTy := a.handleFunctionSignature(SelfScope, &method)
			funcTy.Scope = scope
			funcTy.Trait = trait
			trait.Methods[name] = funcTy
			traitDef.Checks = append(traitDef.Checks, func() {
				funcTy := a.handleFunction(SelfScope, &method)
				funcTy.Scope = scope
				funcTy.Trait = trait
				trait.Methods[name] = funcTy
			})
		}
	}

	for _, implTrait := range a.Ast.ImplTraits {
		traitPath := a.resolvePathSymbol(a.Scope, &implTrait.Trait)
		if !traitPath.IsTrait() {
			a.panic(implTrait.Trait.Span(), "expected trait")
		}
		trait := traitPath.Trait()
		implTrait.ResolvedTrait = trait

		genericsScope := a.setupTypeGenerics(a.Scope, implTrait.Generics, nil)

		stTy := a.resolveType(genericsScope, implTrait.Class)
		if !stTy.IsClass() {
			a.panic(implTrait.Class.Span(), "expected class")
		}
		if err := genericsScope.AddType("Self", stTy); err != nil {
			a.Error(stTy.Span(), err.Error())
		}
		st := stTy.Class()

		if !a.Project.StartsWithWorkspace(trait.Def.Span().Source) &&
			!a.Project.StartsWithWorkspace(st.Def.Span().Source) {
			a.panicf(implTrait.Span(),
				"cannot implement trait `%s` for type `%s` because neither is defined in this package",
				trait.Def.Name.Raw, st.Def.Name.Raw)
		}

		if trait.Def.Attributes.Has("requires_metatable") && st.Def.Attributes.Has("no_metatable") {
			a.panicf(implTrait.Span(), "class `%s` cannot implement trait `%s` because it has no metatable", st.Def.Name.Raw, trait.Def.Name.Raw)
		}

		implTrait.Checks = append(implTrait.Checks, func() {
			for _, superTrait := range trait.SuperTraits {
				if !a.ClassImplementsTrait(st, superTrait) {
					a.panicf(implTrait.Span(), "class `%s` must implement supertrait `%s`", st.Def.Name.Raw, superTrait.Def.Name.Raw)
				}
			}
		})

		implMethods := make(map[string]ast.SemFunction, len(implTrait.Methods))
		for _, method := range implTrait.Methods {
			if _, exists := implMethods[method.Name.Raw]; exists {
				a.panicf(method.Name.Span(), "duplicate method `%s` in trait implementation", method.Name.Raw)
			}
			funcTy := a.handleFunctionSignature(genericsScope, &method)
			funcTy.Scope = a.Scope
			funcTy.Generics = implTrait.Generics
			implMethods[method.Name.Raw] = funcTy
			implTrait.Checks = append(implTrait.Checks, func() {
				funcTy := a.handleFunction(genericsScope, &method)
				funcTy.Scope = a.Scope
				funcTy.Generics = implTrait.Generics
				implMethods[method.Name.Raw] = funcTy
			})
		}

		for name, method := range implMethods {
			if _, exists := trait.Methods[name]; !exists {
				a.panicf(method.Def.Name.Span(), "method `%s` is not a member of trait `%s`", name, trait.Def.Name.Raw)
			}
		}

		var methods = make(map[string]ast.SemFunction, len(trait.Methods))
		for name, method := range trait.Methods {
			stMethod, exists := implMethods[name]
			if !exists {
				if method.Def.Body != nil {
					// a.RegisterStructMethod(st, method)
					methods[name] = method
					continue
				} else {
					a.panicf(implTrait.Span(), "class `%s` does not implement trait `%s` method `%s`", st.Def.Name.Raw, trait.Def.Name.Raw, name)
				}
			}
			params := stMethod.Def.Params
			if len(params) < 1 || params[0].Name.Raw != "self" {
				a.panicf(implTrait.Span(), "class `%s` method `%s` must have a `self` parameter as the first parameter", st.Def.Name.Raw, name)
			}

			methodCopy := method
			methodCopy.Params = append([]Type{}, method.Params[1:]...)

			stMethodCopy := stMethod
			stMethodCopy.Params = append([]Type{}, stMethod.Params[1:]...)

			stMethodTy := ast.NewSemType(stMethodCopy, st.Def.Name.Span())
			if !a.matchFunctionType(methodCopy, stMethodTy) {
				a.panicf(implTrait.Span(), "method `%s` doesn't match trait `%s`: expected %s, got %s", name, trait.Def.Name.Raw, method.String(), stMethodTy.String())
			}

			stMethod.Trait = trait
			methods[name] = stMethod
		}

		a.RegisterClassTraitImplementation(st, trait, methods, implTrait.Span())
	}

	for _, impl := range a.Ast.ImplClasses {
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
		if !a.Project.StartsWithWorkspace(st.Def.Span().Source) {
			a.panicf(impl.Span(), "cannot add methods to types defined outside this package")
		}
		if st.Def.Attributes.Has("no_impl") {
			a.panicf(impl.Span(), "class `%s` cannot implement methods", st.Def.Name.Raw)
		}
		for _, method := range impl.Methods {
			funcTy := a.handleFunctionSignature(genericsScope, &method)
			funcTy.Scope = a.Scope
			funcTy.Generics = impl.Generics
			methodName := method.Name.Raw
			a.RegisterClassMethod(st, funcTy)
			impl.Checks = append(impl.Checks, func() {
				// this hack is needed, so something like `__x_iter_range` can check if `__x_iter_range_bound` exists or not
				a.checkClassMethods(st, methodName)
			})
		}
		impl.GenericsScope = genericsScope
	}
}

func (a *Analysis) analyzeImplementations() {
	for _, implTrait := range a.Ast.ImplTraits {
		for _, check := range implTrait.Checks {
			check()
		}
	}

	for _, let := range a.Ast.Lets {
		a.handleLet(a.Scope, let)
	}

	for _, funcDef := range a.Ast.Funcs {
		a.handleFunction(a.Scope, funcDef)
	}

	for _, impl := range a.Ast.ImplClasses {
		for _, check := range impl.Checks {
			check()
		}
		for _, method := range impl.Methods {
			_ = a.handleFunction(impl.GenericsScope.(*Scope), &method)
		}
	}

	for _, traitDef := range a.Ast.Traits {
		for _, check := range traitDef.Checks {
			check()
		}
	}

	a.CheckConflictingMethodImplementations()
	a.CheckConflictingTraitImplementations()
}
