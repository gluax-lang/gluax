package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/lsp"
)

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

func (a *Analysis) GetClassMethodsRecursively(st *ast.SemClass) []*SemFunction {
	var result []*SemFunction

	for cls := st; cls != nil; cls = cls.Super {
		methodsByName := a.State.MethodsByClass[cls.Def]
		for _, method := range methodsByName {
			result = append(result, a.HandleClassMethod(st, method, false))
		}
	}

	return result
}

func (a *Analysis) GetClassMethods(cls *ast.SemClass) map[string]*SemFunction {
	result := make(map[string]*SemFunction)

	methodsByName := a.State.MethodsByClass[cls.Def]
	for name, method := range methodsByName {
		result[name] = a.HandleClassMethod(cls, method, false)
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
