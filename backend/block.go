package codegen

import "github.com/gluax-lang/gluax/frontend/ast"

type BlockFlag uint8

const BlockNone = 0

const (
	BlockWrap      BlockFlag = 1 << iota
	BlockDropValue           // drop the return value
)

func (cg *Codegen) genBlockX(b *ast.Block, flags BlockFlag) string {
	toReturn := "nil"

	if len(b.Stmts) == 0 {
		return toReturn
	}

	if flags&BlockWrap != 0 {
		cg.ln("do")
		cg.pushIndent()
	}

	for i, stmt := range b.Stmts {
		val, isValue := cg.genStmt(stmt)
		if isValue {
			toReturn = val
		}
		if i == b.StopAt() {
			break
		}
	}

	if flags&BlockDropValue != 0 {
		if toReturn != "nil" {
			cg.ln("local _ = %s", toReturn)
		}
	}

	if flags&BlockWrap != 0 {
		cg.popIndent()
		cg.ln("end")
	}

	return toReturn
}

func (cg *Codegen) genBlockDest(b *ast.Block) string {
	return cg.genBlockX(b, BlockWrap)
}
