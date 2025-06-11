package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleLet(scope *Scope, it *ast.Let) {
	lhsCount := len(it.Names)

	rhsTypes, rhsSpans := a.resolveRHS(scope, it.Values, lhsCount, it.Span())

	// For each LHS identifier, optionally match the explicit type, or add inlay-hint
	for i, ident := range it.Names {
		if it.IsItem && ident.Raw == "_" {
			a.panic(ident.Span(), "cannot use `_` in top-level")
		}
		ty := rhsTypes[i]
		exprSpan := rhsSpans[i]

		// If an explicit type is given, match & use it
		if len(it.Types) != 0 && it.Types[i] != nil {
			explicitTy := a.resolveType(scope, *it.Types[i])
			a.Matches(explicitTy, ty, exprSpan)
			ty = explicitTy
		} else {
			// Provide inlay hint if no explicit type is given
			a.InlayHintType(ty.String(), ident.Span())
		}

		// Add the new variable to the current scope
		variable := ast.NewVariable(*it, i, ty)
		value := ast.NewValue(variable)
		a.AddValueVisibility(scope, ident.Raw, value, ident.Span(), it.Public)
		a.AddSpanSymbol(ident.Span(), *scope.GetSymbol(ident.Raw))
	}
}

// resolveRHS flattens tuples / varargs, enforces the arity rules
// and returns a 1-to-1 list of (type, span) pairs - one for every
// target on the left-hand side.
func (a *Analysis) resolveRHS(
	scope *Scope,
	values []ast.Expr,
	lhsCount int,
	ctxSpan Span, // span used for generic “mismatched arity” error
) (types []Type, spans []Span) {
	types = make([]Type, 0, lhsCount)
	spans = make([]Span, 0, lhsCount)

	lastIdx := len(values) - 1
	for i := range values {
		expr := &values[i]
		a.handleExpr(scope, expr)
		exprTy := expr.Type()
		exprSp := expr.Span()

		if exprTy.IsError() {
			a.panic(exprSp, "error cannot be assigned to a variable")
		}

		switch {
		case exprTy.IsTuple():
			if i != lastIdx {
				a.panic(exprSp, "tuple value is only permitted as the last expression")
			}
			for _, elem := range exprTy.Tuple().Elems {
				if elem.IsVararg() {
					if len(types) >= lhsCount {
						a.panic(exprSp, "unexpected vararg value - all identifiers already bound")
					}
					for len(types) < lhsCount {
						types = append(types, a.optionType(elem.Vararg().Type, elem.Span()))
						spans = append(spans, exprSp)
					}
					break // nothing comes after vararg
				}
				types = append(types, elem)
				spans = append(spans, exprSp)
			}

		case exprTy.IsVararg():
			if i != lastIdx {
				a.panic(exprSp, "vararg value is only permitted as the last expression")
			}
			if len(types) >= lhsCount {
				a.panic(exprSp, "unexpected vararg value - all identifiers already bound")
			}
			for len(types) < lhsCount {
				types = append(types, a.optionType(exprTy.Vararg().Type, exprSp))
				spans = append(spans, exprSp)
			}

		// ordinary
		default:
			types = append(types, exprTy)
			spans = append(spans, exprSp)
		}
	}

	if len(types) != lhsCount {
		a.panicf(ctxSpan, "mismatched arity: %d target(s) on the left, %d value(s) on the right", lhsCount, len(types))
	}

	return
}
