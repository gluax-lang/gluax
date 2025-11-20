package sema

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
	protocol "github.com/gluax-lang/lsp"
)

// don't expose actual paths to the code generation :]
func (pa *ProjectAnalysis) PathRelativeToWorkspace(path string) string {
	ws := pa.Workspace()
	ws, path = common.FilePathClean(ws), common.FilePathClean(path)
	return common.FilePathClean(filepath.Join(pa.CurrentPackage(), strings.TrimPrefix(path, ws)))
}

func (pa *ProjectAnalysis) StartsWithWorkspace(path string) bool {
	ws := common.FilePathClean(pa.Workspace())
	path = common.FilePathClean(path)
	return strings.HasPrefix(path, ws)
}

func (pa *ProjectAnalysis) StripWorkspace(path string) string {
	ws := common.FilePathClean(pa.Workspace()) + "/"
	path = common.FilePathClean(path)
	rel := strings.TrimPrefix(path, ws)
	return rel
}

type Analysis struct {
	Src        string // source file name
	Workspace  string // workspace root
	Scope      *Scope // root scope
	Diags      []Diagnostic
	InlayHints []InlayHint
	Project    *ProjectAnalysis
	Ast        *ast.Ast
	State      *State // current state of the analysis
	Exprs      []*ast.Expr
}

func (a *Analysis) Copy() *Analysis {
	return &Analysis{
		Src:        a.Src,
		Workspace:  a.Workspace,
		Scope:      &Scope{},
		Diags:      []Diagnostic{},
		InlayHints: []InlayHint{},
		Project:    &ProjectAnalysis{},
		Ast:        &ast.Ast{},
		State:      &State{},
		Exprs:      []*ast.Expr{},
	}
}

func (a *Analysis) RemoveRootDirFromPath(path string) string {
	ws := common.FilePathClean(a.Workspace)
	path = common.FilePathClean(path)

	// last folder name of workspace
	base := filepath.Base(ws)

	// ensure trailing slash for prefix trim
	wsSlash := ws + "/"

	rel := strings.TrimPrefix(path, wsSlash)

	// join with /
	if rel == "" {
		return base
	}
	return base + "/" + rel
}

