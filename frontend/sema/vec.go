package sema

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) vecType(ty Type, span common.Span) Type {
	for _, v := range a.State.CreatedVecs {
		if a.MatchTypesStrict(v.Ty, ty) {
			return ast.NewSemType(v, span)
		}
	}
	vecT := ast.NewSemVec(ty, span)
	a.State.CreatedVecs = append(a.State.CreatedVecs, vecT)
	return ast.NewSemType(vecT, span)
}

func (a *Analysis) findVecMethod(ty Type, methodName string) *ast.SemFunction {
	for _, v := range a.State.CreatedVecs {
		if !a.matchTypes(v.Ty, ty.Vec().Ty) {
			continue
		}
		for _, m := range v.Methods {
			if m.Def.Name.Raw == methodName {
				scope := m.Scope.(*Scope).Child(false)
				scope.ForceAddType("vec_T", ty.Vec().Ty)
				mDef := m.Def
				newM := a.handleFunctionSignature(scope, &mDef)
				newM.Ty = &ty
				return newM
			}
		}
	}
	return nil
}

func (a *Analysis) collectVecMethods(vecT Type) []*ast.SemFunction {
	var methods []*ast.SemFunction
	for _, createdVec := range a.State.CreatedVecs {
		if !a.matchTypes(createdVec.Ty, vecT.Vec().Ty) {
			continue
		}
		for _, m := range createdVec.Methods {
			scope := m.Scope.(*Scope).Child(false)
			scope.ForceAddType("vec_T", vecT.Vec().Ty)
			mDef := m.Def
			newM := a.handleFunction(scope, &mDef)
			newM.Ty = &vecT
			methods = append(methods, newM)
		}
	}
	return methods
}

func (a *Analysis) checkVecMethodsConflicts() {
	for _, vec := range a.State.CreatedVecs {
		vecMethods := a.collectVecMethods(ast.NewSemType(vec, vec.Span_))
		methodNames := make(map[string]common.Span)
		for _, method := range vecMethods {
			methodName := method.Def.Name.Raw
			if _, exists := methodNames[methodName]; exists {
				a.Errorf(method.Def.Name.Span(), "duplicate method `%s` for vec of type `%s`", methodName, vec.Ty.String())
			} else {
				methodNames[methodName] = method.Def.Name.Span()
			}
		}
	}
}
