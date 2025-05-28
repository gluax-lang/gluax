package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (cg *Codegen) tempLoop(name *ast.Ident) loopLabel {
	var label loopLabel
	if name == nil {
		idx := strconv.Itoa(cg.tempIdx)
		label = loopLabel{
			cont: CONTINUE_PREFIX + idx,
			brk:  BREAK_PREFIX + idx,
		}
		cg.tempIdx++
	} else {
		label = loopLabel{
			cont: CONTINUE_PREFIX + name.Raw,
			brk:  BREAK_PREFIX + name.Raw,
		}
	}
	return label
}

func (cg *Codegen) pushLoop(ll loopLabel) { // push innermost
	cg.loopLblStack = append(cg.loopLblStack, ll)
}

func (cg *Codegen) popLoop() { // pop after `end`
	cg.loopLblStack = cg.loopLblStack[:len(cg.loopLblStack)-1]
}

func (cg *Codegen) innermostLoop() loopLabel {
	if n := len(cg.loopLblStack); n > 0 {
		return cg.loopLblStack[n-1]
	}
	panic("no loop labels, should not happen")
}

func (cg *Codegen) genExprsToLocals(exprs []ast.Expr, directLast bool) ([]string, string) {
	if len(exprs) == 0 {
		return nil, ""
	}
	var locals, resultParts []string
	for i, expr := range exprs {
		if directLast && i == len(exprs)-1 {
			resultParts = append(resultParts, cg.genExpr(expr))
		} else {
			local := cg.temp()
			locals = append(locals, local)
			cg.ln("local %s = %s;", local, cg.genExpr(expr))
			resultParts = append(resultParts, local)
		}
	}
	return locals, strings.Join(resultParts, ", ")
}

func (cg *Codegen) genExpr(e ast.Expr) string {
	return cg.genExprX(e)
}

func (cg *Codegen) genExprX(e ast.Expr) string {
	switch e.Kind() {
	case ast.ExprKindNil:
		return "nil"
	case ast.ExprKindBool:
		return strconv.FormatBool(e.Bool())
	case ast.ExprKindNumber:
		return e.Number().Raw
	case ast.ExprKindString:
		return strconv.Quote(e.String().Raw)
	case ast.ExprKindVararg:
		return "..."
	case ast.ExprKindPath:
		val := cg.genPathExpr(e.Path())
		if e.AsCond {
			return "(" + val + " ~= nil)"
		}
		return val
	case ast.ExprKindFunction:
		f := e.Type().Function()
		return "(" + cg.genFunction(&f) + ")"
	case ast.ExprKindParenthesized:
		return cg.genExprX(e.Parenthesized().Value)
	case ast.ExprKindPostfix:
		return cg.genPostfixExpr(e.Postfix())
	case ast.ExprKindTuple:
		return cg.genTupleExpr(e.Tuple())
	case ast.ExprKindBlock:
		return cg.genBlockDest(e.Block())
	case ast.ExprKindIf:
		return cg.genIfExpr(e.If())
	case ast.ExprKindWhile:
		return cg.genWhileExpr(e.While())
	case ast.ExprKindLoop:
		return cg.genLoopExpr(e.Loop())
	case ast.ExprKindUnary:
		return cg.genUnaryExpr(e.Unary())
	case ast.ExprKindBinary:
		if e.Binary().IsShortCircuit() {
			return cg.genShortCircuitExpr(e)
		} else {
			return cg.genBinaryExpr(e.Binary())
		}
	case ast.ExprKindStructInit:
		ty := e.Type()
		st := ty.Struct()
		return cg.genStructInit(e.StructInit(), st)
	case ast.ExprKindPathCall:
		callCode := cg.genPathCall(e.PathCall())
		return callCode
	case ast.ExprKindUnsafeCast:
		return cg.genExprX(e.UnsafeCast().Expr)
	case ast.ExprKindRunLua:
		return cg.genRunLua(e.RunLua())
	default:
		panic("unreachable; unhandled expression type")
	}
}

