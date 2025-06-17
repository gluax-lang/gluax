package sema

import (
	"slices"

	"github.com/gluax-lang/gluax/frontend/ast"
)

type ClassMethodEntry struct {
	// These are the types that were passed to the class when doing impl // e.g. `impl MyClass<T, U>`
	TypeParameters []Type
	// The method itself, which is a function
	Method SemFunction
}

type ClassTraitsMeta struct {
	TypeParameters []Type
	Methods        map[string]SemFunction
	Span           Span
}

type State struct {
	Label     string               // "SERVER" or "CLIENT"
	Macros    map[string]string    // e.g. {"SERVER": ""}, {"CLIENT": ""}
	RootScope *Scope               // which root scope we attach to in this pass
	Files     map[string]*Analysis // where we store the resulting analyses

	MethodsByClass map[*ast.Class]map[string][]*ClassMethodEntry
	TraitsByClass  map[*ast.Class]map[*ast.SemTrait][]*ClassTraitsMeta
}

func NewState(label string) *State {
	return &State{
		Label:          label,
		Macros:         make(map[string]string),
		RootScope:      NewScope(nil),
		Files:          make(map[string]*Analysis),
		MethodsByClass: make(map[*ast.Class]map[string][]*ClassMethodEntry),
		TraitsByClass:  make(map[*ast.Class]map[*ast.SemTrait][]*ClassTraitsMeta),
	}
}

func (a *Analysis) ValidateTypeParameterConstraints(constraints, actuals []Type) bool {
	if len(constraints) != len(actuals) {
		return false
	}
	for i, c := range constraints {
		act := actuals[i]

		switch {
		case c.IsGeneric(): // generic → check bounds
			for _, bound := range c.Generic().Traits {
				var ok bool
				switch {
				case act.IsClass():
					ok = a.ClassImplementsTrait(act.Class(), bound)
				case act.IsGeneric():
					ok = slices.Contains(act.Generic().Traits, bound)
				case act.IsDynTrait():
					ok = traitImplements(act.DynTrait().Trait, bound)
				}
				if !ok {
					return false
				}
			}

		default: // concrete → must strictly equal
			if !a.MatchTypesStrict(c, act) {
				return false
			}
		}
	}
	return true
}

func (a *Analysis) TypeParametersConflict(c1, c2 []Type) bool {
	if len(c1) != len(c2) {
		return false
	}

	for i := range c1 {
		t1, t2 := c1[i], c2[i]

		g1 := t1.Kind() == ast.SemGenericKind
		g2 := t2.Kind() == ast.SemGenericKind

		if g1 && g2 {
			// Both are generics: could overlap
			continue
		}

		if g1 && !g2 {
			for _, bound := range t1.Generic().Traits {
				if !a.ClassImplementsTrait(t2.Class(), bound) {
					return false
				}
			}
			continue
		}

		if g2 && !g1 {
			for _, bound := range t2.Generic().Traits {
				if !a.ClassImplementsTrait(t1.Class(), bound) {
					return false
				}
			}
			continue
		}

		if !a.MatchTypesStrict(t1, t2) {
			return false
		}
	}

	return true
}

func (a *Analysis) CheckConflictingMethodImplementations() {
	for _, byName := range a.State.MethodsByClass {
		for name, list := range byName {
			for i := range list {
				m1 := list[i]
				for j := i + 1; j < len(list); j++ {
					m2 := list[j]
					if a.TypeParametersConflict(m1.TypeParameters, m2.TypeParameters) {
						if m2.Method.Def.Span().Source == a.Src {
							a.Errorf(m2.Method.Def.Span(),
								"duplicate method impl for `%s`", name)
						}
						break // one is enough
					}
				}
			}
		}
	}
}

func (a *Analysis) CheckConflictingTraitImplementations() {
	for _, byTrait := range a.State.TraitsByClass {
		for tr, list := range byTrait {
			for i := range list {
				t1 := list[i]
				for j := i + 1; j < len(list); j++ {
					t2 := list[j]
					if a.TypeParametersConflict(t1.TypeParameters, t2.TypeParameters) {
						if t2.Span.Source == a.Src {
							a.Errorf(t2.Span,
								"duplicate trait impl for `%s`", tr.Def.Name.Raw)
						}
						break // one is enough
					}
				}
			}
		}
	}
}

func (a *Analysis) RegisterClassMethod(st *SemClass, method SemFunction) {
	if _, ok := a.State.MethodsByClass[st.Def]; !ok {
		a.State.MethodsByClass[st.Def] = make(map[string][]*ClassMethodEntry)
	}
	byName := a.State.MethodsByClass[st.Def]
	name := method.Def.Name.Raw
	byName[name] = append(byName[name], &ClassMethodEntry{
		TypeParameters: st.Generics.Params,
		Method:         method,
	})
}

func (a *Analysis) RegisterClassTraitImplementation(st *SemClass, trait *ast.SemTrait, methods map[string]SemFunction, span Span) {
	if _, ok := a.State.TraitsByClass[st.Def]; !ok {
		a.State.TraitsByClass[st.Def] = make(map[*ast.SemTrait][]*ClassTraitsMeta)
	}
	byTrait := a.State.TraitsByClass[st.Def]
	byTrait[trait] = append(byTrait[trait], &ClassTraitsMeta{
		TypeParameters: st.Generics.Params,
		Methods:        methods,
		Span:           span,
	})
}

