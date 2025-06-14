package sema

import (
	"slices"

	"github.com/gluax-lang/gluax/frontend/ast"
)

type StructMethodEntry struct {
	// These are the types that were passed to the struct when doing impl // e.g. `impl MyStruct<T, U>`
	TypeParameters []Type
	// The method itself, which is a function
	Method SemFunction
}

type StructTraitsMeta struct {
	TypeParameters []Type
	Methods        map[string]SemFunction
	Span           Span
}

type State struct {
	Label     string               // "SERVER" or "CLIENT"
	Macros    map[string]string    // e.g. {"SERVER": ""}, {"CLIENT": ""}
	RootScope *Scope               // which root scope we attach to in this pass
	Files     map[string]*Analysis // where we store the resulting analyses

	MethodsByStruct map[*ast.Struct]map[string][]*StructMethodEntry
	TraitsByStruct  map[*ast.Struct]map[*ast.SemTrait][]*StructTraitsMeta
}

func NewState(label string) *State {
	return &State{
		Label:           label,
		Macros:          make(map[string]string),
		RootScope:       NewScope(nil),
		Files:           make(map[string]*Analysis),
		MethodsByStruct: make(map[*ast.Struct]map[string][]*StructMethodEntry),
		TraitsByStruct:  make(map[*ast.Struct]map[*ast.SemTrait][]*StructTraitsMeta),
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
				case act.IsStruct():
					ok = a.StructImplementsTrait(act.Struct(), bound)
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
				if !a.StructImplementsTrait(t2.Struct(), bound) {
					return false
				}
			}
			continue
		}

		if g2 && !g1 {
			for _, bound := range t2.Generic().Traits {
				if !a.StructImplementsTrait(t1.Struct(), bound) {
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
	for _, byName := range a.State.MethodsByStruct {
		for name, list := range byName {
			for i := range list {
				m1 := list[i]
				for j := i + 1; j < len(list); j++ {
					m2 := list[j]
					if a.TypeParametersConflict(m1.TypeParameters, m2.TypeParameters) {
						a.Errorf(m2.Method.Def.Span(),
							"duplicate method impl for `%s`", name)
						break // one is enough
					}
				}
			}
		}
	}
}

func (a *Analysis) CheckConflictingTraitImplementations() {
	for _, byTrait := range a.State.TraitsByStruct {
		for tr, list := range byTrait {
			for i := range list {
				t1 := list[i]
				for j := i + 1; j < len(list); j++ {
					t2 := list[j]
					if a.TypeParametersConflict(t1.TypeParameters, t2.TypeParameters) {
						a.Errorf(t2.Span,
							"duplicate trait impl for `%s`", tr.Def.Name.Raw)
						break // one is enough
					}
				}
			}
		}
	}
}

func (a *Analysis) RegisterStructMethod(st *SemStruct, method SemFunction) {
	if _, ok := a.State.MethodsByStruct[st.Def]; !ok {
		a.State.MethodsByStruct[st.Def] = make(map[string][]*StructMethodEntry)
	}
	byName := a.State.MethodsByStruct[st.Def]
	name := method.Def.Name.Raw
	byName[name] = append(byName[name], &StructMethodEntry{
		TypeParameters: st.Generics.Params,
		Method:         method,
	})
}

func (a *Analysis) RegisterStructTraitImplementation(st *SemStruct, trait *ast.SemTrait, methods map[string]SemFunction, span Span) {
	if _, ok := a.State.TraitsByStruct[st.Def]; !ok {
		a.State.TraitsByStruct[st.Def] = make(map[*ast.SemTrait][]*StructTraitsMeta)
	}
	byTrait := a.State.TraitsByStruct[st.Def]
	byTrait[trait] = append(byTrait[trait], &StructTraitsMeta{
		TypeParameters: st.Generics.Params,
		Methods:        methods,
		Span:           span,
	})
}

func (a *Analysis) FindStructMethod(st *ast.SemStruct, name string) *SemFunction {
	actual := st.Generics.Params
	if bucket, exists := a.State.MethodsByStruct[st.Def]; exists {
		for _, meta := range bucket[name] {
			if !a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
				continue
			}
			inst := a.HandleStructMethod(st, meta.Method, false)
			return &inst
		}
	}
	if st.Super != nil {
		method := a.FindStructMethod(st.Super, name)
		if method != nil {
			return method
		}
	}
	method := a.FindStructMethodByTrait(st, name)
	if method != nil {
		inst := a.HandleStructMethod(st, *method, false)
		return &inst
	}
	return nil
}

func (a *Analysis) FindStructMethodByTrait(st *ast.SemStruct, methodName string) *SemFunction {
	var result *SemFunction
	actual := st.Generics.Params

	if bucket, exists := a.State.TraitsByStruct[st.Def]; exists {
		for _, metas := range bucket {
			for _, meta := range metas {
				if a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
					if method, exists := meta.Methods[methodName]; exists {
						result = &method
						return result
					}
				}
			}
		}
	}

	// Check super struct
	if st.Super != nil {
		return a.FindStructMethodByTrait(st.Super, methodName)
	}

	return nil
}

func (a *Analysis) FindAllStructMethods(st *ast.SemStruct) map[string]*SemFunction {
	actual := st.Generics.Params
	result := make(map[string]*SemFunction)

	methodsByName := a.State.MethodsByStruct[st.Def]
	for name, list := range methodsByName {
		for _, meta := range list {
			if !a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
				continue
			}
			inst := a.HandleStructMethod(st, meta.Method, false)
			result[name] = &inst
			break // Take the first valid implementation for this method name
		}
	}

	return result
}

func (a *Analysis) StructImplementsTrait(st *ast.SemStruct, asked *ast.SemTrait) bool {
	actual := st.Generics.Params
	if bucket, exists := a.State.TraitsByStruct[st.Def]; exists {
		for _, meta := range bucket[asked] {
			if a.ValidateTypeParameterConstraints(meta.TypeParameters, actual) {
				return true
			}
		}
	}
	if st.Super != nil {
		return a.StructImplementsTrait(st.Super, asked)
	}
	return false
}
