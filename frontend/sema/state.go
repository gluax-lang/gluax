package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/lsp"
)

type ClassTraitsMeta struct {
	Methods map[string]*SemFunction
	Span    Span
}

type DeclWithRef struct {
	Decl LSPSymbol   // The declaration symbol
	Refs []LSPSymbol // All references to this declaration
}

type State struct {
	Label     string               // "SERVER" or "CLIENT"
	Macros    map[string]string    // e.g. {"SERVER": ""}, {"CLIENT": ""}
	RootScope *Scope               // which root scope we attach to in this pass
	Files     map[string]*Analysis // where we store the resulting analyses

	MethodsByClass map[*ast.Class]map[string]*SemFunction
	TraitsByClass  map[*ast.Class]map[*ast.SemTrait]*ClassTraitsMeta

	DeclRefs []DeclWithRef

	MainFunc *ast.SemFunction // The main function of the program, if any
}

func NewState(label string) *State {
	return &State{
		Label:          label,
		Macros:         make(map[string]string),
		RootScope:      NewScope(nil),
		Files:          make(map[string]*Analysis),
		MethodsByClass: make(map[*ast.Class]map[string]*SemFunction),
		TraitsByClass:  make(map[*ast.Class]map[*ast.SemTrait]*ClassTraitsMeta),
	}
}

func (a *Analysis) RegisterClassMethod(st *SemClass, method *SemFunction) {
	if _, ok := a.State.MethodsByClass[st.Def]; !ok {
		a.State.MethodsByClass[st.Def] = make(map[string]*SemFunction)
	}
	byName := a.State.MethodsByClass[st.Def]
	name := method.Def.Name.Raw
	// check for duplicates
	if _, exists := byName[name]; exists {
		a.Errorf(method.Def.Name.Span(),
			"duplicate method impl `%s` for class `%s`",
			name, st.Def.Name.Raw)
		return
	}
	byName[name] = method
}

func (a *Analysis) RegisterClassTraitImplementation(st *SemClass, trait *ast.SemTrait, methods map[string]*SemFunction, span Span) {
	if _, ok := a.State.TraitsByClass[st.Def]; !ok {
		a.State.TraitsByClass[st.Def] = make(map[*ast.SemTrait]*ClassTraitsMeta)
	}
	byTrait := a.State.TraitsByClass[st.Def]
	if _, exists := byTrait[trait]; exists {
		a.Errorf(span,
			"duplicate trait impl for `%s` in class `%s`",
			trait.Def.Name.Raw, st.Def.Name.Raw)
		return
	}
	byTrait[trait] = &ClassTraitsMeta{
		Methods: methods,
		Span:    span,
	}
}

func (a *Analysis) FindClassMethod(st *ast.SemClass, name string) *SemFunction {
	if bucket, exists := a.State.MethodsByClass[st.Def]; exists {
		if method, exists := bucket[name]; exists {
			inst := a.HandleClassMethod(st, method, false)
			return inst
		}
	}
	if st.Super != nil {
		return a.FindClassMethod(st.Super, name)
	}
	return nil
}

func (a *Analysis) FindClassOrTraitMethod(st *ast.SemClass, name string, scope *Scope) []*SemFunction {
	method := a.FindClassMethod(st, name)
	if method != nil {
		return []*SemFunction{method}
	}
	return a.FindClassMethodByTrait(st, name, scope)
}

func (a *Analysis) FindClassMethodByTrait(st *ast.SemClass, methodName string, scope *Scope) []*SemFunction {
	foundTraits := make(map[*ast.SemTrait]struct{})
	var results []*SemFunction

	for cls := st; cls != nil; cls = cls.Super {
		bucket, exists := a.State.TraitsByClass[cls.Def]
		if !exists {
			continue
		}
		for trait, meta := range bucket {
			if _, already := foundTraits[trait]; already {
				continue // already found in subclass
			}

			if scope != nil && !scope.IsTraitInScope(trait) {
				continue
			}

			if methodName == "" {
				for _, method := range meta.Methods {
					results = append(results, method)
				}
				foundTraits[trait] = struct{}{}
			} else if method, exists := meta.Methods[methodName]; exists {
				method.Class = cls
				results = append(results, method)
				foundTraits[trait] = struct{}{}
			}
		}
	}

	return results
}

func (a *Analysis) FindAllClassAndTraitMethods(st *ast.SemClass, scope *Scope) []*SemFunction {
	var result []*SemFunction

	for cls := st; cls != nil; cls = cls.Super {
		methodsByName := a.State.MethodsByClass[cls.Def]
		for _, method := range methodsByName {
			result = append(result, a.HandleClassMethod(st, method, false))
		}
	}

	for cls := st; cls != nil; cls = cls.Super {
		bucket, exists := a.State.TraitsByClass[cls.Def]
		if !exists {
			continue
		}
		for _, meta := range bucket {
			for _, method := range meta.Methods {
				result = append(result, method)
			}
		}
	}

	return result
}

