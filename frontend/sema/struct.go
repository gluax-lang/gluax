package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) setupStruct(def *ast.Struct, concrete []Type) *SemStruct {
	stScope := a.Scope.Child(false)
	st := ast.NewSemStruct(def)
	st.Scope = stScope
	if def.GetFromStack(concrete) == nil {
		def.AddToStack(st, concrete)
	}
	a.buildGenericsTable(stScope, st, concrete)
	return st
}

func (a *Analysis) buildGenericsTable(scope *Scope, st *SemStruct, concrete []Type) {
	params := make([]Type, 0, len(st.Def.Generics.Params))
	for i, g := range st.Def.Generics.Params {
		var binding, param Type
		if concrete == nil {
			binding = ast.NewSemGenericType(g.Name, true)
			param = ast.NewSemGenericType(g.Name, false)
		} else {
			binding = concrete[i]
			param = binding
		}
		a.AddType(scope, g.Name.Raw, binding)
		params = append(params, param)
	}
	st.Generics.Params = params
}

func (a *Analysis) collectStructFields(st *SemStruct) {
	for _, field := range st.Def.Fields {
		if _, ok := st.Fields[field.Name.Raw]; ok {
			a.Error("duplicate field name", field.Name.Span())
		}
		stScope := st.Scope.(*Scope)
		ty := a.resolveType(stScope, field.Type)
		st.Fields[field.Name.Raw] = ast.NewSemStructField(field, ty)
	}
}

func (a *Analysis) collectStructMethods(st *SemStruct, withBody bool) {
	for _, method := range st.Def.Methods {
		if _, ok := st.Methods[method.Name.Raw]; ok {
			a.Error("duplicate method name", method.Name.Span())
		}
		stScope := st.Scope.(*Scope)
		funcTy := a.handleFunctionImpl(stScope, &method, withBody)
		funcTy.OwnerStruct = st
		st.Methods[method.Name.Raw] = funcTy
	}
}

func (a *Analysis) instantiateStruct(def *ast.Struct, concrete []Type, withBody bool) *SemStruct {
	if st := def.GetFromStack(concrete); st != nil {
		return st
	}

	if len(concrete) != def.Generics.Len() {
		a.Panic(
			fmt.Sprintf("struct `%s` expects %d generic argument(s), but %d provided",
				def.Name.Raw, def.Generics.Len(), len(concrete)),
			def.Span(),
		)
	}

	st := a.setupStruct(def, concrete)

	stScope := st.Scope.(*Scope)
	stScope.ForceAddType("Self", ast.NewSemType(st, def.Span()))
	a.collectStructFields(st)
	a.collectStructMethods(st, withBody)

	return st
}

func (a *Analysis) unify(
	base Type,
	actual Type,
	placeholders map[string]Type,
	span Span,
) Type {
	if disallowedKind(base) || disallowedKind(actual) {
		a.Panic("type cannot be used here", span)
	}

	// If base is already bound to something in placeholders, unify that again:
	if base.IsGeneric() {
		if actual.IsTuple() || actual.IsVararg() {
			return base
		}

		gname := base.Generic().Ident.Raw
		// If we've already bound T => Something, unify that "Something" with actual
		if existing, ok := placeholders[gname]; ok {
			return a.unify(existing, actual, placeholders, span)
		}

		// If 'actual' is also a generic that was previously bound, unify that.
		if actual.IsGeneric() {
			otherG := actual.Generic().Ident.Raw
			if existingOther, ok := placeholders[otherG]; ok {
				// unify base => existingOther
				return a.unify(base, existingOther, placeholders, span)
			}
			placeholders[gname] = ast.NewSemType(
				ast.SemGenericType{
					Ident: actual.Generic().Ident,
					Bound: true, // force "Bound" = true
				},
				actual.Span(),
			)
		}

		// If no existing binding, bind T => actual and return the actual type.
		// But first, check if actual is also an unbound generic => pick whichever name
		// we prefer. For simplicity, just bind base => actual.
		placeholders[gname] = actual
		return actual
	}

	// If base is a struct => unify generics param-by-param
	if base.IsStruct() {
		bs := base.Struct()
		// actual must be a struct
		if !actual.IsStruct() {
			a.Panic(
				fmt.Sprintf("type mismatch: expected struct `%s`, got `%s`",
					bs.String(), actual.String()),
				span,
			)
		}
		as := actual.Struct()

		// same def
		if bs.Def != as.Def {
			a.Panic(
				fmt.Sprintf("type mismatch: expected struct `%s`, got `%s`",
					bs.Def.Name.Raw, as.Def.Name.Raw),
				span,
			)
		}
		// unify each generic param
		if len(bs.Generics.Params) != len(as.Generics.Params) {
			a.Panic(
				fmt.Sprintf(
					"struct `%s` has %d generic param(s), but got %d in `%s`",
					bs.Def.Name.Raw,
					len(bs.Generics.Params),
					len(as.Generics.Params),
					as.Def.Name.Raw),
				span,
			)
		}
		newParams := make([]Type, len(bs.Generics.Params))
		for i := range bs.Generics.Params {
			// unify param i
			pbase := bs.Generics.Params[i]
			pact := as.Generics.Params[i]
			specialized := a.unify(pbase, pact, placeholders, span)
			newParams[i] = specialized
		}
		// after unifying all generics, reconstruct the struct type with the specialized generics
		specializedStruct := ast.NewSemStruct(bs.Def)
		specializedStruct.Generics = ast.SemGenerics{Params: newParams}
		// We won't re-check fields immediately here;
		// a.instantiateStruct does that anyway
		// Return a new Type with that specialized struct.
		return ast.NewSemType(specializedStruct, base.Span())
	}

	if base.IsUnreachable() {
		return base
	}

	if base.IsAny() {
		return base
	}

	// For everything else (e.g. base is a string/number/bool/nil literal struct),
	// just see if they strictly match. If not, panic.
	if !base.StrictMatches(actual) {
		a.Panic(
			fmt.Sprintf("mismatched types: expected `%s`, got `%s`",
				base.String(), actual.String()),
			span,
		)
	}
	// If we get here, base == actual. Return base.
	return base
}

func (a *Analysis) canAccessStructMember(st *SemStruct, memberPublic bool) bool {
	if memberPublic {
		return true
	}
	source := st.Def.Span().Source
	// Private members are only accessible from the same source file
	return a.Src == source
}
