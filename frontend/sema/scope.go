package sema

import (
	"fmt"
	"maps"

	"github.com/gluax-lang/gluax/frontend/ast"
)

type Scope struct {
	Parent         *Scope
	Children       []*Scope
	Symbols        map[string]Symbol
	InFunc         bool
	IsFuncErroable bool
	FuncReturnType *ast.SemType
	InLoop         bool
	Labels         map[string]struct{}
}

func NewScope(parent *Scope) *Scope {
	scope := &Scope{
		Parent:   parent,
		Children: make([]*Scope, 0),
		Symbols:  make(map[string]Symbol),
		Labels:   make(map[string]struct{}),
	}
	if parent != nil {
		parent.Children = append(parent.Children, scope)
	}
	return scope
}

func (s *Scope) Child(copyState bool) *Scope {
	child := NewScope(s)
	if copyState {
		child.InFunc = s.InFunc
		child.IsFuncErroable = s.IsFuncErroable
		child.FuncReturnType = s.FuncReturnType
		child.InLoop = s.InLoop
		child.Labels = maps.Clone(s.Labels)
	}
	return child
}

func (s *Scope) AddLabel(name string) error {
	if s.LabelExists(name) {
		return fmt.Errorf("duplicate label definition of %s", name)
	}
	s.Labels[name] = struct{}{}
	return nil
}

func (s *Scope) LabelExists(name string) bool {
	if _, ok := s.Labels[name]; ok {
		return true
	}
	if s.Parent != nil {
		return s.Parent.LabelExists(name)
	}
	return false
}

func (s *Scope) AddSymbol(name string, sym Symbol) error {
	if s.GetSymbol(name) != nil {
		return fmt.Errorf("duplicate definition of %s", name)
	}
	s.Symbols[name] = sym
	return nil
}

func (s *Scope) GetSymbol(name string) *Symbol {
	if sym, ok := s.Symbols[name]; ok {
		return &sym
	}
	if s.Parent != nil {
		return s.Parent.GetSymbol(name)
	}
	return nil
}

func (s *Scope) GetSymbolInChildren(name string) *Symbol {
	if sym, ok := s.Symbols[name]; ok {
		return &sym
	}
	for _, child := range s.Children {
		if sym := child.GetSymbolInChildren(name); sym != nil {
			return sym
		}
	}
	return nil
}

func (s *Scope) AddValue(name string, val Value, span Span) error {
	return s.AddValueVisibility(name, val, span, true)
}

func (s *Scope) AddValueVisibility(name string, val Value, span Span, public bool) error {
	if old := s.GetSymbol(name); old != nil {
		if old.Kind() == ast.SymValue {
			if !val.CanShadow(*old.Value()) {
				return fmt.Errorf("duplicate definition of %s", name)
			}
		} else {
			return fmt.Errorf("duplicate definition of %s", name)
		}
	}
	s.Symbols[name] = ast.NewSymbol(name, &val, span, public)
	return nil
}

func (s *Scope) GetValue(name string) *Value {
	sym := s.GetSymbol(name)
	if sym == nil || sym.Kind() != ast.SymValue {
		return nil
	}
	val := sym.Value()
	return val
}

func (s *Scope) AddType(name string, ty Type) error {
	return s.AddTypeVisibility(name, ty, true)
}

func (s *Scope) AddTypeVisibility(name string, ty Type, public bool) error {
	span := ty.Span()
	symbol := ast.NewSymbol(name, &ty, span, public)
	return s.AddSymbol(name, symbol)
}

func (s *Scope) ForceAddType(name string, ty Type) {
	span := ty.Span()
	sym := ast.NewSymbol(name, &ty, span, true)
	s.Symbols[name] = sym
}

func (s *Scope) GetType(name string) *Type {
	sym := s.GetSymbol(name)
	if sym == nil || sym.Kind() != ast.SymType {
		return nil
	}
	ty := sym.Type()
	return ty
}

func (s *Scope) AddImport(name string, imp ast.SemImport, span Span, public bool) error {
	symbol := ast.NewSymbol(name, &imp, span, public)
	return s.AddSymbol(name, symbol)
}

func (s *Scope) GetImport(name string) *ast.SemImport {
	sym := s.GetSymbol(name)
	if sym == nil || sym.Kind() != ast.SymImport {
		return nil
	}
	imp := sym.Import()
	return imp
}

func (s *Scope) IsSymbolPublic(name string) bool {
	sym := s.GetSymbol(name)
	if sym == nil {
		return false
	}
	return sym.IsPublic()
}