func (cg *Codegen) genPathExpr(path *ast.Path) string {
	sym := path.Symbols[len(path.Symbols)-1]
	val := sym.Value()
	switch val.Kind() {
	case ast.ValVariable:
		v := val.Variable()
		suffix := ""
		if len(path.Symbols) > 1 {
			suffix = fmt.Sprintf(" --[[%s]]", path.String())
		}
		return cg.decorateLetName(&v.Def, v.N) + suffix
	case ast.ValParameter:
		p := val.Parameter()
		return p.Def.Name.Raw
	case ast.ValFunction:
		v := val.Function()
		suffix := ""
		if len(path.Symbols) > 1 {
			suffix = fmt.Sprintf(" --[[%s]]", path.String())
		}
		return cg.decorateFuncName(&v) + suffix
	case ast.ValSingleVariable:
		return path.String()
	}
	panic("unreachable")
}

func (cg *Codegen) genBinaryExpr(binE *ast.ExprBinary) string {
	lhs := cg.temp()
	cg.ln("local %s = %s", lhs, cg.genExprX(binE.Left))
	rhs := cg.temp()
	cg.ln("local %s = %s", rhs, cg.genExprX(binE.Right))
	var op string
	switch binE.Op {
	case ast.BinaryOpInvalid:
		panic("unreachable")
	case ast.BinaryOpBitwiseOr:
		return fmt.Sprintf("bit.bor(%s,%s)", lhs, rhs)
	case ast.BinaryOpBitwiseAnd:
		return fmt.Sprintf("bit.band(%s,%s)", lhs, rhs)
	case ast.BinaryOpBitwiseXor:
		return fmt.Sprintf("bit.bxor(%s,%s)", lhs, rhs)
	case ast.BinaryOpBitwiseLeftShift:
		return fmt.Sprintf("bit.lshift(%s,%s)", lhs, rhs)
	case ast.BinaryOpBitwiseRightShift:
		return fmt.Sprintf("bit.rshift(%s,%s)", lhs, rhs)
	case ast.BinaryOpLogicalOr:
		panic("unreachable")
	case ast.BinaryOpLogicalAnd:
		panic("unreachable")
	case ast.BinaryOpLess:
		op = "<"
	case ast.BinaryOpGreater:
		op = ">"
	case ast.BinaryOpLessEqual:
		op = "<="
	case ast.BinaryOpGreaterEqual:
		op = ">="
	case ast.BinaryOpEqual:
		op = "=="
	case ast.BinaryOpNotEqual:
		op = "~="
	case ast.BinaryOpAdd:
		op = "+"
	case ast.BinaryOpSub:
		op = "-"
	case ast.BinaryOpMul:
		op = "*"
	case ast.BinaryOpDiv:
		op = "/"
	case ast.BinaryOpMod:
		op = "%"
	case ast.BinaryOpExponent:
		op = "^"
	case ast.BinaryOpConcat:
		op = ".."
	}
	return fmt.Sprintf("(%s%s%s)", lhs, op, rhs)
}

func (cg *Codegen) genUnaryExpr(unE *ast.ExprUnary) string {
	value := cg.temp()
	cg.ln("local %s = %s", value, cg.genExprX(unE.Value))
	switch unE.Op {
	case ast.UnaryOpNot:
		return fmt.Sprintf("(not %s)", value)
	case ast.UnaryOpBitwiseNot:
		return fmt.Sprintf("bit.bnot(%s)", value)
	case ast.UnaryOpNegate:
		return fmt.Sprintf("(-%s)", value)
	case ast.UnaryOpLength:
		return fmt.Sprintf("(#%s)", value)
	}
	panic("unreachable")
}