func (a *Analysis) FindClassMethodForTraitOnly(st *ast.SemClass, trait *ast.SemTrait, methodName string) *SemFunction {
	for cls := st; cls != nil; cls = cls.Super {
		if bucket, exists := a.State.TraitsByClass[cls.Def]; exists {
			if meta, ok := bucket[trait]; ok {
				if method, exists := meta.Methods[methodName]; exists {
					method.Class = cls
					return method
				}
			}
		}
	}
	return nil
}

func (a *Analysis) FindAllClassMethods(st *ast.SemClass) map[string]*SemFunction {
	result := make(map[string]*SemFunction)

	methodsByName := a.State.MethodsByClass[st.Def]
	for name, method := range methodsByName {
		result[name] = a.HandleClassMethod(st, method, false)
	}

	return result
}

func (a *Analysis) ClassImplementsTrait(st *ast.SemClass, asked *ast.SemTrait) bool {
	if bucket, exists := a.State.TraitsByClass[st.Def]; exists {
		if _, ok := bucket[asked]; ok {
			return true
		}
	}
	if st.Super != nil {
		return a.ClassImplementsTrait(st.Super, asked)
	}
	return false
}

func (a *Analysis) GetClassesImplementingTrait(trait *ast.SemTrait) map[*ast.SemClass][]*SemFunction {
	result := make(map[*ast.SemClass][]*SemFunction)

	// Iterate through all classes that have trait implementations
	for classDef, traitMap := range a.State.TraitsByClass {
		// Check if this class implements the requested trait
		if meta, exists := traitMap[trait]; exists {
			// Get the class stack to find all instantiated classes
			classStack := classDef.GetClassStack()

			for _, classInstance := range classStack {
				semClass := classInstance

				// Collect all methods from this trait implementation
				methods := make([]*SemFunction, 0, len(meta.Methods))
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
			}
		}
	}

	return result
}

func (a *Analysis) AddDecl(declaration LSPSymbol) *DeclWithRef {
	if a.State.DeclRefs == nil {
		a.State.DeclRefs = make([]DeclWithRef, 0)
	}

	// Check if declaration already exists
	for _, dR := range a.State.DeclRefs {
		if dR.Decl.Span() == declaration.Span() {
			return &dR
		}
	}

	newDecl := DeclWithRef{
		Decl: declaration,
		Refs: make([]LSPSymbol, 0),
	}
	a.State.DeclRefs = append(a.State.DeclRefs, newDecl)
	return &a.State.DeclRefs[len(a.State.DeclRefs)-1]
}

func (a *Analysis) AddRef(decl LSPSymbol, span Span) {
	ref := ast.NewLSPRef(decl, span)
	declSpan := decl.Span()
	for i := range a.State.DeclRefs {
		if a.State.DeclRefs[i].Decl.Span() == declSpan {
			a.State.DeclRefs[i].Refs = append(a.State.DeclRefs[i].Refs, ref)
			return
		}
	}
	declWithRefs := a.AddDecl(decl)
	declWithRefs.Refs = append(declWithRefs.Refs, ref)

}

func (a *Analysis) GetRefsForDecl(declarationSpan Span) []LSPSymbol {
	for _, dR := range a.State.DeclRefs {
		if dR.Decl.Span() == declarationSpan {
			return dR.Refs
		}
	}
	return nil
}

func (a *Analysis) GetDeclAtPosition(pos lsp.Position, fPath string) *DeclWithRef {
	for _, dR := range a.State.DeclRefs {
		span := dR.Decl.Span()
		if span.Source != fPath {
			continue
		}
		declRng := span.ToRange()
		declRng.End.Character++
		if declRng.Contains(pos) {
			return &dR
		}
	}
	return nil
}

func (a *Analysis) GetSymbolAtPosition(pos lsp.Position, fPath string) *LSPSymbol {
	for _, dR := range a.State.DeclRefs {
		span := dR.Decl.Span()
		if span.Source == fPath {
			declRng := span.ToRange()
			declRng.End.Character++
			if declRng.Contains(pos) {
				return &dR.Decl
			}
		}
		for j, ref := range dR.Refs {
			span := ref.Span()
			if ref, ok := ref.(ast.LSPRef); ok {
				span = ref.RefSpan()
			}
			if span.Source != fPath {
				continue
			}
			refRng := span.ToRange()
			refRng.End.Character++
			if refRng.Contains(pos) {
				return &dR.Refs[j]
			}
		}
	}
	return nil
}