func (a *Analysis) FindClassMethod(st *ast.SemClass, name string) *SemFunction {
	actual := st.Generics.Params
	if bucket, exists := a.State.MethodsByClass[st.Def]; exists {
		for _, meta := range bucket[name] {
			if !a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
				continue
			}
			inst := a.HandleClassMethod(st, meta.Method, false)
			return &inst
		}
	}
	if st.Super != nil {
		return a.FindClassMethod(st.Super, name)
	}
	return nil
}

func (a *Analysis) FindClassOrTraitMethod(st *ast.SemClass, name string, scope *Scope) []SemFunction {
	method := a.FindClassMethod(st, name)
	if method != nil {
		return []SemFunction{*method}
	}
	return a.FindClassMethodByTrait(st, name, scope)
}

func (a *Analysis) FindClassMethodByTrait(st *ast.SemClass, methodName string, scope *Scope) []SemFunction {
	actual := st.Generics.Params
	foundTraits := make(map[*ast.SemTrait]struct{})
	var results []SemFunction

	// collect methods from traits, only once per trait
	var collectMethods func(cls *ast.SemClass)
	collectMethods = func(cls *ast.SemClass) {
		if cls == nil {
			return
		}
		if bucket, exists := a.State.TraitsByClass[cls.Def]; exists {
			for trait, metas := range bucket {
				if _, already := foundTraits[trait]; already {
					continue // already found in subclass
				}

				if scope != nil && !scope.IsTraitInScope(trait) {
					continue
				}

				for _, meta := range metas {
					if a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
						if method, exists := meta.Methods[methodName]; exists {
							method.Class = st
							results = append(results, method)
							foundTraits[trait] = struct{}{}
							break // only one per trait
						}
					}
				}
			}
		}
		collectMethods(cls.Super)
	}

	collectMethods(st)
	return results
}

func (a *Analysis) FindClassMethodForTraitOnly(st *ast.SemClass, trait *ast.SemTrait, methodName string) *SemFunction {
	actual := st.Generics.Params
	if bucket, exists := a.State.TraitsByClass[st.Def]; exists {
		if metas, ok := bucket[trait]; ok {
			for _, meta := range metas {
				if a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
					if method, exists := meta.Methods[methodName]; exists {
						return &method
					}
				}
			}
		}
	}
	return nil
}

func (a *Analysis) FindAllClassMethods(st *ast.SemClass) map[string]*SemFunction {
	actual := st.Generics.Params
	result := make(map[string]*SemFunction)

	methodsByName := a.State.MethodsByClass[st.Def]
	for name, list := range methodsByName {
		for _, meta := range list {
			if !a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
				continue
			}
			inst := a.HandleClassMethod(st, meta.Method, false)
			result[name] = &inst
			break // Take the first valid implementation for this method name
		}
	}

	return result
}

func (a *Analysis) ClassImplementsTrait(st *ast.SemClass, asked *ast.SemTrait) bool {
	actual := st.Generics.Params
	if bucket, exists := a.State.TraitsByClass[st.Def]; exists {
		for _, meta := range bucket[asked] {
			if a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
				return true
			}
		}
	}
	if st.Super != nil {
		return a.ClassImplementsTrait(st.Super, asked)
	}
	return false
}

func (a *Analysis) CollectAllTraitMethods(st *ast.SemClass) map[*ast.SemTrait][]SemFunction {
	result := make(map[*ast.SemTrait][]SemFunction)
	actual := st.Generics.Params

	var collect func(cls *ast.SemClass)
	collect = func(cls *ast.SemClass) {
		if cls == nil {
			return
		}
		if bucket, exists := a.State.TraitsByClass[cls.Def]; exists {
			for trait, metas := range bucket {
				for _, meta := range metas {
					if a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
						methods := make([]SemFunction, 0, len(meta.Methods))
						for _, method := range meta.Methods {
							methods = append(methods, method)
						}
						result[trait] = append(result[trait], methods...)
					}
				}
			}
		}
		collect(cls.Super)
	}

	collect(st)
	return result
}

func (a *Analysis) GetClassesImplementingTrait(trait *ast.SemTrait) map[*ast.SemClass][]SemFunction {
	result := make(map[*ast.SemClass][]SemFunction)

	// Iterate through all classes that have trait implementations
	for classDef, traitMap := range a.State.TraitsByClass {
		// Check if this class implements the requested trait
		if metas, exists := traitMap[trait]; exists {
			// Get the class stack to find all instantiated classes
			classStack := classDef.GetClassStack()

			for _, classInstance := range classStack {
				semClass := classInstance.Type
				actual := semClass.Generics.Params

				// Check each implementation of the trait for this class
				for _, meta := range metas {
					if a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
						// Collect all methods from this trait implementation
						methods := make([]SemFunction, 0, len(meta.Methods))
						for _, method := range meta.Methods {
							methods = append(methods, method)
						}

						// Add to result, merging if class already exists
						if _, exists := result[semClass]; exists {
							// result[semClass] = append(existing, methods...)
							panic("shouldnt happen?")
						} else {
							result[semClass] = methods
						}
						break // Only take the first valid implementation per class
					}
				}
			}
		}
	}

	return result
}