func (cg *Codegen) genIfExpr(i *ast.ExprIf) string {
	toReturn := cg.temp()

	cg.ln("local %s;", toReturn)
	var nested func(cond ast.Expr, thenBlk ast.Block, branches []ast.GuardedBlock, elseBlk *ast.Block)
	nested = func(cond ast.Expr, thenBlk ast.Block, branches []ast.GuardedBlock, elseBlk *ast.Block) {
		cg.ln("if %s then", cg.genExprX(cond))
		cg.pushIndent()

		cg.ln("%s = %s;", toReturn, cg.genBlockX(&thenBlk, BlockNone))

		cg.popIndent()

		if len(branches) > 0 {
			first, rest := branches[0], branches[1:]
			cg.ln("else")
			cg.pushIndent()
			nested(first.Cond, first.Then, rest, elseBlk)
			cg.popIndent()
		} else if elseBlk != nil {
			cg.ln("else")
			cg.pushIndent()
			cg.ln("%s = %s;", toReturn, cg.genBlockX(elseBlk, BlockNone))
			cg.popIndent()
		}

		cg.ln("end")
	}

	nested(i.Main.Cond, i.Main.Then, i.Branches, i.Else)
	return toReturn
}

func (cg *Codegen) genPostfixExpr(p *ast.ExprPostfix) string {
	value := cg.genExpr(p.Left)
	primaryTy := p.Left.Type()
	temp := cg.temp()
	switch op := p.Op.(type) {
	case *ast.DotAccess:
		return cg.genDotAccess(op, value, primaryTy)
	case *ast.Call:
		if op.Method == nil {
			return cg.genCall(op, value, primaryTy)
		} else {
			return cg.genMethodCall(op, value, primaryTy)
		}
	case *ast.Else:
		cg.ln("local %s = %s;", temp, value)
		cg.ln("if %s == nil then", temp)
		cg.pushIndent()
		cg.ln("%s = %s;", temp, cg.genExpr(op.Value))
		cg.popIndent()
		cg.ln("end")
		return temp
	case *ast.UnwrapOption:
		cg.ln("local %s = %s;", temp, value)
		cg.ln("if %s == nil then", temp)
		cg.pushIndent()
		cg.ln("error(\"unwrapping nil value\");")
		cg.popIndent()
		cg.ln("end")
		return temp
	}
	cg.ln("local %s = %s", temp, value)
	return temp
}

func (cg *Codegen) genTupleExpr(t *ast.ExprTuple) string {
	exprs := make([]string, len(t.Values))
	for i, arg := range t.Values {
		exprs[i] = cg.genExpr(arg)
	}
	return strings.Join(exprs, ", ")
}

func (cg *Codegen) genCall(call *ast.Call, toCall string, toCallTy ast.SemType) string {
	_, exprs := cg.genExprsToLocals(call.Args, true)
	callExpr := fmt.Sprintf("%s(%s)", toCall, exprs)
	fun := toCallTy.Function()
	if fun.HasVarargReturn() {
		return callExpr
	}
	locals := make([]string, fun.ReturnCount())
	for i := range locals {
		locals[i] = cg.temp()
	}
	if !call.IsTryCall && call.Catch == nil {
		cg.ln("local %s = %s;", strings.Join(locals, ", "), callExpr)
		return strings.Join(locals, ", ")
	}
	errorTemp := cg.temp()
	cg.ln("local %s, %s = %s;", errorTemp, strings.Join(locals, ", "), callExpr)
	cg.ln("do")
	cg.pushIndent()
	cg.ln("if %s ~= nil then", errorTemp)
	cg.pushIndent()
	if call.IsTryCall {
		cg.ln("return %s;", errorTemp)
	} else {
		catch := call.Catch
		cg.ln("local %s = %s;", catch.Name.Raw, errorTemp)
		blockExpr := ast.NewExpr(&catch.Block)
		cg.ln("%s = %s;", strings.Join(locals, ", "), cg.genExprX(blockExpr))
	}
	cg.popIndent()
	cg.ln("end")
	cg.popIndent()
	cg.ln("end")
	return strings.Join(locals, ", ")
}

