package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
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

func (cg *Codegen) genExprsToTempVars(exprs []ast.Expr) []string {
	if len(exprs) == 0 {
		return nil
	}
	var locals []string
	for _, expr := range exprs {
		local := cg.getTempVar()
		locals = append(locals, local)
		cg.ln("%s = %s;", local, cg.genExpr(expr))
	}
	return locals
}

func (cg *Codegen) genExprsToStrings(exprs []ast.Expr) []string {
	if len(exprs) == 0 {
		return nil
	}
	var resultParts []string
	for i, expr := range exprs {
		isLast := i == len(exprs)-1
		if isSimpleExpr(expr) || isLast {
			resultParts = append(resultParts, cg.genExpr(expr))
		} else {
			local := cg.getTempVar()
			cg.ln("%s = %s;", local, cg.genExpr(expr))
			resultParts = append(resultParts, local)
		}
	}
	return resultParts
}

func (cg *Codegen) genExprsLeftToRight(exprs []ast.Expr) string {
	parts := cg.genExprsToStrings(exprs)
	if parts == nil {
		return ""
	}
	return strings.Join(parts, ", ")
}

func isSimpleExpr(e ast.Expr) bool {
	// Simple expressions are those that can be evaluated directly without
	// needing to generate a temporary variable.
	switch e.Kind() {
	case ast.ExprKindNil, ast.ExprKindBool, ast.ExprKindNumber, ast.ExprKindString,
		ast.ExprKindVararg, ast.ExprKindFunction:
		return true
	case ast.ExprKindParenthesized:
		return isSimpleExpr(e.Parenthesized().Value)
	case ast.ExprKindPath:
		path := e.Path()
		if len(path.Segments) == 1 {
			return true
		}
	}
	return false
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
	case ast.ExprKindQPath:
		return cg.genQPathExpr(e.QPath())
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
		return cg.genBlockX(e.Block(), BlockNone)
	case ast.ExprKindIf:
		return cg.genIfExpr(e.If(), e.Type())
	case ast.ExprKindWhile:
		return cg.genWhileExpr(e.While())
	case ast.ExprKindLoop:
		return cg.genLoopExpr(e.Loop())
	case ast.ExprKindForNum:
		return cg.genForNumExpr(e.ForNum())
	case ast.ExprKindForIn:
		return cg.genForInExpr(e.ForIn())
	case ast.ExprKindUnary:
		return cg.genUnaryExpr(e.Unary())
	case ast.ExprKindBinary:
		if e.Binary().IsShortCircuit() {
			return cg.genShortCircuitExpr(e)
		} else {
			return cg.genBinaryExpr(e.Binary())
		}
	case ast.ExprKindClassInit:
		ty := e.Type()
		st := ty.Class()
		return cg.genClassInit(e.ClassInit(), st)
	case ast.ExprKindUnsafeCast:
		return cg.genExprX(e.UnsafeCast().Expr)
	case ast.ExprKindRunRaw:
		return cg.genRunRaw(e.RunRaw())
	case ast.ExprKindVecInit:
		return cg.genVecInit(e.VecInit(), e.Type())
	case ast.ExprKindMapInit:
		return cg.genMapInit(e.MapInit(), e.Type())
	default:
		panic("unreachable; unhandled expression type")
	}
}

func (cg *Codegen) genPathExpr(path *ast.Path) string {
	sym := path.ResolvedSymbol
	val := sym.Value()
	switch val.Kind() {
	case ast.ValVariable:
		v := val.Variable()
		suffix := ""
		if len(path.Segments) > 1 {
			suffix = fmt.Sprintf(" --[[%s]]", path.String())
		}
		return cg.decorateLetName(&v.Def, v.N) + suffix
	case ast.ValParameter:
		p := val.Parameter()
		return p.Def.Name.Raw
	case ast.ValFunction:
		v := val.Function()
		suffix := ""
		if len(path.Segments) > 1 {
			suffix = fmt.Sprintf(" --[[%s]]", path.String())
		}
		return cg.decorateFuncName(&v) + suffix
	case ast.ValSingleVariable:
		return path.String()
	}
	panic("unreachable")
}

func (cg *Codegen) genQPathExpr(qp *ast.QPath) string {
	return cg.decorateFuncName(qp.ResolvedMethod)
}

func (cg *Codegen) genBinaryExpr(binE *ast.ExprBinary) string {
	exprs := cg.genExprsToStrings([]ast.Expr{binE.Left, binE.Right})
	lhs := exprs[0]
	rhs := exprs[1]
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
	value := cg.genExprX(unE.Value)
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

func (cg *Codegen) genIfExpr(i *ast.ExprIf, outputTy ast.SemType) string {
	count := 1
	if outputTy.IsTuple() {
		count = len(outputTy.Tuple().Elems)
	}
	temps := make([]string, count)
	for j := range temps {
		temps[j] = cg.getTempVar()
	}
	returnList := strings.Join(temps, ", ")

	var nested func(cond ast.Expr, thenBlk ast.Block, branches []ast.GuardedBlock, elseBlk *ast.Block)
	nested = func(cond ast.Expr, thenBlk ast.Block, branches []ast.GuardedBlock, elseBlk *ast.Block) {
		cg.ln("if %s then", cg.genExprX(cond))
		cg.pushIndent()

		cg.ln("%s = %s;", returnList, cg.genBlockX(&thenBlk, BlockNone))

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
			cg.ln("%s = %s;", returnList, cg.genBlockX(elseBlk, BlockNone))
			cg.popIndent()
		}

		cg.ln("end")
	}

	nested(i.Main.Cond, i.Main.Then, i.Branches, i.Else)
	return returnList
}

