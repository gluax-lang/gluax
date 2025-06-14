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
	SpanSymbols           map[Span]ast.Symbol // map of spans to symbols for hover and diagnostics
	State                 *State              // current state of the analysis
	currentClassSetupSpan *Span               // used to track the span of the current class setup
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

func (a *Analysis) handleAst(ast *ast.Ast) {
	a.Ast = ast
	a.handleItems(ast)
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

func (a *Analysis) vecType(t Type, span Span) Type {
	if a.SetClassSetupSpan(span) {
		defer a.ClearClassSetupSpan()
	}
	vec := a.getBuiltinType("vec")
	st := vec.Class()
	newSt := a.instantiateClass(st.Def, []Type{t})
	return ast.NewSemType(newSt, span)
}

func (a *Analysis) mapType(key, value Type, span Span) Type {
	if a.SetClassSetupSpan(span) {
		defer a.ClearClassSetupSpan()
	}
	mapTy := a.getBuiltinType("map")
	st := mapTy.Class()
	newSt := a.instantiateClass(st.Def, []Type{key, value})
	return ast.NewSemType(newSt, span)
}

func (a *Analysis) optionType(t Type, span Span) Type {
	option := a.getBuiltinType("option")
	st := option.Class()
	newSt := a.instantiateClass(st.Def, []Type{t})
	return ast.NewSemType(newSt, span)
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
