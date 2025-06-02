package sema

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
	protocol "github.com/gluax-lang/lsp"
)

// don't expose actual paths to the code generation :]
func (pa *ProjectAnalysis) StripWorkspace(path string) string {
	ws := pa.workspace
	ws, path = common.FilePathClean(ws), common.FilePathClean(path)
	return common.FilePathClean(filepath.Join(pa.CurrentPackage(), strings.TrimPrefix(path, ws)))
}

func (pa *ProjectAnalysis) StartsWithWorkspace(path string) bool {
	ws := common.FilePathClean(pa.workspace)
	path = common.FilePathClean(path)
	return strings.HasPrefix(path, ws)
}

type Analysis struct {
	Src                    string // source file name
	Workspace              string // workspace root
	Scope                  *Scope // root scope
	Diags                  []Diagnostic
	InlayHints             []InlayHint
	TempIdx                *int
	Project                *ProjectAnalysis
	Ast                    *ast.Ast
	SpanSymbols            map[Span]ast.Symbol // map of spans to symbols for hover and diagnostics
	State                  *State              // current state of the analysis
	currentStructSetupSpan *Span               // used to track the span of the current struct setup
}

func (a *Analysis) AddSpanSymbol(span Span, sym ast.Symbol) {
	if a.SpanSymbols == nil {
		a.SpanSymbols = make(map[Span]ast.Symbol)
	}
	if _, ok := a.SpanSymbols[span]; ok {
		return
	}
	a.SpanSymbols[span] = sym
}

func (a *Analysis) SetStructSetupSpan(span Span) bool {
	if a.currentStructSetupSpan == nil {
		a.currentStructSetupSpan = &span
		return true
	}
	return false
}

func (a *Analysis) ClearStructSetupSpan() {
	a.currentStructSetupSpan = nil
}

func (a *Analysis) GetStructSetupSpan(def Span) Span {
	if a.currentStructSetupSpan == nil {
		return def
	}
	return *a.currentStructSetupSpan
}

func (a *Analysis) handleAst(ast *ast.Ast) {
	a.Ast = ast
	a.handleItems(ast.Items)
}

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
			case *ast.Import, *ast.Use, *ast.ImplStruct:
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
			a.AddValue(a.Scope, it.Name.Raw, ast.NewValue(funcSem), it.Name.Span())
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
			for _, method := range it.Methods {
				funcTy := a.handleFunctionSignature(genericsScope, &method)
				funcTy.ImplStruct = it
				if err := a.State.AddStructMethod(st.Def, method.Name.Raw, funcTy, st.Generics.Params); err != nil {
					a.Error(err.Error(), method.Name.Span())
				}
				st.Methods[method.Name.Raw] = funcTy
			}
			it.GenericsScope = genericsScope
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
		}
	}
}

func (a *Analysis) Error(msg string, span Span) {
	a.Diags = append(a.Diags, *common.ErrorDiag(msg, span))
}

func (a *Analysis) Warning(msg string, span Span) {
	a.Diags = append(a.Diags, *common.WarningDiag(msg, span))
}

func (a *Analysis) Panic(msg string, span Span) {
	a.Error(msg, span)
	panic("")
}

func (a *Analysis) AddType(scope *Scope, name string, ty Type) {
	if err := scope.AddType(name, ty); err != nil {
		a.Error(err.Error(), ty.Span())
	}
}

func (a *Analysis) AddTypeVisibility(scope *Scope, name string, ty Type, public bool) {
	if err := scope.AddTypeVisibility(name, ty, public); err != nil {
		a.Error(err.Error(), ty.Span())
	}
}

func (a *Analysis) AddValue(scope *Scope, name string, val Value, span Span) {
	if err := scope.AddValue(name, val, span); err != nil {
		a.Error(err.Error(), span)
	}
}

func (a *Analysis) AddValueVisibility(scope *Scope, name string, val Value, span Span, public bool) {
	if err := scope.AddValueVisibility(name, val, span, public); err != nil {
		a.Error(err.Error(), span)
	}
}

func (a *Analysis) AddLabel(scope *Scope, label *ast.Ident) {
	if err := scope.AddLabel(label.Raw); err != nil {
		a.Error(err.Error(), label.Span())
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
		a.Panic(fmt.Sprintf("unknown type: %s", name), common.SpanDefault())
	}
	if ty.Kind() != ast.SemStructKind {
		a.Panic(fmt.Sprintf("expected struct type, got: %s", ty.Kind()), common.SpanDefault())
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

func (a *Analysis) vecType(t Type, span Span) Type {
	vec := a.getBuiltinType("vec")
	st := vec.Struct()
	newSt := a.instantiateStruct(st.Def, []Type{t})
	return ast.NewSemType(newSt, span)
}

func (a *Analysis) mapType(key, value Type, span Span) Type {
	mapTy := a.getBuiltinType("map")
	st := mapTy.Struct()
	newSt := a.instantiateStruct(st.Def, []Type{key, value})
	return ast.NewSemType(newSt, span)
}

func (a *Analysis) optionType(t Type, span Span) Type {
	option := a.getBuiltinType("option")
	st := option.Struct()
	newSt := a.instantiateStruct(st.Def, []Type{t})
	return ast.NewSemType(newSt, span)
}

func (a *Analysis) Matches(ty, other Type, span Span) {
	if !ty.Matches(other) {
		a.Error(fmt.Sprintf("mismatched types, expected `%s`, got `%s`", ty.String(), other.String()), span)
	}
}

func (a *Analysis) StrictMatches(ty, other Type, span Span) {
	if !ty.StrictMatches(other) {
		a.Error(fmt.Sprintf("mismatched types, expected `%s`, got `%s`", ty.String(), other.String()), span)
	}
}

func (a *Analysis) MatchesPanic(ty, other Type, span Span) {
	if !ty.Matches(other) {
		a.Panic(fmt.Sprintf("mismatched types, expected `%s`, got `%s`", ty.String(), other.String()), span)
	}
}
