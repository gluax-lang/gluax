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

func (a *Analysis) setupClass(def *ast.Class, concrete []Type, buildGenerics bool) *SemClass {
	for _, ty := range concrete {
		if !isValidAsGenericTypeArgument(ty) {
			a.panicf(a.GetClassSetupSpan(def.Span()), "type `%s` cannot be used as a generic type", ty.String())
		}
	}
	stScope := def.Scope.(*Scope).Child(false)
	st := ast.NewSemClass(def)
	st.Scope = stScope
	if a.GetClass(def, concrete) == nil {
		def.AddClass(st, concrete)
	}
	if buildGenerics {
		a.buildGenericsTable(stScope, st, concrete)
	}
	return st
}

func (a *Analysis) HandleClassMethod(st *ast.SemClass, method ast.SemFunction, withBody bool) ast.SemFunction {
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
	funcTy.Class = st
	funcTy.Trait = method.Trait
	return funcTy
}

func (a *Analysis) buildGenericsTable(scope *Scope, st *SemClass, concrete []Type) {
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
				if binding.IsClass() {
					st := binding.Class()
					for _, constraint := range g.Constraints {
						// If the binding is a class, we need to ensure it implements the trait
						// specified in the constraint.
						if !a.ClassImplementsTrait(st, constraint.ResolvedSymbol.Trait()) {
							a.panicf(a.GetClassSetupSpan(binding.Span()),
								"class `%s` does not implement trait `%s`", binding.String(), constraint.ResolvedSymbol.Trait().Def.Name)
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
							a.panicf(a.GetClassSetupSpan(binding.Span()),
								"generic `%s` does not implement trait `%s`", generic.Ident.Raw, trait.Def.Name)
						}
					}
				} else {
					a.panicf(a.GetClassSetupSpan(binding.Span()), "`%s` cannot be used as a generic type", binding.String())
				}
			}
		}
		a.AddType(scope, g.Name.Raw, binding)
		params = append(params, param)
	}
	st.Generics.Params = params
}

func (a *Analysis) collectClassFields(st *SemClass) {
	curIdx := 1
	if st.Super != nil {
		for _, field := range st.Super.Fields {
			name := field.Def.Name.Raw
			st.Fields[name] = field
			curIdx++
		}
	}
	for _, field := range st.Def.Fields {
		if _, ok := st.Fields[field.Name.Raw]; ok {
			a.Error(field.Name.Span(), "duplicate field name")
		}
		stScope := st.Scope.(*Scope)
		ty := a.resolveType(stScope, field.Type)
		st.Fields[field.Name.Raw] = ast.NewSemClassField(field, ty, curIdx)
		curIdx++
	}
}

func (a *Analysis) instantiateClass(def *ast.Class, concrete []Type) *SemClass {
	if st := a.GetClass(def, concrete); st != nil {
		return st
	}

	if len(concrete) != def.Generics.Len() {
		a.panicf(a.GetClassSetupSpan(def.Span()),
			"class `%s` expects %d generic argument(s), but %d provided", def.Name.Raw, def.Generics.Len(), len(concrete))
	}

	st := a.setupClass(def, concrete, true)

	stScope := st.Scope.(*Scope)

	if def.Super != nil {
		superT := a.resolveType(st.Scope.(*Scope), *def.Super)
		st.Super = superT.Class()
	}

	stScope.ForceAddType("Self", ast.NewSemType(st, def.Span()))
	a.collectClassFields(st)

	return st
}

func (a *Analysis) resolveClass(scope *Scope, st *ast.SemClass, generics []ast.Type, span Span) *ast.SemClass {
	if len(generics) == 0 {
		if !st.Def.Generics.IsEmpty() {
			if st.Generics.UnboundCount() == st.Generics.Len() {
				a.panicf(span, "class `%s` is generic but no generic arguments were provided", st.Def.Name.Raw)
			}
		}
		return st
	}

	if st.Def.Generics.IsEmpty() {
		a.panicf(span, "class `%s` is not generic but generics were provided", st.Def.Name.Raw)
	}

	if len(generics) != st.Def.Generics.Len() {
		a.panicf(span, "expected %d generics, got %d", st.Def.Generics.Len(), len(generics))
	}

	concrete := make([]Type, 0, len(generics))
	for _, g := range generics {
		concrete = append(concrete, a.resolveType(scope, g))
	}

	st = a.instantiateClass(st.Def, concrete)

	return st
}

func (a *Analysis) GetClass(def *ast.Class, concrete []Type) *SemClass {
	stack := def.GetClassStack()
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
			return inst.Type.Ref() // reuse cached *ClassType
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

	// If base is a class => unify generics param-by-param
	if base.IsClass() {
		bs := base.Class()
		// actual must be a class
		if !actual.IsClass() {
			a.panicf(span, "type mismatch: expected class `%s`, got `%s`", bs.String(), actual.String())
		}
		as := actual.Class()

		// same def
		if bs.Def != as.Def {
			a.panicf(span, "type mismatch: expected class `%s`, got `%s`", bs.Def.Name.Raw, as.Def.Name.Raw)
		}
		// unify each generic param
		if len(bs.Generics.Params) != len(as.Generics.Params) {
			a.panicf(span,

				"class `%s` has %d generic param(s), but got %d in `%s`",
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
		// after unifying all generics, reconstruct the class type with the specialized generics
		specializedClass := a.instantiateClass(bs.Def, newParams)
		return ast.NewSemType(specializedClass, base.Span())
	}

	if base.IsUnreachable() {
		return base
	}

	if base.IsAny() {
		return base
	}

	// For everything else (e.g. base is a string/number/bool/nil literal class),
	// just see if they strictly match. If not, panic.
	if !a.MatchTypesStrict(base, actual) {
		a.panicf(span, "mismatched types: expected `%s`, got `%s`", base.String(), actual.String())
	}
	// If we get here, base == actual. Return base.
	return base
}

func (a *Analysis) canAccessClassField(clss *SemClass, memberPublic bool) bool {
	if memberPublic {
		return true
	}
	source := clss.Def.Span().Source
	// Private members are only accessible from the same source file
	return a.Src == source
}

func (a *Analysis) canAccessClassMethod(method *SemFunction) bool {
	if method.Trait != nil {
		return true
	}
	if method.Def.Public {
		return true
	}
	source := method.Def.Span().Source
	// Private members are only accessible from the same source file
	return a.Src == source
}
