package codegen

import "github.com/gluax-lang/gluax/frontend/ast"

type LogicalGroup struct {
	Op    ast.BinaryOp
	Exprs []any
}

func groupLogicalExpr(e ast.Expr) any {
	if e.Kind() != ast.ExprKindBinary {
		return e
	}
	b := e.Binary()
	if b.Op != ast.BinaryOpLogicalAnd && b.Op != ast.BinaryOpLogicalOr {
		return e
	}
	g := &LogicalGroup{Op: b.Op}
	for _, child := range []ast.Expr{b.Left, b.Right} {
		sub := groupLogicalExpr(child)
		if sg, ok := sub.(*LogicalGroup); ok && sg.Op == g.Op {
			g.Exprs = append(g.Exprs, sg.Exprs...)
		} else {
			g.Exprs = append(g.Exprs, sub)
		}
	}
	return g
}

func (cg *Codegen) genShortCircuitExpr(e ast.Expr) string {
	dest := cg.getTempVar()
	end := cg.temp() + "_end"
	cg.ln("do")
	cg.pushIndent()
	cg.emitShortCircuit(groupLogicalExpr(e), dest, end)
	cg.ln("::%s::", end)
	cg.popIndent()
	cg.ln("end")
	return dest
}

func (cg *Codegen) emitShortCircuit(node any, dest, end string) {
	g, ok := node.(*LogicalGroup)
	if !ok {
		cg.ln("%s = %s", dest, cg.genExprX(node.(ast.Expr)))
		return
	}
	isAnd := g.Op == ast.BinaryOpLogicalAnd
	for i, expr := range g.Exprs {
		last := i == len(g.Exprs)-1
		if sub, ok := expr.(*LogicalGroup); ok {
			subEnd := end
			if !last {
				subEnd = cg.temp()
			}
			cg.emitShortCircuit(sub, dest, subEnd)
			if !last {
				cg.ln("::%s::", subEnd)
			}
		} else {
			cg.ln("%s = %s", dest, cg.genExprX(expr.(ast.Expr)))
		}
		if !last {
			if isAnd {
				cg.ln("if not %s then goto %s end", dest, end)
			} else {
				cg.ln("if %s then goto %s end", dest, end)
			}
		}
	}
}
