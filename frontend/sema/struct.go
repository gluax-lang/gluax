package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) setupStruct(def *ast.Struct, concrete []Type) *SemStruct {
	for _, ty := range concrete {
		if !isInnerTypeRuleCompliant(ty) {
			a.Panic(
				fmt.Sprintf("type `%s` cannot be used as a generic type", ty.String()),
				a.GetStructSetupSpan(def.Span()),
			)
		}
	}
	stScope := def.Scope.(*Scope).Child(false)
	st := ast.NewSemStruct(def)
	st.Scope = stScope
	if a.State.GetStruct(def, concrete) == nil {
		a.State.AddStruct(def, st, concrete)
	}
	a.buildGenericsTable(stScope, st, concrete)
	return st
}

func (a *Analysis) HandleStructMethod(st *ast.SemStruct, method ast.SemFunction, withBody bool) ast.SemFunction {
	impl := method.ImplStruct
	genericsScope := NewScope(impl.Scope.(*Scope))
	for i, g := range impl.Generics.Params {
		stGTy := st.Generics.Params[i]
		a.AddType(genericsScope, g.Name.Raw, stGTy)
	}
	{
		stTy := ast.NewSemType(st, st.Def.Name.Span())
		if err := genericsScope.AddType("Self", stTy); err != nil {
			a.Error(err.Error(), st.Def.Name.Span())
		}
	}
	var funcTy ast.SemFunction
	if withBody {
		funcTy = a.handleFunction(genericsScope, &method.Def)
	} else {
		funcTy = a.handleFunctionSignature(genericsScope, &method.Def)
	}
	funcTy.Struct = st
	funcTy.ImplStruct = impl
	st.Methods[method.Def.Name.Raw] = funcTy
	return funcTy
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

func (a *Analysis) instantiateStruct(def *ast.Struct, concrete []Type) *SemStruct {
	if st := a.State.GetStruct(def, concrete); st != nil {
		return st
	}

	if len(concrete) != def.Generics.Len() {
		a.Panic(
			fmt.Sprintf("struct `%s` expects %d generic argument(s), but %d provided",
				def.Name.Raw, def.Generics.Len(), len(concrete)),
			a.GetStructSetupSpan(def.Span()),
		)
	}

	st := a.setupStruct(def, concrete)

	stScope := st.Scope.(*Scope)
	stScope.ForceAddType("Self", ast.NewSemType(st, def.Span()))
	a.collectStructFields(st)

	return st
}

func (a *Analysis) resolveStruct(scope *Scope, st *ast.SemStruct, generics []ast.Type, span Span) *ast.SemStruct {
	if len(generics) == 0 {
		if !st.Def.Generics.IsEmpty() {
			if st.Generics.UnboundCount() == st.Generics.Len() {
				a.Panic(fmt.Sprintf(
					"struct `%s` is generic but no generic arguments were provided",
					st.Def.Name.Raw,
				), span)
			}
		}
		return st
	}

	if st.Def.Generics.IsEmpty() {
		a.Panic(fmt.Sprintf("struct `%s` is not generic but generics were provided", st.Def.Name.Raw), span)
	}

	if len(generics) != st.Def.Generics.Len() {
		a.Panic(fmt.Sprintf("expected %d generics, got %d", st.Def.Generics.Len(), len(generics)), span)
	}

	concrete := make([]Type, 0, len(generics))
	for _, g := range generics {
		concrete = append(concrete, a.resolveType(scope, g))
	}

	st = a.instantiateStruct(st.Def, concrete)

	return st
}

var getImplType = func(sI StructInstance, idx int) (Type, bool) {
	if idx < 0 || idx >= len(sI.Args) {
		return Type{}, false
	}
	ty := sI.Args[idx]
	if ty.IsGeneric() {
		return Type{}, false
	}
	return ty, true
}

func (a *Analysis) addStructMethod(st *ast.SemStruct, method ast.SemFunction) {
	methodName := method.Def.Name.Raw
	if _, exists := a.getStructMethod(st, methodName); exists {
		a.Error(fmt.Sprintf("method '%s' already exists for these concrete types", methodName), method.Def.Name.Span())
		return
	}
	st.Methods[methodName] = method
}

func (a *Analysis) getStructMethod(st *ast.SemStruct, methodName string) (ast.SemFunction, bool) {
	if method, ok := st.Methods[methodName]; ok {
		return method, true
	}
	stack := a.State.GetStructStack(st.Def)
	for _, inst := range stack {
		if method, ok := inst.Type.Methods[methodName]; ok {
			this := true
			for i, t := range st.Generics.Params {
				ty, ok := getImplType(inst, i)
				if !ok {
					continue
				}
				if !t.StrictMatches(ty) {
					this = false
					break
				}
			}
			if this {
				method = a.HandleStructMethod(st, method, false) // handle it without body
				return method, true
			}
		}
	}
	return ast.SemFunction{}, false
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
