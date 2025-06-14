package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleClassInit(scope *Scope, si *ast.ExprClassInit) Type {
	baseTy := a.resolvePathType(scope, &si.Name)
	if baseTy.Kind() != ast.SemClassKind {
		a.panic(si.Name.Span(), fmt.Sprintf("expected class type for `%s`, found `%s`", si.Name.String(), baseTy.String()))
	}
	baseClass := baseTy.Class()
	expected := len(baseClass.Generics.Params)
	provided := len(si.Generics)

	if a.SetClassSetupSpan(si.Span()) {
		defer a.ClearClassSetupSpan()
	}

	// If user provided generics but class is not generic:
	switch {
	case expected == 0 && provided > 0:
		a.panic(si.Name.Span(), fmt.Sprintf("class `%s` is not generic but generics were provided", baseClass.Def.Name.Raw))
	case provided > expected:
		a.panic(si.Name.Span(), fmt.Sprintf("class `%s` expects %d generic argument(s), but %d provided", baseClass.Def.Name.Raw, expected, provided))
	}

	// 1) If user explicitly gave generics, resolve them
	var explicitGenerics []Type
	if provided > 0 {
		if provided != expected {
			a.panic(si.Name.Span(), fmt.Sprintf("class `%s` expects %d generic argument(s), but %d provided", baseClass.Def.Name.Raw, expected, provided))
		}
		explicitGenerics = make([]Type, 0, provided)
		for _, tyAst := range si.Generics {
			explicitGenerics = append(explicitGenerics, a.resolveType(scope, tyAst))
		}
	}

	// 2) If no generics provided but class has generics => infer
	var inferredGenerics []Type
	if provided == 0 && expected > 0 {
		var err error
		inferredGenerics, err = a.inferClassGenericsForInit(scope, baseClass, si.Fields)
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
	newClass := a.instantiateClass(baseClass.Def, concrete)

	// ensure all required fields are present
	providedFields := make(map[string]struct{}, len(si.Fields))
	for _, f := range si.Fields {
		providedFields[f.Name.Raw] = struct{}{}
	}
	for name := range newClass.AllFields() {
		// if _, ok := providedFields[name]; !ok && ty.Kind() != ast.SemOptionalKind {
		if _, ok := providedFields[name]; !ok {
			a.panic(si.Span(), fmt.Sprintf("missing required field `%s` in class `%s` initialization", name, baseClass.Def.Name.Raw))
		}
	}

	// type-check each provided field
	for i := range si.Fields {
		f := &si.Fields[i]
		field, ok := newClass.GetField(f.Name.Raw)
		if !ok {
			a.panic(f.Name.Span(),
				fmt.Sprintf("class `%s` has no field named `%s`",
					baseClass.Def.Name.Raw, f.Name.Raw),
			)
		}
		if !a.canAccessClassMember(newClass, field.IsPublic()) {
			a.Errorf(f.Name.Span(), "field `%s` of class `%s` is private", f.Name.Raw, newClass.Def.Name.Raw)
		}
		a.handleExpr(scope, &f.Value)
		exprTy := f.Value.Type()
		a.Matches(field.Ty, exprTy, f.Value.Span())
	}

	return ast.NewSemType(newClass, si.Span())
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
