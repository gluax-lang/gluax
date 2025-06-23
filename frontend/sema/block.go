package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleBlock(scope *Scope, block *ast.Block) FlowStatus {
	child := scope.ChildWithScope(true, block.Span())
	blockTy, blockFlow := a.nilType(), FlowNormal
	lastStmtSpan := block.Span()

	stmtCount := len(block.Stmts)
	for i, raw := range block.Stmts {
		stmtTy, stmtFlow := a.handleStmt(child, raw)

		if exprStmt, ok := raw.(*ast.StmtExpr); ok {
			lastStmt := (i == stmtCount-1)
			if !exprStmt.HasSemicolon {
				if !lastStmt {
					if !exprStmt.Expr.IsBlock() {
						a.panic(exprStmt.Expr.Span(), "expected `;` after expression statement")
					} else {
						a.Matches(a.nilType(), stmtTy, exprStmt.Expr.Span())
					}
				} else {
					// last statement in the block with no semicolon
					blockTy, blockFlow = stmtTy, stmtFlow
					lastStmtSpan = raw.Span()
				}
			} else {
				// It has a semicolon => do not override stmtTy => that statement
				// produces no final value
			}
		}

		switch stmtFlow {
		case FlowExit:
			block.SetStopAt(i)
			block.SetType(stmtTy)
			return stmtFlow
		case FlowJump:
			block.SetStopAt(i)
			block.SetType(stmtTy)
			return stmtFlow
		case FlowNormal:
			// just keep going
		}
	}

	if blockTy.IsTuple() {
		elemTys := blockTy.Tuple().Elems
		if len(elemTys) > 0 && elemTys[len(elemTys)-1].IsVararg() {
			a.Error(lastStmtSpan, "cannot return vararg value")
		}
	} else if blockTy.IsVararg() {
		a.Error(lastStmtSpan, "cannot return vararg value")
	}

	block.SetStopAt(-1)
	block.SetType(blockTy)

	// If we get here, the block ended normally, no break/continue/return/throw
	// return blockTy, blockFlow
	return blockFlow
}
