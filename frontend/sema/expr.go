package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
)

func (a *Analysis) handleExprWithFlow(scope *Scope, expr *ast.Expr) FlowStatus {
	var retTy Type
	flow := FlowNormal
	switch expr.Kind() {
	case ast.ExprKindNil:
		retTy = a.nilType()
	case ast.ExprKindBool:
		retTy = a.boolType()
	case ast.ExprKindNumber:
		retTy = a.numberType()
	case ast.ExprKindString:
		retTy = a.stringType()
	case ast.ExprKindVararg:
		retTy = ast.NewSemType(ast.NewSemVararg(), expr.Span())
	case ast.ExprKindBinary:
		retTy = a.handleBinaryExpr(scope, expr.Binary())
	case ast.ExprKindUnary:
		retTy = a.handleUnaryExpr(scope, expr.Unary())
	case ast.ExprKindBlock:
		flow = a.handleBlock(scope, expr.Block())
		retTy = expr.Block().Type()
	case ast.ExprKindIf:
		retTy, flow = a.handleIfExpr(scope, expr.If())
	case ast.ExprKindWhile:
		a.handleWhileExpr(scope, expr.While())
		retTy = a.nilType()
	case ast.ExprKindLoop:
		a.handleLoopExpr(scope, expr.Loop())
		retTy = a.nilType()
	case ast.ExprKindPath:
		valueTy := a.resolvePathValue(scope, expr.Path())
		retTy = valueTy.Type()
	case ast.ExprKindParenthesized:
		flow = a.handleExprWithFlow(scope, &expr.Parenthesized().Value)
		retTy = expr.Parenthesized().Value.Type()
	case ast.ExprKindTuple:
		values := expr.Tuple().Values
		elems := make([]Type, len(values))
		last := len(values) - 1
		for i := range values {
			v := &values[i]
			a.handleExpr(scope, v)
			ty := v.Type()
			if ty.IsTuple() {
				a.Panic("cannot have nested tuples", v.Span())
			}
			if ty.IsVararg() {
				if i != last {
					a.Panic("vararg value is only permitted as the last expression", v.Span())
				}
			}
			elems[i] = ty
		}
		tuple := ast.SemTuple{Elems: elems}
		retTy = ast.NewSemType(tuple, expr.Span())
	case ast.ExprKindFunction:
		funcTy := a.handleFunction(scope, expr.Function())
		retTy = ast.NewSemType(funcTy, expr.Span())
	case ast.ExprKindPostfix:
		retTy = a.handlePostfixExpr(scope, expr.Postfix())
	case ast.ExprKindStructInit:
		retTy = a.handleStructInit(scope, expr.StructInit())
	case ast.ExprKindPathCall:
		retTy = a.handlePathCall(scope, expr.PathCall())
	}
	expr.SetType(retTy)
	return flow
}

func (a *Analysis) handleExpr(scope *Scope, expr *ast.Expr) {
	_ = a.handleExprWithFlow(scope, expr)
}

