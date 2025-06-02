package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

type StructMethodImpl struct {
	Method   ast.SemFunction
	Concrete []Type // concrete types for this implementation
}

type StructMethods struct {
	Methods map[string][]StructMethodImpl // method name -> list of implementations
}

func NewStructMethods() *StructMethods {
	return &StructMethods{
		Methods: make(map[string][]StructMethodImpl),
	}
}

func (s *State) AddStructMethod(structDef *ast.Struct, methodName string, method ast.SemFunction, concrete []Type) error {
	if s.StructsMethods[structDef] == nil {
		s.StructsMethods[structDef] = NewStructMethods()
	}
	if _, ok := s.GetStructMethod(structDef, methodName, concrete); ok {
		return fmt.Errorf("method '%s' already exists for these concrete types", methodName)
	}
	return s.StructsMethods[structDef].AddStructMethod(methodName, method, concrete)
}

func (s *State) GetStructMethod(structDef *ast.Struct, methodName string, requestedTypes []Type) (ast.SemFunction, bool) {
	var fun ast.SemFunction
	ok := false
	if s.StructsMethods[structDef] != nil {
		fun, ok = s.StructsMethods[structDef].GetStructMethod(methodName, requestedTypes)
	}
	return fun, ok
}

func (sm *StructMethods) AddStructMethod(methodName string, method ast.SemFunction, concrete []Type) error {
	if sm.Methods == nil {
		sm.Methods = make(map[string][]StructMethodImpl)
	}
	impl := StructMethodImpl{
		Method:   method,
		Concrete: concrete,
	}
	sm.Methods[methodName] = append(sm.Methods[methodName], impl)
	return nil
}

func (sm *StructMethods) GetStructMethod(methodName string, requestedTypes []Type) (ast.SemFunction, bool) {
	if sm.Methods == nil {
		return ast.SemFunction{}, false
	}

	impls, exists := sm.Methods[methodName]
	if !exists {
		return ast.SemFunction{}, false
	}

	var getImplType = func(impl StructMethodImpl, idx int) (Type, bool) {
		if idx < 0 || idx >= len(impl.Concrete) {
			return Type{}, false
		}
		ty := impl.Concrete[idx]
		if ty.IsGeneric() {
			return Type{}, false
		}
		return ty, true
	}

	// Try to find exact concrete match
	for _, impl := range impls {
		this := true
		for i, t := range requestedTypes {
			ty, ok := getImplType(impl, i)
			if !ok {
				continue
			}
			if !t.StrictMatches(ty) {
				this = false
				break
			}
		}
		if this {
			return impl.Method, true
		}
	}

	return ast.SemFunction{}, false
}
