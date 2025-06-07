package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
	"github.com/gluax-lang/gluax/frontend/sema"
)

type Analysis = sema.Analysis

type bufCtx struct {
	buf strings.Builder
}

type tempScope struct {
	all       []string // ALL temp vars created in this scope (for emitTempLocals)
	available []string // temp vars that can be reused
	allocated []string // temp vars currently allocated (not available)
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

	tempVarStack []tempScope
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
	cg.tempVarStack = append(cg.tempVarStack, tempScope{
		all:       []string{},
		available: []string{},
		allocated: []string{},
	})
}

func (cg *Codegen) currentTempScope() *tempScope {
	if len(cg.tempVarStack) == 0 {
		panic("codegen: currentTempScope called without a temp scope")
	}
	return &cg.tempVarStack[len(cg.tempVarStack)-1]
}

func (cg *Codegen) popTempScope() []string {
	if len(cg.tempVarStack) == 0 {
		panic("codegen: popTempScope underflow")
	}
	scope := cg.currentTempScope()
	cg.tempVarStack = cg.tempVarStack[:len(cg.tempVarStack)-1]
	return scope.all
}

func (cg *Codegen) getTempVar() string {
	if len(cg.tempVarStack) == 0 {
		panic("codegen: getTempVar called without a temp scope")
	}

	scope := cg.currentTempScope()

	var name string
	if len(scope.available) > 0 {
		// reuse an available temp var
		name = scope.available[len(scope.available)-1]
		scope.available = scope.available[:len(scope.available)-1]
	} else {
		// create a new temp var
		name = cg.temp()
		scope.all = append(scope.all, name)
	}

	scope.allocated = append(scope.allocated, name)
	return name
}

func (cg *Codegen) collectTemps() func() {
	if len(cg.tempVarStack) == 0 {
		panic("codegen: collectTemps called without a temp scope")
	}

	scope := cg.currentTempScope()
	marker := len(scope.allocated)
	released := false

	return func() {
		if released || len(cg.tempVarStack) == 0 {
			return // gracefully handle double-release or popped scope
		}

		currentScope := cg.currentTempScope()
		if marker < len(currentScope.allocated) {
			// move variables allocated since marker back to available pool
			releasedVars := currentScope.allocated[marker:]
			currentScope.available = append(currentScope.available, releasedVars...)
			currentScope.allocated = currentScope.allocated[:marker]
		}

		released = true
	}
}

func (cg *Codegen) emitTempLocals() {
	vars := cg.popTempScope()
	if len(vars) > 0 {
		cg.ln("local %s;", strings.Join(vars, ", "))
	}
}

func (cg *Codegen) generate() {
	for _, imp := range cg.Ast.Imports {
		cg.genImport(imp)
	}

	cg.ln("")

	for _, st := range cg.Ast.Structs {
		for _, inst := range cg.Analysis.State.GetStructStack(st) {
			cg.generateStruct(inst.Type)
		}
	}

	for _, let := range cg.Ast.Lets {
		cg.genLet(let)
		cg.ln("")
	}

	for _, funDef := range cg.Ast.Funcs {
		fun := funDef.Sem()
		name := cg.decorateFuncName(fun)
		if !funDef.Public {
			cg.currentTempScope().all = append(cg.currentTempScope().all, name)
		}
		cg.ln("%s = %s;", name, cg.genFunction(funDef.Sem()))
		cg.ln("")
	}
}

// check if "s" is a no-op expression
func isNoOp(s string) bool {
	if lexer.IsValidIdent(s) {
		return true
	}
	return false
}

func pathToLuaString(path string) string {
	return fmt.Sprintf(" [===[%s]===] ", path)
}
