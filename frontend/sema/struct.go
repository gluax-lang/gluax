package sema

import (
	"fmt"
	"maps"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) setupTypeGenerics(scope *Scope, generics ast.Generics, concrete []Type) *Scope {
	scope = NewScope(scope)
	for i, g := range generics.Params {
		var binding Type
		if concrete == nil {
			binding = ast.NewSemGenericType(g.Name, true)
		} else {
			binding = concrete[i]
		}
		a.AddType(scope, g.Name.Raw, binding)
	}
	return scope
}

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
	if a.GetStruct(def, concrete) == nil {
		def.AddStruct(st, concrete)
	}
	a.buildGenericsTable(stScope, st, concrete)
	return st
}

func (a *Analysis) HandleStructMethod(st *ast.SemStruct, method ast.SemFunction, withBody bool) ast.SemFunction {
	genericsScope := a.setupTypeGenerics(method.Scope.(*Scope), method.Generics, st.Generics.Params)
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
	funcTy.Generics = method.Generics
	funcTy.Scope = method.Scope
	funcTy.Struct = st
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
	if st := a.GetStruct(def, concrete); st != nil {
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

// Helper function to check if an impl pattern matches a concrete struct
func implMatchesStruct(a *Analysis, inst StructInstance, st *ast.SemStruct) bool {
	if len(inst.Args) != len(st.Generics.Params) {
		return false
	}

	// Build a mapping from impl's generic parameter names to types
	genericBindings := make(map[string]Type)

	for i, implType := range inst.Args {
		targetType := st.Generics.Params[i]

		if implType.IsGeneric() {
			// impl has a generic parameter at this position
			genericName := implType.Generic().Ident.Raw
			if existingBinding, exists := genericBindings[genericName]; exists {
				// This generic was already bound to a type, check consistency
				if targetType.IsGeneric() {
					// Both are generic - they can potentially match
					continue
				}
				if !a.matchTypesStrict(existingBinding, targetType) {
					return false
				}
			} else {
				// First time seeing this generic, bind it
				genericBindings[genericName] = targetType
			}
		} else {
			// impl has a concrete type at this position
			if targetType.IsGeneric() {
				// impl is concrete, target is generic - no match
				return false
			}
			// both concrete - must match exactly
			if !a.matchTypesStrict(implType, targetType) {
				return false
			}
		}
	}

	return true
}

func findInStructStack[T any](
	a *Analysis,
	st *ast.SemStruct,
	getItem func(inst StructInstance) (T, bool),
	match func(item T) bool,
) (T, bool) {
	stack := st.Def.GetStructStack()
	for _, inst := range stack {
		item, ok := getItem(inst)
		if !ok {
			continue
		}
		if implMatchesStruct(a, inst, st) && match(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

func (a *Analysis) addStructMethod(st *ast.SemStruct, method ast.SemFunction) {
	methodName := method.Def.Name.Raw
	if _, exists := a.GetStructMethod(st, methodName); exists {
		a.Error(fmt.Sprintf("method '%s' already exists for these concrete types", methodName), method.Def.Name.Span())
		return
	}
	method.Struct = st
	st.Methods[methodName] = method
}

func (a *Analysis) GetStructMethod(st *ast.SemStruct, methodName string) (ast.SemFunction, bool) {
	if method, ok := st.Methods[methodName]; ok {
		return method, true
	}
	getMethod := func(inst StructInstance) (ast.SemFunction, bool) {
		m, ok := inst.Type.Methods[methodName]
		return m, ok
	}
	match := func(_ ast.SemFunction) bool { return true }
	if method, ok := findInStructStack(a, st, getMethod, match); ok {
		method = a.HandleStructMethod(st, method, false)
		return method, true
	}
	return ast.SemFunction{}, false
}

func (a *Analysis) addStructTrait(st *ast.SemStruct, trait *ast.SemTrait, span Span) {
	if a.structHasTrait(st, trait) {
		a.Error(fmt.Sprintf("trait `%s` already exists for this struct", trait.Def.Name.Raw), span)
		return
	}
	st.Traits[trait] = struct{}{}
}

func (a *Analysis) structHasTrait(st *ast.SemStruct, trait *ast.SemTrait) bool {
	if _, ok := st.Traits[trait]; ok {
		return true
	}
	getTrait := func(inst StructInstance) (*ast.SemTrait, bool) {
		for tr := range inst.Type.Traits {
			if tr == trait {
				return tr, true
			}
		}
		return nil, false
	}
	match := func(_ *ast.SemTrait) bool { return true }
	if _, ok := findInStructStack(a, st, getTrait, match); ok {
		st.Traits[trait] = struct{}{}
		return true
	}
	return false
}

func (a *Analysis) GetStruct(def *ast.Struct, concrete []Type) *SemStruct {
	stack := def.GetStructStack()
	for _, inst := range stack {
		if len(inst.Args) != len(concrete) {
			continue
		}
		same := true
		for i, ty := range concrete {
			o := inst.Args[i]
			if !a.matchTypesStrict(ty, o) {
				same = false
				break
			}
		}
		if same {
			return inst.Type.Ref() // reuse cached *StructType
		}
	}
	return nil
}

func (a *Analysis) GetStructMethods(st *ast.SemStruct) map[string]ast.SemFunction {
	methods := make(map[string]ast.SemFunction, len(st.Methods))
	maps.Copy(methods, st.Methods) // start with already cached methods
	stack := st.Def.GetStructStack()
	for _, inst := range stack {
		if implMatchesStruct(a, inst, st) {
			for name, method := range inst.Type.Methods {
				if _, exists := methods[name]; !exists {
					methods[name] = method
				}
			}
		}
	}
	return methods
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
	if !a.matchTypesStrict(base, actual) {
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
