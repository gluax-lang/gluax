package ast

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type PostfixOp interface {
	isPostfixOp()
	Span() common.Span
}

/* DotAccess */

type DotAccess struct {
	Name lexer.TokIdent
	span common.Span
}

func NewDotAccess(name lexer.TokIdent, span common.Span) *DotAccess {
	return &DotAccess{Name: name, span: span}
}

func (d *DotAccess) isPostfixOp() {}

func (d *DotAccess) Span() common.Span {
	return common.SpanFrom(d.span, d.Name.Span())
}

/* Call */

type Catch struct {
	Name  Ident
	Block Block
	span  common.Span
}

func NewCatch(name Ident, block Block, span common.Span) *Catch {
	return &Catch{Name: name, Block: block, span: span}
}

func (c *Catch) Span() common.Span {
	return c.span
}

type Call struct {
	Method    *Ident // nil if regular call
	Args      []Expr
	IsTryCall bool
	Catch     *Catch
	SemaFunc  *SemFunction
	span      common.Span
}

func NewCall(method *Ident, args []Expr, isTryCall bool, catch *Catch, span common.Span) *Call {
	return &Call{Method: method, Args: args, IsTryCall: isTryCall, Catch: catch, span: span}
}

func (c *Call) isPostfixOp() {}

func (c *Call) Span() common.Span {
	return c.span
}

/* Else for nilables */

type Else struct {
	Value Expr
	span  common.Span
}

func NewElse(value Expr, span common.Span) *Else {
	return &Else{Value: value, span: span}
}

func (e *Else) isPostfixOp() {}

func (e *Else) Span() common.Span {
	return e.span
}

/* Unwrap Nilable */

type UnwrapNilable struct {
	span common.Span
}

func NewUnwrapNilable(span common.Span) *UnwrapNilable {
	return &UnwrapNilable{span: span}
}

func (u *UnwrapNilable) isPostfixOp() {}

func (u *UnwrapNilable) Span() common.Span {
	return u.span
}
