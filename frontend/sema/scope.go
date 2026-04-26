package sema

import (
	"fmt"
	"maps"

	"github.com/gluax-lang/gluax/frontend/ast"
)

// ScopeContext holds state that gets inherited (copied) into child scopes.
type ScopeContext struct {
	Func   *ast.SemFunction
	InLoop bool
	Labels map[string]struct{}
}

type Scope struct {
	Parent   *Scope
	Children []*Scope
	Symbols  map[string][]*Symbol
	Span     *Span
	Ctx      ScopeContext
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Parent:  parent,
		Symbols: make(map[string][]*Symbol),
		Ctx:     ScopeContext{Labels: make(map[string]struct{})},
	}
}

func (s *Scope) walkScopes(fn func(*Scope) bool) bool {
	for current := s; current != nil; current = current.Parent {
		if fn(current) {
			return true
		}
	}
	return false
}

func (s *Scope) Child(inheritCtx bool) *Scope {
	child := NewScope(s)
	if inheritCtx {
		child.Ctx = ScopeContext{
			Func:   s.Ctx.Func,
			InLoop: s.Ctx.InLoop,
			Labels: maps.Clone(s.Ctx.Labels),
		}
	}
	s.Children = append(s.Children, child)
	return child
}

func (s *Scope) ChildWithScope(inheritCtx bool, span Span) *Scope {
	child := s.Child(inheritCtx)
	child.Span = &span
	return child
}

func (s *Scope) IsFuncErrorable() bool {
	return s.Ctx.Func != nil && s.Ctx.Func.Def.Errorable
}

func (s *Scope) AddLabel(name string) error {
	if s.LabelExists(name) {
		return fmt.Errorf("duplicate label definition of %s", name)
	}
	s.Ctx.Labels[name] = struct{}{}
	return nil
}

func (s *Scope) LabelExists(name string) bool {
	return s.walkScopes(func(scope *Scope) bool {
		_, ok := scope.Ctx.Labels[name]
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

func (s *Scope) GetSymbolExceptRoot(name string) *Symbol {
	var result *Symbol
	s.walkScopes(func(scope *Scope) bool {
		if scope.Parent == nil {
			return false
		}
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
	return sym.Value()
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
	return sym.Type()
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
	return sym.Import()
}

func (s *Scope) IsSymbolPublic(name string) bool {
	sym := s.GetSymbol(name)
	return sym != nil && sym.IsPublic()
}