func (a *Analysis) handleBinaryExpr(scope *Scope, binE *ast.ExprBinary) Type {
	// check for disallowed chained binary expressions
	if binE.Left.Kind() == ast.ExprKindBinary {
		switch binE.Op {
		case ast.BinaryOpLess,
			ast.BinaryOpGreater,
			ast.BinaryOpLessEqual,
			ast.BinaryOpGreaterEqual,
			ast.BinaryOpEqual,
			ast.BinaryOpNotEqual:
			a.Panic("chained comparisons are not allowed", binE.Span())
		}
	}

	a.handleExpr(scope, &binE.Left)
	a.handleExpr(scope, &binE.Right)

	lty := binE.Left.Type()
	rty := binE.Right.Type()

	switch binE.Op {
	case ast.BinaryOpEqual, ast.BinaryOpNotEqual:
		// we need to compare from left and right
		// Matches is built that if left side is not optional and right side is optional, it will not match, but works vice versa
		if !lty.Matches(rty) && !rty.Matches(lty) {
			a.Error(fmt.Sprintf("cannot `%s` with `%s`", lty.String(), rty.String()), binE.Span())
		}
		return a.boolType()
	case ast.BinaryOpLogicalOr, ast.BinaryOpLogicalAnd:
		if !lty.IsLogical() {
			a.Error("expected boolean value", binE.Left.Span())
		}
		if !rty.IsLogical() {
			a.Error("expected boolean value", binE.Right.Span())
		}
		binE.Left.AsCond = true
		binE.Right.AsCond = true
		return a.boolType()
	case ast.BinaryOpLess, ast.BinaryOpGreater,
		ast.BinaryOpLessEqual, ast.BinaryOpGreaterEqual:
		if !lty.IsNumber() {
			a.Error("attempted to perform comparison on non-number value", binE.Left.Span())
		}
		if !rty.IsNumber() {
			a.Error("attempted to perform comparison on non-number value", binE.Right.Span())
		}
		return a.boolType()
	case ast.BinaryOpBitwiseOr, ast.BinaryOpBitwiseXor, ast.BinaryOpBitwiseAnd,
		ast.BinaryOpBitwiseLeftShift, ast.BinaryOpBitwiseRightShift,
		ast.BinaryOpAdd, ast.BinaryOpSub,
		ast.BinaryOpMul, ast.BinaryOpDiv,
		ast.BinaryOpMod, ast.BinaryOpExponent:
		if !lty.IsNumber() {
			a.Error("attempted to perform arithmetic on non-number value", binE.Left.Span())
		}
		if !rty.IsNumber() {
			a.Error("attempted to perform arithmetic on non-number value", binE.Right.Span())
		}
		return a.numberType()
	case ast.BinaryOpConcat:
		if !lty.IsString() {
			a.Error("attempted to concatenate non-string value", binE.Left.Span())
		}
		if !rty.IsString() {
			a.Error("attempted to concatenate non-string value", binE.Right.Span())
		}
		return a.stringType()
	}

	return a.anyType()
}

func (a *Analysis) handleUnaryExpr(scope *Scope, unE *ast.ExprUnary) Type {
	a.handleExpr(scope, &unE.Value)
	ty := unE.Value.Type()
	switch unE.Op {
	case ast.UnaryOpNot:
		if !ty.IsLogical() {
			a.Panic("unary not operator requires a boolean value", unE.Span())
		}
		return a.boolType()
	case ast.UnaryOpNegate:
		if !ty.IsNumber() {
			a.Panic("unary negate operator requires a number value", unE.Span())
		}
		return a.numberType()
	case ast.UnaryOpBitwiseNot:
		if !ty.IsNumber() {
			a.Panic("unary bitwise not operator requires an integer value", unE.Span())
		}
		return a.numberType()
	default:
		return a.anyType()
	}
}

func combineFlows(flows []FlowStatus) FlowStatus {
	if len(flows) == 0 {
		return FlowNormal
	}
	allReturn := true
	allBreak := true
	allNormal := true
	for _, f := range flows {
		if f != FlowExit {
			allReturn = false
		}
		if f != FlowJump {
			allBreak = false
		}
		if f != FlowNormal {
			allNormal = false
		}
	}
	switch {
	case allReturn:
		return FlowExit
	case allBreak:
		return FlowJump
	case allNormal:
		return FlowNormal
	default:
		// Mixed flows => treat it as normal (because it doesn't unconditionally break/return/throw).
		return FlowNormal
	}
}

func (a *Analysis) handleIfExpr(scope *Scope, ifE *ast.ExprIf) (Type, FlowStatus) {
	var branchTypes []Type
	var branchFlows []FlowStatus

	// If
	a.handleExpr(scope, &ifE.Main.Cond)
	condTy := ifE.Main.Cond.Type()
	if !condTy.IsOption() {
		a.Matches(a.boolType(), condTy, ifE.Main.Cond.Span())
	}
	ifE.Main.Cond.AsCond = true

	thenFlow := a.handleBlock(scope, &ifE.Main.Then)
	thenTy := ifE.Main.Then.Type()

	branchTypes = append(branchTypes, thenTy)
	branchFlows = append(branchFlows, thenFlow)

	// Else-if
	for i := range ifE.Branches {
		br := &ifE.Branches[i]
		a.handleExpr(scope, &br.Cond)
		cTy := br.Cond.Type()
		if !cTy.IsOption() {
			a.Matches(a.boolType(), cTy, br.Cond.Span())
		}
		br.Cond.AsCond = true

		blockFlow := a.handleBlock(scope, &br.Then)
		blockTy := br.Then.Type()
		branchTypes = append(branchTypes, blockTy)
		branchFlows = append(branchFlows, blockFlow)
	}

	// Else (if it exists)
	if ifE.Else != nil {
		elseFlow := a.handleBlock(scope, ifE.Else)
		elseTy := ifE.Else.Type()
		branchTypes = append(branchTypes, elseTy)
		branchFlows = append(branchFlows, elseFlow)
	} else {
		branchTypes = append(branchTypes, a.nilType())
		branchFlows = append(branchFlows, FlowNormal)
	}

	overallFlow := combineFlows(branchFlows)

	// Next, unify only the types from branches that ended in FlowNormal.
	// If a branch ended in FlowReturn or FlowBreak, that branch doesn't produce a
	// "usable expression result" at runtime, so skip it in the final type unification.

	var resultType *Type // nil means "not set yet"
	for i, f := range branchFlows {
		if f == FlowNormal {
			if resultType == nil {
				// first normal-flow branch
				tmp := branchTypes[i]
				resultType = &tmp
			} else {
				a.StrictMatches(*resultType, branchTypes[i], ifE.Span())
			}
		}
	}

	// If NO branch was normal => that means all branches ended with break/return/throw.
	// So the entire if-expr is effectively unreachable from the outside.
	if resultType == nil {
		unreachable := ast.NewSemType(ast.SemUnreachable{}, ifE.Span())
		return unreachable, overallFlow
	}

	// Return whichever type we unified, plus the final flow
	return *resultType, overallFlow
}

