package sema

import (
	"fmt"
	"maps"

	"github.com/gluax-lang/gluax/frontend/ast"
)

type Scope struct {
	Parent   *Scope
	Children []*Scope
	Symbols  map[string][]*Symbol
	Func     *ast.SemFunction // the function that this scope is in, if any
	InLoop   bool
	Labels   map[string]struct{}
	Span     *Span
}

func NewScope(parent *Scope) *Scope {
	scope := &Scope{
		Parent:  parent,
		Symbols: make(map[string][]*Symbol),
		Labels:  make(map[string]struct{}),
	}
	return scope
}

func (s *Scope) walkScopes(fn func(*Scope) bool) bool {
	current := s
	for current != nil {
		if fn(current) {
			return true
		}
		current = current.Parent
	}
	return false
}

func (s *Scope) Child(copyState bool) *Scope {
	child := NewScope(s)
	if copyState {
		child.Func = s.Func
		child.InLoop = s.InLoop
		child.Labels = maps.Clone(s.Labels)
	}
	s.Children = append(s.Children, child)
	return child
}

func (s *Scope) ChildWithScope(copyState bool, span Span) *Scope {
	child := s.Child(copyState)
	child.Span = &span
	return child
}

func (s *Scope) IsFuncErrorable() bool {
	if s.Func != nil {
		return s.Func.Def.Errorable
	}
	return false
}

func (s *Scope) AddLabel(name string) error {
	if s.LabelExists(name) {
		return fmt.Errorf("duplicate label definition of %s", name)
	}
	s.Labels[name] = struct{}{}
	return nil
}

func (s *Scope) LabelExists(name string) bool {
	return s.walkScopes(func(scope *Scope) bool {
		_, ok := scope.Labels[name]
		return ok
	})
}

func (s *Scope) AddSymbol(name string, sym *Symbol) error {
	if s.GetSymbol(name) != nil {
		return fmt.Errorf("duplicate definition of %s", name)
	}
	s.Symbols[name] = append(s.Symbols[name], sym)
	return nil
}

func (s *Scope) GetSymbol(name string) *Symbol {
	var result *Symbol
	s.walkScopes(func(scope *Scope) bool {
		if symbols, ok := scope.Symbols[name]; ok && len(symbols) > 0 {
			result = symbols[len(symbols)-1]
			return true
		}
		return false
	})
	return result
}

func (s *Scope) AddValue(name string, val *Value, span Span) error {
	return s.AddValueVisibility(name, val, span, true)
}

func (s *Scope) AddValueVisibility(name string, val *Value, span Span, public bool) error {
	if old := s.GetSymbol(name); old != nil {
		if old.Kind() == ast.SymValue {
			if !val.CanShadow(*old.Value()) {
				return fmt.Errorf("duplicate definition of %s", name)
			}
		} else {
			return fmt.Errorf("duplicate definition of %s", name)
		}
	}
	symbol := ast.NewSymbol(name, val, span, public)
	s.Symbols[name] = append(s.Symbols[name], symbol)
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
	s.Symbols[name] = append(s.Symbols[name], sym)
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

func (s *Scope) AddTrait(name string, trait *ast.SemTrait, span Span, public bool) error {
	symbol := ast.NewSymbol(name, trait, span, public)
	return s.AddSymbol(name, symbol)
}

func (s *Scope) GetTrait(name string) *ast.SemTrait {
	sym := s.GetSymbol(name)
	if sym == nil || sym.Kind() != ast.SymTrait {
		return nil
	}
	trait := sym.Trait()
	return trait
}

func (s *Scope) IsSymbolPublic(name string) bool {
	sym := s.GetSymbol(name)
	if sym == nil {
		return false
	}
	return sym.IsPublic()
}

func (s *Scope) IsTraitInScope(trait *ast.SemTrait) bool {
	return s.walkScopes(func(scope *Scope) bool {
		for _, symbolSlice := range scope.Symbols {
			for _, sym := range symbolSlice {
				if sym.Kind() == ast.SymTrait && sym.Trait() == trait {
					return true
				}
			}
		}
		return false
	})
}
