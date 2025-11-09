package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) setupClass(def *ast.Class) *SemClass {
	stScope := def.Scope.(*Scope).Child(false)
	st := ast.NewSemClass(def)
	st.Scope = stScope
	if a.GetClass(def) == nil {
		def.AddClass(st)
	}
	return st
}

func (a *Analysis) HandleClassMethod(st *ast.SemClass, method *ast.SemFunction, withBody bool) *ast.SemFunction {
	classScope := (method.Scope.(*Scope)).Child(false)
	{
		stTy := ast.NewSemType(st, st.Def.Name.Span())
		if err := classScope.AddType("Self", stTy); err != nil {
			a.Error(st.Def.Name.Span(), err.Error())
		}
	}
	var funcTy *ast.SemFunction
	if withBody {
		funcTy = a.handleFunction(classScope, &method.Def)
	} else {
		funcTy = a.handleFunctionSignature(classScope, &method.Def)
	}
	funcTy.Scope = method.Scope
	funcTy.Class = st
	funcTy.Trait = method.Trait
	return funcTy
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

func (a *Analysis) instantiateClass(def *ast.Class) *SemClass {
	if st := a.GetClass(def); st != nil {
		return st
	}

	st := a.setupClass(def)

	stScope := st.Scope.(*Scope)

	if def.Super != nil {
		superT := a.resolveType(st.Scope.(*Scope), *def.Super)
		st.Super = superT.Class()
	}

	stScope.ForceAddType("Self", ast.NewSemType(st, def.Span()))
	a.collectClassFields(st)

	return st
}

func (a *Analysis) resolveClass(scope *Scope, st *ast.SemClass, span Span) *ast.SemClass {
	st = a.instantiateClass(st.Def)
	return st
}

func (a *Analysis) GetClass(def *ast.Class) *SemClass {
	stack := def.GetClassStack()
	for _, inst := range stack {
		return inst.Type.Ref() // reuse cached *ClassType
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
	if method.Trait != nil {
		return true
	}
	if method.Def.Public {
		return true
	}
	source := method.Def.Span().Source
	// Private members are only accessible from the same source file
	return a.Src == source
}
