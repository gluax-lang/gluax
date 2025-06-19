package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

type FlowStatus int

const (
	FlowNormal FlowStatus = iota // no early exit
	FlowExit                     // a return/throw occurred
	FlowJump                     // a break/continue occurred
)

func (a *Analysis) handleStmt(scope *Scope, raw ast.Stmt) (Type, FlowStatus) {
	switch stmt := raw.(type) {
	case *ast.Let:
		a.handleLet(scope, stmt)
		return a.nilType(), FlowNormal

	case *ast.StmtReturn:
		a.handleReturn(scope, stmt)
		unreachable := ast.NewSemType(ast.SemUnreachable{}, stmt.Span())
		return unreachable, FlowExit
	case *ast.StmtThrow:
		a.handleThrow(scope, stmt)
		unreachable := ast.NewSemType(ast.SemUnreachable{}, stmt.Span())
		return unreachable, FlowExit

	case *ast.StmtBreak:
		a.handleBreak(scope, stmt)
		unreachable := ast.NewSemType(ast.SemUnreachable{}, stmt.Span())
		return unreachable, FlowJump
	case *ast.StmtContinue:
		a.handleContinue(scope, stmt)
		unreachable := ast.NewSemType(ast.SemUnreachable{}, stmt.Span())
		return unreachable, FlowJump

	case *ast.StmtExpr:
		flow := a.handleExprWithFlow(scope, &stmt.Expr)
		return stmt.Expr.Type(), flow

	case *ast.StmtAssignment:
		a.handleAssignment(scope, stmt)
		return a.nilType(), FlowNormal

	default:
		panic("unimplemented statement type")
	}
}

func (a *Analysis) handleReturn(scope *Scope, stmt *ast.StmtReturn) {
	if scope.Func == nil {
		a.panic(stmt.Span(), "return statement outside of function")
	}

	stmt.IsFuncErroable = scope.IsFuncErrorable()

	getReturnTypes := func() Type {
		exprs := stmt.Exprs
		var retTys []Type
		for i := range exprs {
			expr := &exprs[i]
			a.handleExpr(scope, expr)
			exprTy := expr.Type()
			if len(exprs) == 1 && i == 0 {
				return exprTy
			}
			retTys = append(retTys, exprTy)
		}
		tuple := ast.SemTuple{Elems: retTys}
		return ast.NewSemType(tuple, stmt.Span())
	}

	returnType := getReturnTypes()

	a.Matches(scope.Func.Return, returnType, stmt.Span())
}

func (a *Analysis) handleThrow(scope *Scope, stmt *ast.StmtThrow) {
	if !scope.IsFuncErrorable() {
		a.Error(stmt.Span(), "throw is not allowed outside of an erroable function")
	}
	a.handleExpr(scope, &stmt.Value)
	value := stmt.Value.Type()
	a.Matches(a.stringType(), value, stmt.Value.Span())
}

func (a *Analysis) handleBreak(scope *Scope, stmt *ast.StmtBreak) {
	if !scope.InLoop {
		a.Error(stmt.Span(), "break is not allowed outside of a loop")
	}
	if stmt.Label != nil {
		if !scope.LabelExists(stmt.Label.Raw) {
			a.Error(stmt.Label.Span(), "label not found")
		}
	}
}

func (a *Analysis) handleContinue(scope *Scope, stmt *ast.StmtContinue) {
	if !scope.InLoop {
		a.Error(stmt.Span(), "continue is not allowed outside of a loop")
	}
	if stmt.Label != nil {
		if !scope.LabelExists(stmt.Label.Raw) {
			a.Error(stmt.Label.Span(), "label not found")
		}
	}
}

func (a *Analysis) handleAssignment(scope *Scope, stmt *ast.StmtAssignment) {
	lhsCount := len(stmt.LhsExprs)
	rhsTypes, rhsSpans := a.resolveRHS(scope, stmt.RhsExpr, lhsCount, stmt.Span())
	for i := range stmt.LhsExprs {
		expr := &stmt.LhsExprs[i]
		a.handleExpr(scope, expr)
		exprTy := expr.Type()
		a.Matches(exprTy, rhsTypes[i], rhsSpans[i])
	}
}
