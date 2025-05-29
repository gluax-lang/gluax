package sema

import "github.com/gluax-lang/gluax/frontend/ast"

// disallowedKind returns true if the given type is a type that cannot be used/assigned to.
func disallowedKind(t Type) bool {
	switch t.Kind() {
	case ast.SemErrorKind:
		return true
	default:
		return false
	}
}

func isInnerTypeRuleCompliant(ty Type) bool {
	switch {
	case ty.IsOption(), ty.IsNil(), ty.IsVararg(), ty.IsTuple():
		return false
	default:
		return true
	}
}
