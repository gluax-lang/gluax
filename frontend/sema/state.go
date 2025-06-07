package sema

import (
	"maps"

	"github.com/gluax-lang/gluax/frontend/ast"
)

type StructInstance struct {
	Args []Type
	Type *SemStruct
}

type StructsStack []StructInstance

type State struct {
	Label          string                       // "SERVER" or "CLIENT"
	Macros         map[string]string            // e.g. {"SERVER": ""}, {"CLIENT": ""}
	RootScope      *Scope                       // which root scope we attach to in this pass
	Files          map[string]*Analysis         // where we store the resulting analyses
	CreatedStructs map[*ast.Struct]StructsStack // Created structs stack for this state
}

func NewState(label string) *State {
	return &State{
		Label:          label,
		Macros:         make(map[string]string),
		RootScope:      NewScope(nil),
		Files:          make(map[string]*Analysis),
		CreatedStructs: make(map[*ast.Struct]StructsStack),
	}
}

func (s *State) AddStruct(def *ast.Struct, st *SemStruct, concrete []Type) {
	if _, ok := s.CreatedStructs[def]; !ok {
		s.CreatedStructs[def] = make(StructsStack, 0, 4)
	}
	s.CreatedStructs[def] = append(s.CreatedStructs[def], StructInstance{concrete, st})
}

func (s *State) GetStruct(def *ast.Struct, concrete []Type) *SemStruct {
	if stack, ok := s.CreatedStructs[def]; ok {
		for _, inst := range stack {
			if len(inst.Args) != len(concrete) {
				continue
			}
			same := true
			for i, ty := range concrete {
				if !ty.StrictMatches(inst.Args[i]) {
					same = false
					break
				}
			}
			if same {
				return inst.Type.Ref() // reuse cached *StructType
			}
		}
	}
	return nil
}

func (s *State) GetStructMethods(st *ast.SemStruct) map[string]ast.SemFunction {
	methods := make(map[string]ast.SemFunction, len(st.Methods))
	maps.Copy(methods, st.Methods) // start with already cached methods

	stack := s.GetStructStack(st.Def)
	for _, inst := range stack {
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
			for name, method := range inst.Type.Methods {
				if _, exists := methods[name]; !exists {
					methods[name] = method
				}
			}
		}
	}

	return methods
}

func (s *State) GetStructStack(def *ast.Struct) StructsStack {
	if stack, ok := s.CreatedStructs[def]; ok {
		return stack
	}
	return nil
}
