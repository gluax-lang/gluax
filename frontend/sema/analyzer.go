package sema

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	protocol "github.com/gluax-lang/lsp"
)

// don't expose actual paths to the code generation :]
func (pa *ProjectAnalysis) StripWorkspace(path string) string {
	ws := pa.workspace
	ws, path = common.FilePathClean(ws), common.FilePathClean(path)
	return common.FilePathClean(filepath.Join(pa.currentPackage, strings.TrimPrefix(path, ws)))
}

type Analysis struct {
	Src         string // source file name
	Workspace   string // workspace root
	Scope       *Scope // root scope
	Diags       []Diagnostic
	InlayHints  []InlayHint
	TempIdx     *int
	Project     *ProjectAnalysis
	Ast         *ast.Ast
	SpanSymbols map[Span]ast.Symbol // map of spans to symbols for hover and diagnostics
	// UseAliases map[string][]string
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

func (a *Analysis) IsStdTypes() bool {
	return a.Project.Config.Std && strings.HasSuffix(a.Src, "src/types.gluax")
}

func (a *Analysis) handleAst(ast *ast.Ast) {
	a.Ast = ast
	a.addItems(ast.Items)
	for _, item := range ast.Items {
		a.handleItem(a.Scope, item)
	}
}

func (a *Analysis) addItems(items []ast.Item) {
	// if a.Project.Config.Std {
	// 	for name, ty := range ast.StdBuiltinTypes {
	// 		a.Scope.ForceAddType(name, ty)
	// 	}
	// }

	// imports first
	if !a.IsStdTypes() {
		for _, item := range items {
			switch it := item.(type) {
			case *ast.Import:
				a.handleImport(a.Scope, it)
			}
		}
	}

	// struct names with their generics phase
	for _, item := range items {
		switch it := item.(type) {
		case *ast.Struct:
			oldScope := a.Scope
			if a.IsStdTypes() {
				a.Scope = a.Project.rootScope
			}
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
			if a.IsStdTypes() {
				ast.AddBuiltinType(it.Name.Raw, stSem)
				a.Scope = oldScope
			}
		}
	}

	if a.IsStdTypes() {
		for _, item := range items {
			switch it := item.(type) {
			case *ast.Import:
				a.handleImport(a.Scope, it)
			}
		}
	}

	// struct fields phase
	for _, item := range items {
		switch it := item.(type) {
		case *ast.Struct:
			st := it.GetFromStack(nil)
			stScope := st.Scope.(*Scope)
			SelfSt := stScope.GetType("Self").Struct()
			a.collectStructFields(SelfSt)
			a.collectStructFields(st)
		}
	}

	// struct methods phase
	for _, item := range items {
		switch it := item.(type) {
		case *ast.Struct:
			st := it.GetFromStack(nil)
			stScope := st.Scope.(*Scope)
			SelfSt := stScope.GetType("Self").Struct()
			a.collectStructMethods(SelfSt, false)
			a.collectStructMethods(st, true)
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
	// label = ": " + label
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
	if ty := ast.GetBuiltinType(name); ty != nil {
		return *ty
	}
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