func (a *Analysis) handleWhileExpr(scope *Scope, whileE *ast.ExprWhile) {
	a.handleExpr(scope, &whileE.Cond)
	condTy := whileE.Cond.Type()
	if !condTy.IsOption() {
		a.Matches(a.boolType(), condTy, whileE.Cond.Span())
	}
	whileE.Cond.AsCond = true

	child := scope.Child(true)
	child.InLoop = true

	if whileE.Label != nil {
		a.AddLabel(child, whileE.Label)
	}

	_ = a.handleBlock(child, &whileE.Body)
	a.Matches(a.nilType(), whileE.Body.Type(), whileE.Body.Span())
}

func (a *Analysis) handleLoopExpr(scope *Scope, loopE *ast.ExprLoop) {
	child := scope.Child(true)
	child.InLoop = true

	if loopE.Label != nil {
		a.AddLabel(child, loopE.Label)
	}

	_ = a.handleBlock(child, &loopE.Body)
	a.Matches(a.nilType(), loopE.Body.Type(), loopE.Body.Span())
}

func (a *Analysis) handlePostfixExpr(scope *Scope, e *ast.ExprPostfix) Type {
	expr := &e.Left
	a.handleExpr(scope, expr)
	exprTy := e.Left.Type()

	var ty Type

	switch op := e.Op.(type) {
	case *ast.Call:
		if op.Method == nil {
			ty, _ = a.handleCall(scope, op, exprTy, expr.Span())
		} else {
			ty = a.handleMethodCall(scope, op, expr)
		}
	case *ast.DotAccess:
		ty = a.handleDotAccess(op, expr)
	case *ast.UnsafeCast:
		ty = a.handleUnsafeCast(scope, op, expr)
	case *ast.Else:
		ty = a.handleElse(scope, op, expr)
	case *ast.UnwrapOption:
		ty = a.handleUnwrapOption(scope, op, expr)
	}

	return ty
}