func (cg *Codegen) genPostfixExpr(p *ast.ExprPostfix) string {
	value := cg.genExpr(p.Left)
	primaryTy := p.Left.Type()
	switch op := p.Op.(type) {
	case *ast.DotAccess:
		return cg.genDotAccess(op, value, primaryTy)
	case *ast.Call:
		return cg.genCall(op, value, primaryTy)
	case *ast.Else:
		temp := cg.getTempVar()
		cg.ln("%s = %s;", temp, value)
		cg.ln("if %s == nil then", temp)
		cg.pushIndent()
		cg.ln("%s = %s;", temp, cg.genExpr(op.Value))
		cg.popIndent()
		cg.ln("end")
		return temp
	case *ast.UnwrapNilable:
		// TODO: handle unwrapping nil value in DEBUG mode
		return value
		temp := cg.getTempVar()
		cg.ln("%s = %s;", temp, value)
		cg.ln("if %s == nil then", temp)
		cg.pushIndent()
		cg.ln("error(\"unwrapping nil value\");")
		cg.popIndent()
		cg.ln("end")
		return temp
	default:
		panic("unreachable; unhandled postfix operator")
	}
}

func (cg *Codegen) genTupleExpr(t *ast.ExprTuple) string {
	return cg.genExprsLeftToRight(t.Values)
}

func (cg *Codegen) genRunRaw(run *ast.ExprRunRaw) string {
	var tempVars []string
	if len(run.Args) > 0 {
		tempVars = cg.genExprsToStrings(run.Args)
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
			newTemp := cg.getTempVar()
			numberedTemps[tempNum] = newTemp
			return newTemp
		}
		return cg.getTempVar() // fallback
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

func (cg *Codegen) genVecInit(v *ast.ExprVecInit, ty ast.SemType) string {
	values := cg.genExprsLeftToRight(v.Values)
	return fmt.Sprintf("setmetatable({%s}, %s)", values, cg.decorateClassName(ty.Class()))
}

func (cg *Codegen) genMapInit(m *ast.ExprMapInit, ty ast.SemType) string {
	pairs := make([]ast.Expr, 0, len(m.Entries)*2)
	for _, entry := range m.Entries {
		pairs = append(pairs, entry.Key, entry.Value)
	}
	all := cg.genExprsToStrings(pairs)
	var sb strings.Builder
	sb.WriteString("{")
	for i := 0; i < len(all); i += 2 {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("[%s] = %s", all[i], all[i+1]))
	}
	sb.WriteString("}")
	return fmt.Sprintf("setmetatable(%s, %s)", sb.String(), cg.decorateClassName(ty.Class()))
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
	cg.genBlockX(&w.Body, BlockWrap|BlockDropValue)

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
	cg.genBlockX(&l.Body, BlockWrap|BlockDropValue)

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

func (cg *Codegen) genForNumExpr(e *ast.ExprForNum) string {
	lopLbl := cg.tempLoop(e.Label)

	cg.ln("do")
	cg.pushIndent()

	exprs := []ast.Expr{e.Start, e.End}
	if e.Step != nil {
		exprs = append(exprs, *e.Step)
	}

	cg.ln("for %s = %s do", e.Var.Raw, cg.genExprsLeftToRight(exprs))
	cg.pushIndent()

	cg.pushLoop(lopLbl)

	cg.genBlockX(&e.Body, BlockWrap|BlockDropValue)

	cg.popLoop()
	cg.ln("::%s::", lopLbl.cont)

	cg.popIndent()
	cg.ln("end")

	cg.ln("::%s::", lopLbl.brk)

	cg.popIndent()
	cg.ln("end")

	return "nil"
}

func (cg *Codegen) genForInExpr(e *ast.ExprForIn) string {
	lopLbl := cg.tempLoop(e.Label)

	isRange := e.IsRange

	cg.ln("do")
	cg.pushIndent()

	names := make([]string, len(e.Vars))
	for i, v := range e.Vars {
		names[i] = v.Raw
	}

	if isRange {
		boundMethod := lexer.NewTokIdent("__x_iter_range_bound", e.InExpr.Span())
		toCall := cg.getTempVar()
		cg.ln("%s = %s;", toCall, cg.genExpr(e.InExpr))
		boundCall := ast.Call{
			Method:   &boundMethod,
			SemaFunc: e.BoundMethod,
		}
		cg.ln("for %s = 1, %s do", names[0], cg.genCall(&boundCall, toCall, e.InExpr.Type()))
		cg.pushIndent()
		if len(names) > 1 {
			rangeMethod := lexer.NewTokIdent("__x_iter_range", e.InExpr.Span())
			rangeCall := ast.Call{
				Method:   &rangeMethod,
				Args:     []ast.Expr{ast.NewExpr(e.IdxPath)},
				SemaFunc: e.RangeMethod,
			}
			cg.ln("local %s = %s;", names[1], cg.genCall(&rangeCall, toCall, e.InExpr.Type()))
		}
	} else {
		method := lexer.NewTokIdent("__x_iter_pairs", e.InExpr.Span())
		toCall := cg.genExpr(e.InExpr)
		call := ast.Call{
			Method:   &method,
			SemaFunc: e.PairsMethod,
		}
		cg.ln("for %s in %s do", strings.Join(names, ", "), cg.genCall(&call, toCall, e.InExpr.Type()))
		cg.pushIndent()
	}

	cg.pushLoop(lopLbl)

	cg.genBlockX(&e.Body, BlockWrap|BlockDropValue)

	cg.popLoop()
	cg.ln("::%s::", lopLbl.cont)

	cg.popIndent()
	cg.ln("end")

	cg.ln("::%s::", lopLbl.brk)

	cg.popIndent()
	cg.ln("end")

	return "nil"
}
