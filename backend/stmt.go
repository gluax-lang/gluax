package codegen

import (
	"strings"

	"github.com/gluax-lang/gluax/frontend"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) genStmt(stmt ast.Stmt) (string, bool) {
	releaseTemps := cg.collectTemps()
	defer releaseTemps()
	switch stmt := stmt.(type) {
	case *ast.StmtExpr:
		if stmt.HasSemicolon {
			// we wrap them inside a do ... end block because luajit sometimes can stack allocate them if they can't be used
			cg.ln("do")
			cg.pushIndent()
			val := cg.genExprX(stmt.Expr)
			if !isNoOp(val) {
				cg.ln("local _ = %s;", val)
			}
			cg.popIndent()
			cg.ln("end")
		} else {
			val := cg.genExprX(stmt.Expr)
			// return to avoid trying to generate any unreachable code
			return val, true
		}
	case *ast.Let:
		cg.genLet(stmt)
	case *ast.StmtContinue:
		cg.genStmtContinue(stmt)
	case *ast.StmtBreak:
		cg.genStmtBreak(stmt)
	case *ast.StmtAssignment:
		cg.genStmtAssignment(stmt)
	case *ast.StmtReturn:
		cg.genStmtReturn(stmt)
	case *ast.StmtThrow:
		cg.genStmtThrow(stmt)
	}
	return "nil", false
}

func (cg *Codegen) genStmtContinue(stmt *ast.StmtContinue) {
	var label string
	if stmt.Label != nil {
		label = frontend.CONTINUE_PREFIX + stmt.Label.Raw
	} else {
		label = cg.innermostLoop().cont
	}
	cg.ln("goto %s", label)
}

func (cg *Codegen) genStmtBreak(stmt *ast.StmtBreak) {
	var label string
	if stmt.Label != nil {
		label = frontend.BREAK_PREFIX + stmt.Label.Raw
	} else {
		label = cg.innermostLoop().brk
	}
	cg.ln("goto %s", label)
}

func (cg *Codegen) genStmtAssignment(stmt *ast.StmtAssignment) {
	lhs := make([]string, len(stmt.LhsExprs))
	for i, lhsExpr := range stmt.LhsExprs {
		lhs[i] = cg.genExpr(lhsExpr)
	}
	cg.ln("%s = %s;", strings.Join(lhs, ", "), cg.genExprsLeftToRight(stmt.RhsExpr))
}

func (cg *Codegen) genStmtReturn(stmt *ast.StmtReturn) {
	funcScope := cg.currentFuncScope()
	if funcScope.inlining {
		toReturn := cg.genExprsLeftToRight(stmt.Exprs)
		if toReturn == "" {
			toReturn = "nil"
		}
		cg.ln("%s = %s;", strings.Join(funcScope.returnVars, ", "), toReturn)
		funcScope.usedLabel = true
		cg.ln("goto %s;", funcScope.returnLabel)
		return
	}

	if stmt.Exprs == nil {
		cg.ln("do return nil; end;")
		return
	}

	retVals := cg.genExprsToStrings(stmt.Exprs)

	if stmt.IsFuncErroable {
		retVals = append([]string{"nil"}, retVals...)
	}

	cg.ln("do return %s; end;", strings.Join(retVals, ", "))
}

func (cg *Codegen) genStmtThrow(stmt *ast.StmtThrow) {
	funcScope := cg.currentFuncScope()
	if funcScope.inlining {
		value := cg.genExpr(stmt.Value)
		cg.ln("%s = %s;", funcScope.errorVar, value)
		cg.ln("goto %s;", funcScope.returnLabel)
		funcScope.usedLabel = true
		return
	}
	value := cg.genExpr(stmt.Value)
	cg.ln("do return %s; end;", value)
}
