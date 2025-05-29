package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/sema"
)

type Analysis = sema.Analysis

type bufCtx struct {
	buf strings.Builder
}

type Codegen struct {
	Analysis *Analysis
	Ast      *ast.Ast

	tempIdx int
	indent  int

	// bufCtx
	bufCtx bufCtx

	loopLblStack []loopLabel

	publicIndex int            // next index for public symbols
	publicMap   map[string]int // from symbol's "raw" name -> integer index

	generatedStructs map[string]struct{} // from decorated struct name -> struct

	tempVarStack [][]string
}

type loopLabel struct{ cont, brk string }

func (cg *Codegen) setAnalysis(analysis *Analysis) {
	cg.Analysis = analysis
	cg.Ast = analysis.Ast
}

func (cg *Codegen) buf() *strings.Builder {
	return &cg.bufCtx.buf
}

func (cg *Codegen) newBuf() bufCtx {
	old := cg.bufCtx
	cg.bufCtx = bufCtx{
		buf: strings.Builder{},
	}
	cg.bufCtx.buf.Grow(1024)
	return old
}

func (cg *Codegen) restoreBuf(old bufCtx) string {
	snippet := cg.bufCtx.buf.String()
	cg.bufCtx = old
	return snippet
}

func (cg *Codegen) writeIndent() {
	for range cg.indent {
		cg.writeByte('\t')
	}
}

func (cg *Codegen) pushIndent() { cg.indent++ }
func (cg *Codegen) popIndent() {
	if cg.indent == 0 {
		panic("codegen: popIndent underflow")
	}
	cg.indent--
}

func (cg *Codegen) writef(format string, args ...any) {
	cg.bufCtx.buf.WriteString(fmt.Sprintf(format, args...))
}

func (cg *Codegen) writeByte(b byte) {
	cg.bufCtx.buf.WriteByte(b)
}

func (cg *Codegen) writeString(s string) {
	cg.bufCtx.buf.WriteString(s)
}

func (cg *Codegen) ln(format string, args ...any) {
	if format == "" && len(args) == 0 {
		cg.writeByte('\n')
		return
	}
	cg.writeIndent()
	cg.writef(format, args...)
	cg.writeByte('\n')
}

func (cg *Codegen) temp() string {
	name := fmt.Sprintf(TEMP_PREFIX, cg.tempIdx)
	cg.tempIdx++
	return name
}

func (cg *Codegen) getPublic(symName string) string {
	if idx, exists := cg.publicMap[symName]; exists {
		return fmt.Sprintf("%s[%d]", PUBLIC_TBL, idx)
	}
	idx := cg.publicIndex
	cg.publicIndex++
	cg.publicMap[symName] = idx
	return fmt.Sprintf("%s[%d]", PUBLIC_TBL, idx)
}

func (cg *Codegen) isPublic(symName string) bool {
	_, exists := cg.publicMap[symName]
	return exists
}

func (cg *Codegen) getSymbol(symName string) *sema.Symbol {
	return cg.Analysis.Scope.GetSymbol(symName)
}

func (cg *Codegen) pushTempScope() {
	cg.tempVarStack = append(cg.tempVarStack, []string{})
}

func (cg *Codegen) popTempScope() []string {
	if len(cg.tempVarStack) == 0 {
		panic("codegen: popTempScope underflow")
	}
	vars := cg.tempVarStack[len(cg.tempVarStack)-1]
	cg.tempVarStack = cg.tempVarStack[:len(cg.tempVarStack)-1]
	return vars
}

func (cg *Codegen) getTempVar() string {
	name := cg.temp()
	if len(cg.tempVarStack) == 0 {
		panic("codegen: getTempVar called without a temp scope")
	}
	scopeIdx := len(cg.tempVarStack) - 1
	cg.tempVarStack[scopeIdx] = append(cg.tempVarStack[scopeIdx], name)
	return name
}

func (cg *Codegen) emitTempLocals() {
	vars := cg.popTempScope()
	if len(vars) > 0 {
		cg.ln("local %s;", strings.Join(vars, ", "))
	}
}

func (cg *Codegen) generate() {
	for _, item := range cg.Ast.Items {
		switch it := item.(type) {
		case *ast.Import:
			cg.genImport(it)
		}
	}
	cg.ln("")

	for _, item := range cg.Ast.Items {
		switch it := item.(type) {
		case *ast.Struct:
			if !it.Public {
				for _, inst := range cg.Analysis.State.GetStructStack(it) {
					cg.ln("local %s;", cg.decorateStName(inst.Type))
				}
			}
		case *ast.Let:
			if !it.Public {
				lhs := cg.genLetLHS(it)
				cg.ln("local %s;", strings.Join(lhs, ", "))
			}
		}
	}

	for _, item := range cg.Ast.Items {
		switch it := item.(type) {
		case *ast.Function:
			if !it.Public {
				cg.ln("local %s -- function", it.Name.Raw)
			}
		}
	}

	cg.ln("")

	for _, item := range cg.Ast.Items {
		switch st := item.(type) {
		case *ast.Struct:
			for _, inst := range cg.Analysis.State.GetStructStack(st) {
				cg.generateStruct(inst.Type)
			}
		}
	}

	for _, item := range cg.Ast.Items {
		cg.genItem(item)
	}
}

// returns "\x6D\x61\x69\x6E"
func toHexEscapedLiteral(s string) string {
	var builder strings.Builder
	for i := range len(s) {
		fmt.Fprintf(&builder, "\\x%02X", s[i])
	}
	return builder.String()
}
