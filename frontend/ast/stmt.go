package ast

import (
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type Stmt interface {
	isStmt()
	Span() common.Span
}

/* Let */

type Let struct {
	Public bool

	Attributes []Attribute
	Names      []lexer.TokIdent
	Types      []*Type
	Values     []Expr

	IsItem bool
	span   common.Span

	IsGlobalDef bool // true if this is a global definition
}

func NewLet(
	names []lexer.TokIdent,
	types []*Type, // may be nil
	values []Expr,
	span common.Span,
	isItem bool,
) *Let {
	return &Let{
		Public: false,
		Names:  names,
		Types:  types,
		Values: values,
		IsItem: isItem,
		span:   span,
	}
}

func (l *Let) isItem() {}
func (l *Let) isStmt() {}

func (l *Let) Span() common.Span {
	return l.span
}

/* Return */

type StmtReturn struct {
	Exprs          []Expr
	IsFuncErroable bool
	span           common.Span
}

func NewReturnStmt(exprs []Expr, span common.Span) *StmtReturn {
	return &StmtReturn{Exprs: exprs, span: span}
}

func (r *StmtReturn) isStmt() {}

func (r *StmtReturn) Span() common.Span {
	return r.span
}

/* ExprStatement */

type StmtExpr struct {
	Expr         Expr
	HasSemicolon bool
	span         common.Span
}

func NewStmtExpr(expr Expr, hasSemicolon bool, span common.Span) *StmtExpr {
	return &StmtExpr{Expr: expr, HasSemicolon: hasSemicolon, span: span}
}

func (es *StmtExpr) isStmt() {}

func (es *StmtExpr) Span() common.Span {
	return es.span
}

/* Assignment */

type StmtAssignment struct {
	LhsExprs []Expr
	RhsExpr  []Expr
	span     common.Span
}

func NewAssignment(lhsExprs []Expr, rhsExpr []Expr, span common.Span) *StmtAssignment {
	return &StmtAssignment{LhsExprs: lhsExprs, RhsExpr: rhsExpr, span: span}
}

func (a *StmtAssignment) isStmt() {}

func (a *StmtAssignment) Span() common.Span {
	return a.span
}

/* Throw */

type StmtThrow struct {
	Value Expr
	span  common.Span
}

func NewThrowStmt(expr Expr, span common.Span) *StmtThrow {
	return &StmtThrow{Value: expr, span: span}
}

func (t *StmtThrow) isStmt() {}

func (t *StmtThrow) Span() common.Span {
	return t.span
}

/* Break */

type StmtBreak struct {
	Label *Ident
	span  common.Span
}

func NewBreakStmt(label *Ident, span common.Span) *StmtBreak {
	return &StmtBreak{Label: label, span: span}
}

func (b *StmtBreak) isStmt() {}

func (b *StmtBreak) Span() common.Span {
	return b.span
}

/* Continue */

type StmtContinue struct {
	Label *Ident
	span  common.Span
}

func NewContinueStmt(label *Ident, span common.Span) *StmtContinue {
	return &StmtContinue{Label: label, span: span}
}

func (c *StmtContinue) isStmt() {}

func (c *StmtContinue) Span() common.Span {
	return c.span
}