func (cg *Codegen) genMethodCall(call *ast.Call, toCall string, toCallTy ast.SemType) string {
	exprs := make([]string, len(call.Args))
	exprs = append(exprs, toCall)
	for i, arg := range call.Args {
		exprs[i] = cg.genExpr(arg)
	}
	st := toCallTy.Struct()
	stName := cg.decorateStName(st)
	return fmt.Sprintf("%s.%s(%s)", stName, call.Method.Raw, strings.Join(exprs, ", "))
}

func (cg *Codegen) genRunLua(run *ast.ExprRunLua) string {
	var tempVars []string
	if len(run.Args) > 0 {
		tempVars, _ = cg.genExprsToLocals(run.Args, false)
	}

	code := run.Code.Raw

	// Replace argument placeholders {@1@}
	if len(tempVars) > 0 {
		code = run.GetArgRegex().ReplaceAllStringFunc(code, func(match string) string {
			submatch := run.GetArgRegex().FindStringSubmatch(match)
			argIndex, _ := strconv.Atoi(submatch[1])
			return tempVars[argIndex-1] // Convert 1-based to 0-based indexing
		})
	}

	// Replace numbered temp placeholders {@TEMP1@}
	numberedTemps := make(map[string]string)
	code = run.GetTempRegex().ReplaceAllStringFunc(code, func(match string) string {
		submatch := run.GetTempRegex().FindStringSubmatch(match)
		if len(submatch) > 1 {
			tempNum := submatch[1]
			if existing, exists := numberedTemps[tempNum]; exists {
				return existing
			}
			newTemp := cg.temp()
			numberedTemps[tempNum] = newTemp
			return newTemp
		}
		return cg.temp() // fallback
	})

	// Extract and store the return expression
	returnExpr := "nil" // Default return value if not specified
	code = run.GetReturnRegex().ReplaceAllStringFunc(code, func(match string) string {
		submatch := run.GetReturnRegex().FindStringSubmatch(match)
		if len(submatch) > 1 {
			returnExpr = strings.TrimSpace(submatch[1])
		}
		return "" // Remove the {@RETURN ...@} placeholder from the code
	})

	cg.ln("%s", code)

	return returnExpr
}

/* Loops */

func (cg *Codegen) genWhileExpr(w *ast.ExprWhile) string {
	lopLbl := cg.tempLoop(w.Label)

	cg.ln("do")
	cg.pushIndent()

	// loop entry label and test
	cg.ln("::%s::", lopLbl.cont)
	// `if %s then` is not used because it does not produce `LOOP` instruction
	// which is used by luajit to jit compile the loop
	cg.ln("if not %s then goto %s end", cg.genExpr(w.Cond), lopLbl.brk)

	cg.pushLoop(lopLbl)

	// body
	cg.genBlockX(&w.Body, BlockNone|BlockDropValue)

	cg.popLoop()

	// repeat
	cg.ln("goto %s", lopLbl.cont)

	// break label
	cg.ln("::%s::", lopLbl.brk)

	cg.popIndent()
	cg.ln("end")

	// while‐expressions don't produce a value, always return "nil"
	return "nil"
}

func (cg *Codegen) genLoopExpr(l *ast.ExprLoop) string {
	lopLbl := cg.tempLoop(l.Label)

	cg.ln("do")
	cg.pushIndent()

	// loop entry label
	cg.ln("::%s::", lopLbl.cont)

	cg.pushLoop(lopLbl)

	// body
	cg.genBlockX(&l.Body, BlockNone|BlockDropValue)

	cg.popLoop()

	// repeat
	cg.ln("goto %s", lopLbl.cont)

	// break label
	cg.ln("::%s::", lopLbl.brk)

	cg.popIndent()
	cg.ln("end")

	// loop‐expressions don't produce a value, always return "nil"
	return "nil"
}
