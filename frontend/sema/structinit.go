package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleStructInit(scope *Scope, si *ast.ExprStructInit) Type {
	baseTy := a.resolvePathType(scope, &si.Name)
	if baseTy.Kind() != ast.SemStructKind {
		a.panic(si.Name.Span(), fmt.Sprintf("expected struct type for `%s`, found `%s`", si.Name.String(), baseTy.String()))
	}
	baseStruct := baseTy.Struct()
	expected := len(baseStruct.Generics.Params)
	provided := len(si.Generics)

	if a.SetStructSetupSpan(si.Span()) {
		defer a.ClearStructSetupSpan()
	}

	// If user provided generics but struct is not generic:
	switch {
	case expected == 0 && provided > 0:
		a.panic(si.Name.Span(), fmt.Sprintf("struct `%s` is not generic but generics were provided", baseStruct.Def.Name.Raw))
	case provided > expected:
		a.panic(si.Name.Span(), fmt.Sprintf("struct `%s` expects %d generic argument(s), but %d provided", baseStruct.Def.Name.Raw, expected, provided))
	}

	// 1) If user explicitly gave generics, resolve them
	var explicitGenerics []Type
	if provided > 0 {
		if provided != expected {
			a.panic(si.Name.Span(), fmt.Sprintf("struct `%s` expects %d generic argument(s), but %d provided", baseStruct.Def.Name.Raw, expected, provided))
		}
		explicitGenerics = make([]Type, 0, provided)
		for _, tyAst := range si.Generics {
			explicitGenerics = append(explicitGenerics, a.resolveType(scope, tyAst))
		}
	}

	// 2) If no generics provided but struct has generics => infer
	var inferredGenerics []Type
	if provided == 0 && expected > 0 {
		var err error
		inferredGenerics, err = a.inferStructGenericsForInit(scope, baseStruct, si.Fields)
		if err != nil {
			a.panic(si.Span(), err.Error())
		}
	}

	// Combine final generics
	concrete := make([]Type, expected)
	if provided > 0 {
		copy(concrete, explicitGenerics)
	} else if expected > 0 {
		copy(concrete, inferredGenerics)
	}

	// Now instantiate
	newStruct := a.instantiateStruct(baseStruct.Def, concrete)

	// ensure all required fields are present
	providedFields := make(map[string]struct{}, len(si.Fields))
	for _, f := range si.Fields {
		providedFields[f.Name.Raw] = struct{}{}
	}
	for name := range newStruct.AllFields() {
		// if _, ok := providedFields[name]; !ok && ty.Kind() != ast.SemOptionalKind {
		if _, ok := providedFields[name]; !ok {
			a.panic(si.Span(), fmt.Sprintf("missing required field `%s` in struct `%s` initialization", name, baseStruct.Def.Name.Raw))
		}
	}

	// type-check each provided field
	for i := range si.Fields {
		f := &si.Fields[i]
		field, ok := newStruct.GetField(f.Name.Raw)
		if !ok {
			a.panic(f.Name.Span(),
				fmt.Sprintf("struct `%s` has no field named `%s`",
					baseStruct.Def.Name.Raw, f.Name.Raw),
			)
		}
		if !a.canAccessStructMember(newStruct, field.IsPublic()) {
			a.Errorf(f.Name.Span(), "field `%s` of struct `%s` is private", f.Name.Raw, newStruct.Def.Name.Raw)
		}
		a.handleExpr(scope, &f.Value)
		exprTy := f.Value.Type()
		a.Matches(field.Ty, exprTy, f.Value.Span())
	}

	return ast.NewSemType(newStruct, si.Span())
}

func (a *Analysis) inferStructGenericsForInit(
	scope *Scope,
	baseStruct *SemStruct,
	fields []ast.ExprStructField,
) ([]Type, error) {
	expected := baseStruct.Generics.Len()
	placeholders := make(map[string]Type, expected)
	// unify declared field types with actual expression types
	for _, f := range fields {
		field, ok := baseStruct.Fields[f.Name.Raw]
		if !ok {
			// unknown field -> let handleStructInit panic
			continue
		}
		a.handleExpr(scope, &f.Value)
		exprTy := f.Value.Type()
		a.unify(field.Ty, exprTy, placeholders, f.Value.Span())
	}

	results := make([]Type, expected)
	for i, g := range baseStruct.Generics.Params {
		bound, ok := placeholders[g.Generic().Ident.Raw]
		if !ok {
			// If still unbound, raise an error
			return nil, fmt.Errorf(
				"could not infer generic `%s` for struct `%s`",
				g.Generic().Ident.Raw, baseStruct.Def.Name.Raw,
			)
		}
		results[i] = bound
	}
	return results, nil
}
