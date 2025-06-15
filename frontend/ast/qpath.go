package ast

import "github.com/gluax-lang/gluax/common"

type QPath struct {
	Type           Type
	As             Path
	MethodName     Ident
	ResolvedMethod *SemFunction
	span           common.Span
}

func NewQPath(ty Type, as Path, methodName Ident, span common.Span) *QPath {
	return &QPath{Type: ty, As: as, MethodName: methodName, span: span}
}

func (p *QPath) isType() {}
func (q *QPath) ExprKind() ExprKind {
	return ExprKindQPath
}

func (q *QPath) Span() common.Span {
	return q.span
}

func (e *Expr) QPath() *QPath {
	if e.Kind() != ExprKindQPath {
		panic("not a qualified path")
	}
	return e.data.(*QPath)
}
