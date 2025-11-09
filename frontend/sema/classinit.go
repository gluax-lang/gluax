package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleClassInit(scope *Scope, si *ast.ExprClassInit) Type {
	if a.SetClassSetupSpan(si.Span()) {
		defer a.ClearClassSetupSpan()
	}

	baseTy := a.resolvePathType(scope, &si.Name)
	if baseTy.Kind() != ast.SemClassKind {
		a.panic(si.Name.Span(), fmt.Sprintf("expected class type for `%s`, found `%s`", si.Name.String(), baseTy.String()))
	}
	baseClass := baseTy.Class()

	// ensure all required fields are present
	providedFields := make(map[string]struct{}, len(si.Fields))
	for _, f := range si.Fields {
		providedFields[f.Name.Raw] = struct{}{}
	}
	for name, field := range baseClass.AllFields() {
		// if _, ok := providedFields[name]; !ok && ty.Kind() != ast.SemOptionalKind {
		if _, ok := providedFields[name]; !ok {
			if !field.Ty.IsNilable() {
				a.panicf(si.Span(), "missing required field `%s` in class `%s` initialization", name, baseClass.Def.Name.Raw)
			}
		}
	}

	// type-check each provided field
	for i := range si.Fields {
		f := &si.Fields[i]
		field, ok := baseClass.GetField(f.Name.Raw)
		if !ok {
			a.panic(f.Name.Span(),
				fmt.Sprintf("class `%s` has no field named `%s`",
					baseClass.Def.Name.Raw, f.Name.Raw),
			)
		}
		a.AddRef(field, f.Name.Span())
		if !a.CanAccessClassField(baseClass, field.IsPublic()) {
			a.Errorf(f.Name.Span(), "field `%s` of class `%s` is private", f.Name.Raw, baseClass.Def.Name.Raw)
		}
		a.handleExpr(scope, &f.Value)
		exprTy := f.Value.Type()
		a.Matches(field.Ty, exprTy, f.Value.Span())
	}

	return ast.NewSemType(baseClass, si.Span())
}
