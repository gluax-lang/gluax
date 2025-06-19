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
	expected := len(baseClass.Generics.Params)

	// 2) If no generics provided but class has generics => infer
	var inferredGenerics []Type
	if expected > 0 && len(si.Name.LastSegment().Generics) == 0 {
		var err error
		inferredGenerics, err = a.inferClassGenericsForInit(scope, baseClass, si.Fields)
		if err != nil {
			a.panic(si.Span(), err.Error())
		}

		baseClass = a.instantiateClass(baseClass.Def, inferredGenerics)
	}

	// ensure all required fields are present
	providedFields := make(map[string]struct{}, len(si.Fields))
	for _, f := range si.Fields {
		providedFields[f.Name.Raw] = struct{}{}
	}
	for name := range baseClass.AllFields() {
		// if _, ok := providedFields[name]; !ok && ty.Kind() != ast.SemOptionalKind {
		if _, ok := providedFields[name]; !ok {
			a.panic(si.Span(), fmt.Sprintf("missing required field `%s` in class `%s` initialization", name, baseClass.Def.Name.Raw))
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
		if !a.canAccessClassField(baseClass, field.IsPublic()) {
			a.Errorf(f.Name.Span(), "field `%s` of class `%s` is private", f.Name.Raw, baseClass.Def.Name.Raw)
		}
		a.handleExpr(scope, &f.Value)
		exprTy := f.Value.Type()
		a.Matches(field.Ty, exprTy, f.Value.Span())
	}

	return ast.NewSemType(baseClass, si.Span())
}

func (a *Analysis) inferClassGenericsForInit(
	scope *Scope,
	baseClass *SemClass,
	fields []ast.ExprClassField,
) ([]Type, error) {
	expected := baseClass.Generics.Len()
	placeholders := make(map[string]Type, expected)
	// unify declared field types with actual expression types
	for _, f := range fields {
		field, ok := baseClass.Fields[f.Name.Raw]
		if !ok {
			// unknown field -> let handleClassInit panic
			continue
		}
		a.handleExpr(scope, &f.Value)
		exprTy := f.Value.Type()
		a.unify(field.Ty, exprTy, placeholders, f.Value.Span())
	}

	results := make([]Type, expected)
	for i, g := range baseClass.Generics.Params {
		bound, ok := placeholders[g.Generic().Ident.Raw]
		if !ok {
			// If still unbound, raise an error
			return nil, fmt.Errorf(
				"could not infer generic `%s` for class `%s`",
				g.Generic().Ident.Raw, baseClass.Def.Name.Raw,
			)
		}
		results[i] = bound
	}
	return results, nil
}