func (a *Analysis) handleCall(scope *Scope, call *ast.Call, toCallTy Type, span Span) (Type, *SemStruct) {
	if toCallTy.Kind() != ast.SemFunctionKind {
		a.Panic(fmt.Sprintf("expected function type, got: %s", toCallTy.String()), span)
	}
	funcTy := toCallTy.Function()

	if call.Catch != nil && !funcTy.Def.Errorable {
		a.Panic("cannot catch on non-erroable function", call.Span())
	}

	if call.IsTryCall {
		if !funcTy.Def.Errorable {
			a.Panic("cannot try-call on non-erroable function", call.Span())
		}

		if !scope.IsFuncErroable {
			a.Panic("cannot call try-call outside non erroable function", call.Span())
		}
	}

	var fixedParamTys []Type
	hasVarArgParam := false
	for i, param := range funcTy.Def.Params {
		if ast.IsVararg(param.Type) {
			hasVarArgParam = true
			break
		}
		fixedParamTys = append(fixedParamTys, funcTy.Params[i])
	}
	requiredCount := len(fixedParamTys)

	var (
		flatArgTys   []Type
		flatArgSpans []Span
	)
	lastArgIndex := len(call.Args) - 1

	appendArg := func(t Type, s Span) {
		flatArgTys = append(flatArgTys, t)
		flatArgSpans = append(flatArgSpans, s)
	}

	for i := range call.Args {
		rawArg := &call.Args[i]
		a.handleExpr(scope, rawArg)
		exprTy := rawArg.Type()
		switch exprTy.Kind() {
		case ast.SemVarargKind:
			if i != lastArgIndex {
				a.Panic("vararg value is only permitted as the last argument in a call", rawArg.Span())
			}
			if !hasVarArgParam {
				a.Panic("function does not accept vararg arguments", rawArg.Span())
			}
		case ast.SemTupleKind:
			if i != lastArgIndex {
				a.Panic("tuple value is only permitted as the last argument in a call", rawArg.Span())
			}
			for _, elemType := range exprTy.Tuple().Elems {
				appendArg(elemType, rawArg.Span())
			}
		default:
			appendArg(exprTy, rawArg.Span())
		}
	}

	if len(flatArgTys) < requiredCount {
		a.Panic(
			fmt.Sprintf("expected at least %d argument(s), found %d", requiredCount, len(flatArgTys)),
			call.Span(),
		)
	}
	if !hasVarArgParam && len(flatArgTys) != requiredCount {
		a.Panic(
			fmt.Sprintf("expected exactly %d argument(s), found %d", requiredCount, len(flatArgTys)),
			call.Span(),
		)
	}

	var newSt *SemStruct
	if owner := funcTy.OwnerStruct; owner != nil && owner.Generics.Len() > 0 {
		generics := owner.Generics
		unboundCount := generics.UnboundCount()
		if unboundCount > 0 {
			placeholders := make(map[string]Type, unboundCount)

			for i, paramTy := range funcTy.Params {
				if ast.IsVararg(funcTy.Def.Params[i].Type) {
					// for j := i; j < len(flatArgTys); j++ {
					// 	a.unify(paramTy, flatArgTys[j], placeholders, flatArgSpans[j])
					// }
					break // done unifying
				}

				if i >= len(flatArgTys) {
					break // not enough arguments, but we already handled that above
				}
				argTy := flatArgTys[i]
				a.unify(paramTy, argTy, placeholders, flatArgSpans[i])
			}

			// Build final generics
			finalGenerics := make([]Type, len(generics.Params))
			for i, g := range generics.Params {
				bound, ok := placeholders[g.Generic().Ident.Raw]
				if !ok {
					a.Panic(
						fmt.Sprintf("cannot infer generic `%s` for call", g.Generic().Ident.Raw),
						common.SpanFrom(span, call.Span()),
					)
				}
				finalGenerics[i] = bound
			}

			newSt = a.instantiateStruct(owner.Def, finalGenerics, false)
			funcTy = newSt.Methods[funcTy.Def.Name.Raw]
		}
	}

	for i := range requiredCount {
		a.Matches(funcTy.Params[i], flatArgTys[i], flatArgSpans[i])
	}

	if call.IsTryCall {
		return funcTy.Return, newSt
	}

	if call.Catch != nil {
		catch := call.Catch
		catchScope := scope.Child(true)
		errVariable := ast.NewSingleVariable(catch.Name.Raw, a.stringType())
		a.AddValue(catchScope, catch.Name.Raw, ast.NewValue(errVariable), catch.Name.Span())

		a.handleBlock(catchScope, &catch.Block)
		a.Matches(funcTy.Return, catch.Block.Type(), catch.Block.Span())
		return funcTy.Return, newSt
	}

	if funcTy.Def.Errorable {
		a.Warning("unhandled error", call.Span())
		return ast.NewErrorType(call.Span()), newSt
	}

	return funcTy.Return, newSt
}

func (a *Analysis) handleDotAccess(expr *ast.DotAccess, toIndex *ast.Expr) Type {
	toIndexTy := toIndex.Type()
	if !toIndexTy.IsStruct() {
		a.Panic(fmt.Sprintf("cannot index into non-struct type `%s`", toIndexTy.String()), expr.Span())
	}

	st := toIndexTy.Struct()

	field := expr.Name

	flds := st.Fields
	if fld, ok := flds[field.Raw]; ok {
		if !a.canAccessStructMember(st, fld.IsPublic()) {
			a.Error(fmt.Sprintf("field `%s` of struct `%s` is private", field.Raw, st.Def.Name.Raw), field.Span())
		}
		return fld.Ty
	}

	a.Panic(
		fmt.Sprintf("no field named `%s` in `%s`", field.Raw, st.Def.Name.Raw),
		field.Span(),
	)
	panic("unreachable")
}

