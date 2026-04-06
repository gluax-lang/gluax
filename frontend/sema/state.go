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

	CreatedClasses []*ast.SemClass
	CreatedVecs    []*ast.SemVec
	CreatedMaps    []*ast.SemMap

	DeclRefs []DeclWithRef

	MainFunc *ast.SemFunction // The main function of the program, if any

	ValueDeps map[Span]map[Span]struct{}
}

func NewState(label string) *State {
	return &State{
		Label:     label,
		Macros:    make(map[string]string),
		RootScope: NewScope(nil),
		Files:     make(map[string]*Analysis),
		ValueDeps: make(map[Span]map[Span]struct{}),
	}
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

func (s *State) DetectCycles() [][]Span {
	var cycles [][]Span
	visited := make(map[Span]int)
	var stack []Span
	inCycle := make(map[Span]bool)

	var dfs func(Span)
	dfs = func(id Span) {
		visited[id] = 1
		stack = append(stack, id)
		for dep := range s.ValueDeps[id] {
			switch visited[dep] {
			case 1:
				if !inCycle[dep] {
					for i, v := range stack {
						if v == dep {
							cycle := make([]Span, len(stack[i:]))
							copy(cycle, stack[i:])
							for _, c := range cycle {
								inCycle[c] = true
							}
							cycles = append(cycles, cycle)
							break
						}
					}
				}
			case 0:
				dfs(dep)
			}
		}
		stack = stack[:len(stack)-1]
		visited[id] = 2
	}

	for id := range s.ValueDeps {
		if visited[id] == 0 {
			dfs(id)
		}
	}
	return cycles
}
