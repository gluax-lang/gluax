package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func getGenericParamTraits(g ast.GenericParam) []*ast.SemTrait {
	traits := make([]*ast.SemTrait, 0, len(g.Constraints))
	for _, constraint := range g.Constraints {
		trait := constraint.ResolvedSymbol.Trait()
		traits = append(traits, trait)
	}
	return traits
}

func (a *Analysis) setupTypeGenerics(scope *Scope, generics ast.Generics, concrete []Type) *Scope {
	scope = NewScope(scope)
	for i, g := range generics.Params {
		var binding Type
		if concrete == nil {
			var traits []*ast.SemTrait
			for _, constraint := range g.Constraints {
				trait := a.resolvePathTrait(scope, &constraint)
				traits = append(traits, trait)
			}
			binding = ast.NewSemGenericType(g.Name, traits, true)
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
			a.panicf(a.GetStructSetupSpan(def.Span()), "type `%s` cannot be used as a generic type", ty.String())
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
			a.Error(st.Def.Name.Span(), err.Error())
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
		if len(g.Constraints) > 0 {
			for i := range g.Constraints {
				constraint := &g.Constraints[i]
				if constraint.ResolvedSymbol == nil {
					a.resolvePathTrait(scope, constraint)
				} else {
					break // already resolved
				}
			}
		}
		var binding, param Type
		if concrete == nil {
			traits := getGenericParamTraits(g)
			binding = ast.NewSemGenericType(g.Name, traits, true)
			param = ast.NewSemGenericType(g.Name, traits, false)
		} else {
			binding = concrete[i]
			param = binding
			if len(g.Constraints) > 0 {
				if binding.IsStruct() {
					st := binding.Struct()
					for _, constraint := range g.Constraints {
						// If the binding is a struct, we need to ensure it implements the trait
						// specified in the constraint.
						if !a.StructImplementsTrait(st, constraint.ResolvedSymbol.Trait()) {
							a.panicf(a.GetStructSetupSpan(binding.Span()),
								"struct `%s` does not implement trait `%s`", binding.String(), constraint.ResolvedSymbol.Trait().Def.Name)
						}
					}
				} else if binding.IsGeneric() {
					generic := binding.Generic()
					for _, constraint := range g.Constraints {
						trait := constraint.ResolvedSymbol.Trait()
						implements := false
						for _, t := range generic.Traits {
							if traitImplements(t, trait) {
								implements = true
								break
							}
						}
						if !implements {
							a.panicf(a.GetStructSetupSpan(binding.Span()),
								"generic `%s` does not implement trait `%s`", generic.Ident.Raw, trait.Def.Name)
						}
					}
				} else {
					a.panicf(a.GetStructSetupSpan(binding.Span()), "`%s` cannot be used as a generic type", binding.String())
				}
			}
		}
		a.AddType(scope, g.Name.Raw, binding)
		params = append(params, param)
	}
	st.Generics.Params = params
}

func (a *Analysis) collectStructFields(st *SemStruct) {
	for _, field := range st.Def.Fields {
		if _, ok := st.Fields[field.Name.Raw]; ok {
			a.Error(field.Name.Span(), "duplicate field name")
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
		a.panicf(a.GetStructSetupSpan(def.Span()),
			"struct `%s` expects %d generic argument(s), but %d provided", def.Name.Raw, def.Generics.Len(), len(concrete))
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
				a.panicf(span, "struct `%s` is generic but no generic arguments were provided", st.Def.Name.Raw)
			}
		}
		return st
	}

	if st.Def.Generics.IsEmpty() {
		a.panicf(span, "struct `%s` is not generic but generics were provided", st.Def.Name.Raw)
	}

	if len(generics) != st.Def.Generics.Len() {
		a.panicf(span, "expected %d generics, got %d", st.Def.Generics.Len(), len(generics))
	}

	concrete := make([]Type, 0, len(generics))
	for _, g := range generics {
		concrete = append(concrete, a.resolveType(scope, g))
	}

	st = a.instantiateStruct(st.Def, concrete)

	return st
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
			if !a.MatchTypesStrict(ty, o) {
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

func (a *Analysis) unify(
	base Type,
	actual Type,
	placeholders map[string]Type,
	span Span,
) Type {
	if disallowedKind(base) || disallowedKind(actual) {
		a.panic(span, "type cannot be used here")
	}

	if base.IsGeneric() {
		baseName := base.Generic().Ident.Raw

		// (a) Already bound?  Just return it — no more recursion.
		if bound, ok := placeholders[baseName]; ok {
			return bound
		}

		// (b) If actual is also a generic that’s already bound,
		//     bind base → that same concrete, and return.
		if actual.IsGeneric() {
			otherName := actual.Generic().Ident.Raw
			if otherBound, ok := placeholders[otherName]; ok {
				placeholders[baseName] = otherBound
				return otherBound
			}
		}

		// (c) Otherwise bind base → actual (whatever it is) and return.
		placeholders[baseName] = actual
		return actual
	}

	// If base is a struct => unify generics param-by-param
	if base.IsStruct() {
		bs := base.Struct()
		// actual must be a struct
		if !actual.IsStruct() {
			a.panicf(span, "type mismatch: expected struct `%s`, got `%s`", bs.String(), actual.String())
		}
		as := actual.Struct()

		// same def
		if bs.Def != as.Def {
			a.panicf(span, "type mismatch: expected struct `%s`, got `%s`", bs.Def.Name.Raw, as.Def.Name.Raw)
		}
		// unify each generic param
		if len(bs.Generics.Params) != len(as.Generics.Params) {
			a.panicf(span,

				"struct `%s` has %d generic param(s), but got %d in `%s`",
				bs.Def.Name.Raw,
				len(bs.Generics.Params),
				len(as.Generics.Params),
				as.Def.Name.Raw,
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
		specializedStruct := a.instantiateStruct(bs.Def, newParams)
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
	if !a.MatchTypesStrict(base, actual) {
		a.panicf(span, "mismatched types: expected `%s`, got `%s`", base.String(), actual.String())
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