func (a *Analysis) LocationFromSpan(span Span) string {
	relPath := a.RemoveRootDirFromPath(span.Source)
	return fmt.Sprintf("--[[%s:%d:%d]]", relPath, span.LineStart+1, span.ColumnStart+1)
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

func (a *Analysis) AddValue(scope *Scope, name string, val *Value, span Span) {
	if err := scope.AddValue(name, val, span); err != nil {
		a.Errorf(span, "%s", err.Error())
	}
}

func (a *Analysis) AddValueVisibility(scope *Scope, name string, val *Value, span Span, public bool) {
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
			Line:      span.LineStart,
			Character: span.ColumnEndUTF16,
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

func (a *Analysis) unionType(span Span, types ...Type) Type {
	if len(types) == 0 {
		panic("unionType called with no types")
	}
	unionT := &SemUnion{Types: types, Span_: span}
	return ast.NewSemType(unionT, span)
}

func (a *Analysis) nilableType(base Type, span Span) Type {
	return a.unionType(span, a.nilType(), base)
}

func (a *Analysis) tupleType(span Span, other ...Type) Type {
	if len(other) == 0 {
		panic("tupleType called with no types")
	}
	return ast.NewSemType(ast.SemTuple{Elems: other}, span)
}

func (a *Analysis) functionType(name string, params []Type, returnType Type, span Span) Type {
	ident := lexer.NewTokIdent(name, span)
	funcT := &SemFunction{
		Def: ast.Function{
			Name:  &ident,
			Span_: span,
		},
		Params: params,
		Return: returnType,
	}
	return ast.NewSemType(funcT, span)
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

	for _, stDef := range astD.Classes {
		stDef.Scope = a.Scope
		st := a.setupClass(stDef)
		stSem := ast.NewSemType(st, stDef.Name.Span())
		a.AddTypeVisibility(a.Scope, stDef.Name.Raw, stSem, stDef.Public)
		a.AddDecl(stSem)
	}

	for _, funcDef := range astD.Funcs {
		fun := &SemFunction{}
		val := ast.NewValue(fun)
		a.AddValueVisibility(a.Scope, funcDef.Name.Raw, val, funcDef.Name.Span(), funcDef.Public)
	}

	for _, letDef := range astD.Lets {
		for i, ident := range letDef.Names {
			if ident.Raw == "_" {
				a.panic(ident.Span(), "cannot use `_` in top level let binding")
			}
			variable := ast.NewVariable(letDef, i, a.anyType())
			val := ast.NewValue(variable)
			a.AddValueVisibility(a.Scope, ident.Raw, val, ident.Span(), letDef.Public)
		}
	}
}

func (a *Analysis) resolveUses() {
	for _, use := range a.Ast.Uses {
		a.handleUse(a.Scope, use)
	}
}

func (a *Analysis) resolveImplementations() {
	for _, funcDef := range a.Ast.Funcs {
		funcSem := a.handleFunctionSignature(a.Scope, funcDef)
		funcDef.SetSem(funcSem)
		sym := a.Scope.GetSymbol(funcDef.Name.Raw)
		if sym == nil {
			continue
		}
		sym.SetData(ast.NewValue(funcSem))
		a.AddDecl(funcSem)
	}

	for _, letDef := range a.Ast.Lets {
		for i, ident := range letDef.Names {
			ty := a.resolveType(a.Scope, *letDef.Types[i])
			variable := ast.NewVariable(letDef, i, ty)
			sym := a.Scope.GetSymbol(ident.Raw)
			sym.SetData(ast.NewValue(variable))
		}
	}

	for _, stDef := range a.Ast.Classes {
		superDef := stDef.Super
		if superDef == nil {
			continue
		}
		st := a.GetClass(stDef)
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
		st := a.GetClass(stDef)
		a.collectClassFields(st)

		for _, field := range st.Fields {
			a.AddDecl(field)
		}
	}

	for _, impl := range a.Ast.ImplClasses {
		impl.Scope = a.Scope
		stTy := a.resolveType(a.Scope, impl.Class)
		if !stTy.IsClass() {
			a.panicf(impl.Class.Span(), "expected class type, got: %s", stTy.String())
		}
		st := stTy.Class()
		if !a.Project.StartsWithWorkspace(st.Def.Span().Source) {
			a.panicf(impl.Span(), "cannot add methods to types defined outside this package")
		}
		if st.Def.Attributes.Has("no_impl") {
			a.panicf(impl.Span(), "class `%s` cannot implement methods", st.Def.Name.Raw)
		}
		impl.ClassSema = st
		for _, method := range impl.Methods {
			var funcTy *ast.SemFunction
			if method.IsStatic() {
				funcTy = a.handleFunctionSignature(a.Scope, method)
			} else {
				method.Params[0].Type = impl.Class
				funcTy = a.handleFunctionSignature(a.Scope, method)
			}
			funcTy.Class = st
			methodName := method.Name.Raw
			if _, ok := st.Methods[methodName]; ok {
				a.Errorf(method.Span(), "duplicate method impl `%s` for class `%s`", methodName, st.Def.Name.Raw)
				continue
			}
			st.Methods[methodName] = funcTy
			impl.Checks = append(impl.Checks, func() {
				// this hack is needed, so something like `__x_iter_range` can check if `__x_iter_range_bound` exists or not
				a.checkClassMethods(st, methodName)

				if st.Super == nil {
					return
				}

				superMethod := st.Super.GetMethod(methodName, true)
				if superMethod == nil || superMethod.IsStatic() {
					return
				}

				if funcTy.IsStatic() {
					a.Errorf(
						funcTy.Span(),
						"method `%s` does not match superclass `%s` signature",
						methodName,
						superMethod.Class.String(),
					)
					return
				}

				superMethodCopy := *superMethod
				superMethodCopy.Params = superMethodCopy.Params[1:] // remove `self` param
				funcTyCopy := *funcTy
				funcTyCopy.Params = funcTyCopy.Params[1:] // remove `self` param

				if !a.matchFunction(&superMethodCopy, &funcTyCopy) {
					a.Errorf(
						funcTy.Span(),
						"method `%s` does not match superclass `%s` signature",
						methodName,
						superMethod.Class.String(),
					)
				}
			})
		}
	}
}

func (a *Analysis) analyzeImplementations() {
	for _, let := range a.Ast.Lets {
		a.handleLet(a.Scope, let)
	}

	for _, f := range a.Ast.Funcs {
		if f.Body == nil {
			if !f.IsGlobal() {
				a.Error(f.Span(), "function must have a body")
			}
		} else {
			if f.IsGlobal() {
				a.Error(f.Span(), "function cannot have a body")
			}
		}
		a.handleFunction(a.Scope, f)
	}

	for _, impl := range a.Ast.ImplClasses {
		for _, check := range impl.Checks {
			check()
		}
		if impl.Scope == nil {
			continue
		}
		for _, method := range impl.Methods {
			if method.Body == nil {
				// "must have a body" if the method is non-global and the class is
				// either non-global or unknown (ClassSema == nil)
				if !method.IsGlobal() && (impl.ClassSema == nil || !impl.ClassSema.IsGlobal()) {
					a.Error(method.Span(), "must have a body")
				}
			} else {
				if method.IsGlobal() {
					a.Error(method.Span(), "cannot have a body")
				}

				// Only do class-global checks when ClassSema is available
				if impl.ClassSema != nil && impl.ClassSema.IsGlobal() {
					if !method.IsStatic() && !method.Attributes.Has("local_method") {
						a.Error(
							method.Span(),
							"cannot have a body, because class is global (use `local_method` attribute to allow this)",
						)
					}
				}
			}

			if method.IsStatic() {
				_ = a.handleFunction(impl.Scope.(*Scope), method)
			} else {
				method.Params[0].Type = impl.Class
				_ = a.handleFunction(impl.Scope.(*Scope), method)
			}
		}
	}

	if a.Project.Main == a.Src && !a.Project.Config.Lib {
		// check that `main` function exists in the main file
		mainFuncValue := a.Scope.GetValue("main")
		if mainFuncValue == nil {
			a.panic(common.SpanDefault(), "main function not found")
		}
		if !mainFuncValue.IsFunction() {
			a.panic(mainFuncValue.Span(), "`main` is not a function")
		}
		// check that main has no parameters and return type is `nil`
		mainFunc := mainFuncValue.Function()
		if len(mainFunc.Params) != 0 {
			a.panicf(mainFunc.Span(), "`main` function must not have parameters")
		}
		returnType := mainFunc.Return
		if !a.MatchTypesStrict(a.nilType(), returnType) {
			a.panicf(mainFunc.Span(), "`main` function return type must be `nil`, got `%s`", returnType.String())
		}

		a.State.MainFunc = mainFunc
	}
}