func (a *Analysis) handleMethodCall(scope *Scope, call *ast.Call, toCall *ast.Expr) Type {
	toCallTy := toCall.Type()
	if !toCallTy.IsStruct() {
		a.Panic(fmt.Sprintf("cannot call method on non-struct type `%s`", toCallTy.String()), toCallTy.Span())
	}

	st := toCallTy.Struct()

	method, exists := st.GetMethod(call.Method.Raw)
	if !exists {
		a.Panic(
			fmt.Sprintf("no method named `%s` in `%s`", call.Method.Raw, st.Def.Name.Raw),
			call.Method.Span(),
		)
	}

	if len(method.Params) < 1 {
		a.Panic(
			fmt.Sprintf("no method named `%s` in `%s`", call.Method.Raw, st.Def.Name.Raw),
			call.Method.Span(),
		)
	}

	if !method.Params[0].StrictMatches(toCallTy) {
		a.Panic(
			fmt.Sprintf("no method named `%s` in `%s`", call.Method.Raw, st.Def.Name.Raw),
			call.Method.Span(),
		)
	}

	method.Params = method.Params[1:]
	method.Def.Params = method.Def.Params[1:]

	methodTy := ast.NewSemType(method, call.Span())

	ret, _ := a.handleCall(scope, call, methodTy, call.Span())
	return ret
}

func (a *Analysis) handleUnsafeCast(scope *Scope, as *ast.UnsafeCast, _ *ast.Expr) Type {
	unsafeCastTy := a.resolveType(scope, as.Type)
	return unsafeCastTy
}

func (a *Analysis) handleElse(scope *Scope, elseOp *ast.Else, expr *ast.Expr) Type {
	exprTy := expr.Type()
	if !exprTy.IsOption() {
		a.Panic("`else` can only be used on options", elseOp.Span())
	}
	a.handleExpr(scope, &elseOp.Value)
	a.Matches(exprTy.OptionInnerType(), elseOp.Value.Type(), elseOp.Value.Span())
	return elseOp.Value.Type()
}

func (a *Analysis) handleUnwrapOption(_ *Scope, unwrapOp *ast.UnwrapOption, expr *ast.Expr) Type {
	exprTy := expr.Type()
	if !exprTy.IsOption() {
		a.Panic("`?` can only be used on options", unwrapOp.Span())
	}
	return exprTy.OptionInnerType()
}

func (a *Analysis) handlePathCall(scope *Scope, call *ast.ExprPathCall) Type {
	{
		sym := a.resolvePathSymbol(scope, &call.Name)
		if sym.IsImport() {
			imp := sym.Import()
			analysis := imp.Analysis.(*Analysis)
			funVal := analysis.Scope.GetValue(call.MethodName.Raw)
			if funVal == nil {
				a.Panic(fmt.Sprintf("cannot find method `%s` in `%s`", call.MethodName.Raw, call.Name.String()), call.Span())
			}
			funcTy := funVal.Type()
			ret, _ := a.handleCall(scope, &call.Call, funcTy, call.Span())
			call.SetImportedFunc(funVal.Type())
			return ret
		}
	}

	baseTy := a.resolvePathType(scope, &call.Name)
	if !baseTy.IsStruct() {
		a.Panic(
			fmt.Sprintf("expected struct type, got: %s", baseTy.String()),
			call.Name.Span(),
		)
	}

	st := baseTy.Struct()
	if call.Generics != nil {
		var concrete []Type
		for _, tyAst := range call.Generics {
			concrete = append(concrete, a.resolveType(scope, tyAst))
		}
		st = a.instantiateStruct(st.Def, concrete, false)
	}

	method, exists := st.GetMethod(call.MethodName.Raw)
	if !exists {
		a.Panic(
			fmt.Sprintf("no method named `%s` in `%s`", call.MethodName.Raw, st.Def.Name.Raw),
			call.MethodName.Span(),
		)
	}

	call.SetStructSem(st)

	methodTy := ast.NewSemType(method, call.Span())

	ret, newSt := a.handleCall(scope, &call.Call, methodTy, common.SpanFrom(call.Name.Span(), call.Span()))
	if newSt != nil {
		call.SetStructSem(newSt)
	}

	return ret
}
