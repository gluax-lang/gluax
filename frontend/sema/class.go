package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) setupClass(def *ast.Class) *SemClass {
	stScope := def.Scope.(*Scope).Child(false)
	st := ast.NewSemClass(def)
	st.Scope = stScope
	if a.GetClass(def) == nil {
		a.State.CreatedClasses = append(a.State.CreatedClasses, st)
	}
	return st
}

func (a *Analysis) collectClassFields(st *SemClass) {
	curIdx := 1
	if st.Super != nil {
		for _, field := range st.Super.Fields {
			name := field.Def.Name.Raw
			st.Fields[name] = field
			curIdx++
		}
	}
	for _, field := range st.Def.Fields {
		if _, ok := st.Fields[field.Name.Raw]; ok {
			a.Error(field.Name.Span(), "duplicate field name")
		}
		stScope := st.Scope.(*Scope)
		ty := a.resolveType(stScope, field.Type)
		st.Fields[field.Name.Raw] = ast.NewSemClassField(field, ty, curIdx)
		curIdx++
	}
}

func (a *Analysis) instantiateClass(clss *ast.SemClass) *SemClass {
	def := clss.Def
	if st := a.GetClass(def); st != nil {
		return st
	}

	st := a.setupClass(def)

	if def.Super != nil {
		superT := a.resolveType(st.Scope.(*Scope), *def.Super)
		st.Super = superT.Class()
	}

	a.collectClassFields(st)

	return st
}

func (a *Analysis) GetClass(def *ast.Class) *SemClass {
	for _, semClss := range a.State.CreatedClasses {
		if semClss.Def == def {
			return semClss
		}
	}
	return nil
}

func (a *Analysis) CanAccessClassField(clss *SemClass, memberPublic bool) bool {
	if memberPublic {
		return true
	}
	source := clss.Def.Span().Source
	// Private members are only accessible from the same source file
	return a.Src == source
}

func (a *Analysis) CanAccessClassMethod(method *SemFunction) bool {
	if method.Def.Public {
		return true
	}
	source := method.Def.Span().Source
	// Private members are only accessible from the same source file
	return a.Src == source
}

func (a *Analysis) handleClassInit(scope *Scope, si *ast.ExprClassInit) Type {
	baseTy := a.resolvePathType(scope, &si.Name)
	if baseTy.Kind() != ast.SemClassKind {
		a.panic(si.Name.Span(), fmt.Sprintf("expected class type for `%s`, found `%s`", si.Name.String(), baseTy.String()))
	}
	baseClass := baseTy.Class()

	// ensure all required fields are present
	providedFields := make(map[string]struct{}, len(si.Fields))
	for _, f := range si.Fields {
		providedFields[f.Name.Raw] = struct{}{}
	}
	for name, field := range baseClass.AllFields() {
		// if _, ok := providedFields[name]; !ok && ty.Kind() != ast.SemOptionalKind {
		if _, ok := providedFields[name]; !ok {
			if !field.Ty.IsNilable() {
				a.panicf(si.Span(), "missing required field `%s` in class `%s` initialization", name, baseClass.Def.Name.Raw)
			}
		}
	}

	// type-check each provided field
	for i := range si.Fields {
		f := &si.Fields[i]
		field, ok := baseClass.GetField(f.Name.Raw)
		if !ok {
			a.panic(f.Name.Span(),
				fmt.Sprintf("class `%s` has no field named `%s`",
					baseClass.Def.Name.Raw, f.Name.Raw),
			)
		}
		a.AddRef(field, f.Name.Span())
		if !a.CanAccessClassField(baseClass, field.IsPublic()) {
			a.Errorf(f.Name.Span(), "field `%s` of class `%s` is private", f.Name.Raw, baseClass.Def.Name.Raw)
		}
		a.handleExpr(scope, &f.Value)
		exprTy := f.Value.Type()
		a.Matches(field.Ty, exprTy, f.Value.Span())
	}

	return ast.NewSemType(baseClass, si.Span())
}
