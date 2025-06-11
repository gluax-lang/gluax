package ast

import "github.com/gluax-lang/gluax/common"

type Block struct {
	Stmts  []Stmt
	stopAt int // index of the unreachable statement to stop at
	sem    SemType
	span   common.Span
}

func NewBlock(stmts []Stmt, span common.Span) Block {
	return Block{Stmts: stmts, span: span}
}

func (b *Block) ExprKind() ExprKind { return ExprKindBlock }

func (b Block) Span() common.Span {
	return b.span
}

func (b *Block) SetType(sem SemType) {
	b.sem = sem
}

func (b Block) Type() SemType {
	return b.sem
}

func (b *Block) SetStopAt(idx int) {
	b.stopAt = idx
}

func (b Block) StopAt() int {
	return b.stopAt
}
