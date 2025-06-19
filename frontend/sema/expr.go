package sema

import (
	"strconv"

	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
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
		fun := scope.Func
		if fun == nil {
			a.panic(expr.Span(), "vararg outside of function")
		}
		if !fun.HasVarargParam() {
			a.panic(expr.Span(), "vararg used in function that does not accept varargs")
		}
		retTy = ast.NewSemType(ast.NewSemVararg(fun.VarargParamType()), expr.Span())
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
	case ast.ExprKindForNum:
		a.handleForNumExpr(scope, expr.ForNum())
		retTy = a.nilType()
	case ast.ExprKindForIn:
		a.handleForInExpr(scope, expr.ForIn())
		retTy = a.nilType()
	case ast.ExprKindPath:
		valueTy := a.resolvePathValue(scope, expr.Path())
		retTy = valueTy.Type()
	case ast.ExprKindQPath:
		retTy = a.handleQPathExpr(scope, expr.QPath())
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
				a.panic(v.Span(), "cannot have nested tuples")
			}
			if ty.IsVararg() {
				if i != last {
					a.panic(v.Span(), "vararg value is only permitted as the last expression")
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
	case ast.ExprKindClassInit:
		retTy = a.handleClassInit(scope, expr.ClassInit())
	case ast.ExprKindUnsafeCast:
		retTy = a.handleUnsafeCast(scope, expr.UnsafeCast())
	case ast.ExprKindRunRaw:
		retTy = a.handleRunRaw(scope, expr.RunRaw())
	case ast.ExprKindVecInit:
		retTy = a.handleVecInit(scope, expr.VecInit())
	case ast.ExprKindMapInit:
		retTy = a.handleMapInit(scope, expr.MapInit())
	default:
		panic("unreachable: unknown expression kind " + expr.Kind().String())
	}
	expr.SetType(retTy)
	return flow
}

func (a *Analysis) handleExpr(scope *Scope, expr *ast.Expr) {
	_ = a.handleExprWithFlow(scope, expr)
}

func (a *Analysis) handleQPathExpr(scope *Scope, qPath *ast.QPath) Type {
	methodName := qPath.MethodName.Raw

	toCastTy := a.resolveType(scope, qPath.Type)
	if toCastTy.IsClass() {
		class := toCastTy.Class()
		as := a.resolvePathTrait(scope, &qPath.As)
		if !a.ClassImplementsTrait(class, as) {
			a.panicf(qPath.Type.Span(), "class `%s` does not implement trait `%s`", class.Def.Name.Raw, as.Def.Name.Raw)
		}
		methodP := a.FindClassMethodForTraitOnly(class, as, methodName)
		if methodP == nil {
			a.panicf(qPath.Type.Span(), "no method found for `%s` in trait `%s`", methodName, as.Def.Name.Raw)
		}
		qPath.ResolvedMethod = methodP
		return ast.NewSemType(*methodP, qPath.Span())
	} else {
		a.panicf(qPath.Type.Span(), "expected class type, got: %s", toCastTy.String())
	}
	return a.anyType()
}

func isComparisonOp(op ast.BinaryOp) bool {
	switch op {
	case ast.BinaryOpLess,
		ast.BinaryOpGreater,
		ast.BinaryOpLessEqual,
		ast.BinaryOpGreaterEqual,
		ast.BinaryOpEqual,
		ast.BinaryOpNotEqual:
		return true
	}
	return false
}

func (a *Analysis) handleBinaryExpr(scope *Scope, binE *ast.ExprBinary) Type {
	if binE.Left.Kind() == ast.ExprKindBinary {
		leftBin := binE.Left.Binary()
		if isComparisonOp(leftBin.Op) && isComparisonOp(binE.Op) {
			a.panic(binE.Span(), "chained comparisons are not allowed")
		}
	}

	a.handleExpr(scope, &binE.Left)
	a.handleExpr(scope, &binE.Right)

	lty := binE.Left.Type()
	rty := binE.Right.Type()

	switch binE.Op {
	case ast.BinaryOpEqual, ast.BinaryOpNotEqual:
		// we need to compare from left and right
		// Matches is built that if left side is not nilable and right side is nilable, it will not match, but works vice versa
		if !a.matchTypes(lty, rty) && !a.matchTypes(rty, lty) {
			a.Errorf(binE.Span(), "cannot `%s` with `%s`", lty.String(), rty.String())
		}
		return a.boolType()
	case ast.BinaryOpLogicalOr, ast.BinaryOpLogicalAnd:
		if !lty.IsLogical() {
			a.Errorf(binE.Left.Span(), "expected boolean value, got: %s", lty.String())
		}
		if !rty.IsLogical() {
			a.Errorf(binE.Right.Span(), "expected boolean value, got: %s", rty.String())
		}
		binE.Left.AsCond = true
		binE.Right.AsCond = true
		return a.boolType()
	case ast.BinaryOpLess, ast.BinaryOpGreater,
		ast.BinaryOpLessEqual, ast.BinaryOpGreaterEqual:
		if !lty.IsNumber() {
			a.Errorf(binE.Left.Span(), "attempted to perform comparison on non-number value, got: %s", lty.String())
		}
		if !rty.IsNumber() {
			a.Errorf(binE.Right.Span(), "attempted to perform comparison on non-number value, got: %s", rty.String())
		}
		return a.boolType()
	case ast.BinaryOpBitwiseOr, ast.BinaryOpBitwiseXor, ast.BinaryOpBitwiseAnd,
		ast.BinaryOpBitwiseLeftShift, ast.BinaryOpBitwiseRightShift,
		ast.BinaryOpAdd, ast.BinaryOpSub,
		ast.BinaryOpMul, ast.BinaryOpDiv,
		ast.BinaryOpMod, ast.BinaryOpExponent:
		if !lty.IsNumber() {
			a.Errorf(binE.Left.Span(), "attempted to perform arithmetic on non-number value, got: %s", lty.String())
		}
		if !rty.IsNumber() {
			a.Errorf(binE.Right.Span(), "attempted to perform arithmetic on non-number value, got: %s", rty.String())
		}
		return a.numberType()
	case ast.BinaryOpConcat:
		if !lty.IsString() {
			a.Errorf(binE.Left.Span(), "attempted to concatenate non-string value, got: %s", lty.String())
		}
		if !rty.IsString() {
			a.Errorf(binE.Right.Span(), "attempted to concatenate non-string value, got: %s", rty.String())
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
			a.panic(unE.Span(), "unary not operator requires a boolean value")
		}
		return a.boolType()
	case ast.UnaryOpNegate:
		if !ty.IsNumber() {
			a.panic(unE.Span(), "unary negate operator requires a number value")
		}
		return a.numberType()
	case ast.UnaryOpBitwiseNot:
		if !ty.IsNumber() {
			a.panic(unE.Span(), "unary bitwise not operator requires an integer value")
		}
		return a.numberType()
	case ast.UnaryOpLength:
		if !ty.IsString() {
			a.panic(unE.Span(), "unary length operator requires a string value")
		}
		return a.numberType()
	default:
		panic("unreachable: unknown unary operator")
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
	if !condTy.IsNilable() {
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
		if !cTy.IsNilable() {
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

	var resultType *Type // nil means "not set yet"
	for i, branchType := range branchTypes {
		if branchFlows[i] != FlowNormal {
			continue
		}
		if branchType.Kind() == ast.SemUnreachableKind {
			continue
		}

		if resultType == nil {
			// First reachable type becomes the result type
			tmp := branchTypes[i]
			resultType = &tmp
		} else {
			a.StrictMatches(*resultType, branchTypes[i], ifE.Span())
		}
	}

	// If NO branch had a reachable type, the whole expression is unreachable
	if resultType == nil {
		unreachable := ast.NewSemType(ast.SemUnreachable{}, ifE.Span())
		return unreachable, overallFlow
	}

	return *resultType, overallFlow
}

func (a *Analysis) handleWhileExpr(scope *Scope, whileE *ast.ExprWhile) {
	a.handleExpr(scope, &whileE.Cond)
	condTy := whileE.Cond.Type()
	if !condTy.IsNilable() {
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

func (a *Analysis) handleForNumExpr(scope *Scope, forE *ast.ExprForNum) {
	a.handleExpr(scope, &forE.Start)
	a.Matches(a.numberType(), forE.Start.Type(), forE.Start.Span())

	a.handleExpr(scope, &forE.End)
	a.Matches(a.numberType(), forE.End.Type(), forE.End.Span())

	if forE.Step != nil {
		a.handleExpr(scope, forE.Step)
		a.Matches(a.numberType(), forE.Step.Type(), forE.Step.Span())
	}

	child := scope.Child(true)
	child.InLoop = true

	idxVariable := ast.NewSingleVariable(forE.Var, a.numberType())
	a.AddValue(child, forE.Var.Raw, ast.NewValue(idxVariable), forE.Var.Span())

	if forE.Label != nil {
		a.AddLabel(child, forE.Label)
	}

	a.handleBlock(child, &forE.Body)
	a.Matches(a.nilType(), forE.Body.Type(), forE.Body.Span())
}

func (a *Analysis) handleForInExpr(scope *Scope, forIn *ast.ExprForIn) {
	inExpr := &forIn.InExpr
	a.handleExpr(scope, inExpr)

	inExprTy := inExpr.Type()
	if !inExprTy.IsClass() {
		a.panicf(inExpr.Span(), "expected class type, got: %s", inExprTy.String())
	}

	var iterReturn Type
	var iterReturnCount int

	st := inExprTy.Class()
	if method := a.FindClassMethod(st, "__x_iter_pairs"); method != nil {
		firstReturn := method.FirstReturnType()
		iterFunc := firstReturn.Function()
		iterReturn = iterFunc.Return
		iterReturnCount = iterFunc.ReturnCount()
	} else if method := a.FindClassMethod(st, "__x_iter_range"); method != nil {
		iterReturn = a.tupleType(inExpr.Span(), a.numberType(), method.FirstReturnType())
		iterReturnCount = 2
		forIn.IsRange = true
	} else {
		a.panic(inExpr.Span(), "cannot iterate over class without __x_iter_pairs method")
	}

	variableCount := len(forIn.Vars)
	if variableCount > iterReturnCount {
		a.panicf(forIn.Span(), "for-in loop declares %d variables, but iterator returns %d value(s)", variableCount, iterReturnCount)
	}

	varsTypes := make([]Type, variableCount)
	if iterReturn.IsTuple() {
		tuple := iterReturn.Tuple()
		for i := range variableCount {
			varsTypes[i] = tuple.Elems[i]
		}
	} else {
		varsTypes[0] = iterReturn
	}

	child := scope.Child(true)
	child.InLoop = true

	for i, v := range forIn.Vars {
		varName := v.Raw
		varType := varsTypes[i]
		idxVariable := ast.NewSingleVariable(v, varType)
		a.AddValue(child, varName, ast.NewValue(idxVariable), v.Span())
		if i == 0 && forIn.IsRange {
			idxPath := ast.NewSimplePath(lexer.NewTokIdent(varName, v.Span()))
			idxPath.ResolvedSymbol = child.GetSymbol(varName)
			forIn.IdxPath = &idxPath
		}
		a.InlayHintType(varType.String(), v.Span())
	}

	if forIn.Label != nil {
		a.AddLabel(child, forIn.Label)
	}

	a.handleBlock(child, &forIn.Body)
	a.Matches(a.nilType(), forIn.Body.Type(), forIn.Body.Span())
}

func (a *Analysis) handlePostfixExpr(scope *Scope, e *ast.ExprPostfix) Type {
	expr := &e.Left
	a.handleExpr(scope, expr)
	exprTy := e.Left.Type()

	var ty Type

	switch op := e.Op.(type) {
	case *ast.Call:
		if op.Method == nil {
			ty = a.handleCall(scope, op, exprTy, expr.Span())
		} else {
			ty = a.handleMethodCall(scope, op, expr)
		}
	case *ast.DotAccess:
		ty = a.handleDotAccess(op, expr)
	case *ast.Else:
		ty = a.handleElse(scope, op, expr)
	case *ast.UnwrapNilable:
		ty = a.handleUnwrapNilable(scope, op, expr)
	}

	return ty
}

func (a *Analysis) handleCall(scope *Scope, call *ast.Call, toCallTy Type, span Span) Type {
	if toCallTy.Kind() != ast.SemFunctionKind {
		a.panicf(span, "expected function type, got: %s", toCallTy.String())
	}
	funcTy := toCallTy.Function()

	if call.Catch != nil && !funcTy.Def.Errorable {
		a.panic(call.Span(), "cannot catch on non-erroable function")
	}

	if call.IsTryCall {
		if !funcTy.Def.Errorable {
			a.panic(call.Span(), "cannot try-call on non-erroable function")
		}

		if !scope.IsFuncErrorable() {
			a.panic(call.Span(), "cannot call try-call outside non erroable function")
		}
	}

	var fixedParams []Type
	var varargParam Type
	hasVararg := false
	for i, param := range funcTy.Def.Params {
		if ast.IsVararg(param.Type) {
			hasVararg = true
			varargParam = funcTy.Params[i]
			break
		}
		fixedParams = append(fixedParams, funcTy.Params[i])
	}

	var (
		processedArgs  []Type
		processedSpans []Span
	)

	appendArg := func(t Type, s Span) {
		processedArgs = append(processedArgs, t)
		processedSpans = append(processedSpans, s)
	}

	for i := range call.Args {
		rawArg := &call.Args[i]
		a.handleExpr(scope, rawArg)
		argType := rawArg.Type()
		isLastArg := i == len(call.Args)-1

		switch argType.Kind() {
		case ast.SemVarargKind:
			if !isLastArg {
				a.panic(rawArg.Span(), "vararg value is only permitted as the last argument in a call")
			}
			if !hasVararg {
				a.panic(rawArg.Span(), "function does not accept vararg arguments")
			}
			a.Matches(varargParam, argType, rawArg.Span())
		case ast.SemTupleKind:
			if !isLastArg {
				a.panic(rawArg.Span(), "tuple value is only permitted as the last argument in a call")
			}
			for _, elemType := range argType.Tuple().Elems {
				appendArg(elemType, rawArg.Span())
			}
		default:
			appendArg(argType, rawArg.Span())
		}
	}

	requiredCount := len(fixedParams)
	actualCount := len(processedArgs)

	if actualCount < requiredCount {
		a.panicf(call.Span(), "expected at least %d argument(s), found %d", requiredCount, actualCount)
	}
	if !hasVararg && actualCount != requiredCount {
		a.panicf(call.Span(), "expected exactly %d argument(s), found %d", requiredCount, actualCount)
	}

	for i := range requiredCount {
		a.Matches(funcTy.Params[i], processedArgs[i], processedSpans[i])
	}

	if hasVararg {
		for i := requiredCount; i < actualCount; i++ {
			a.Matches(varargParam, processedArgs[i], processedSpans[i])
		}
	}

	if call.SemaFunc == nil {
		call.SemaFunc = &funcTy
	}

	if call.IsTryCall {
		return funcTy.Return
	}

	if call.Catch != nil {
		catch := call.Catch
		catchScope := scope.Child(true)
		errVariable := ast.NewSingleVariable(catch.Name, a.stringType())
		a.AddValue(catchScope, catch.Name.Raw, ast.NewValue(errVariable), catch.Name.Span())

		a.handleBlock(catchScope, &catch.Block)
		a.Matches(funcTy.Return, catch.Block.Type(), catch.Block.Span())
		return funcTy.Return
	}

	if funcTy.Def.Errorable {
		a.Warning(call.Span(), "unhandled error")
		return ast.NewErrorType(call.Span())
	}

	return funcTy.Return
}

func (a *Analysis) handleDotAccess(expr *ast.DotAccess, toIndex *ast.Expr) Type {
	toIndexTy := toIndex.Type()
	if !toIndexTy.IsClass() {
		a.panicf(expr.Span(), "cannot index into non-class type `%s`", toIndexTy.String())
	}

	st := toIndexTy.Class()

	field := expr.Name

	flds := st.Fields
	if fld, ok := flds[field.Raw]; ok {
		if !a.canAccessClassMember(st, fld.IsPublic()) {
			a.Errorf(field.Span(), "field `%s` of class `%s` is private", field.Raw, st.Def.Name.Raw)
		}
		fldSym := ast.NewSymbol(field.Raw, &fld, fld.Def.Name.Span(), true)
		a.AddRef(fldSym, field.Span())
		return fld.Ty
	}

	a.panicf(field.Span(), "no field named `%s` in `%s`", field.Raw, st.Def.Name.Raw)
	panic("unreachable")
}

func (a *Analysis) handleMethodCall(scope *Scope, call *ast.Call, toCall *ast.Expr) Type {
	toCallTy := toCall.Type()
	toCallName := toCallTy.String()

	methods := a.FindMethodsOnType(scope, toCallTy, call.Method.Raw)

	if len(methods) == 0 {
		a.panicf(call.Method.Span(), "no method named `%s` in `%s`", call.Method.Raw, toCallName)
	}
	if len(methods) > 1 {
		a.panicf(call.Method.Span(), "ambiguous method call `%s` in `%s`", call.Method.Raw, toCallName)
	}

	method := methods[0]

	if len(method.Params) < 1 || method.Def.Params[0].Name.Raw != "self" {
		a.panicf(call.Method.Span(), "no method named `%s` in `%s`", call.Method.Raw, toCallName)
	}

	call.SemaFunc = &method

	a.AddRef(method, call.Method.Span())

	methodCopy := method
	methodCopy.Params = method.Params[1:]
	methodCopy.Def.Params = method.Def.Params[1:]

	methodTy := ast.NewSemType(methodCopy, call.Span())
	return a.handleCall(scope, call, methodTy, call.Span())
}

func (a *Analysis) handleUnsafeCast(scope *Scope, as *ast.UnsafeCast) Type {
	a.handleExpr(scope, &as.Expr)
	unsafeCastTy := a.resolveType(scope, as.Type)
	return unsafeCastTy
}

func (a *Analysis) handleElse(scope *Scope, elseOp *ast.Else, expr *ast.Expr) Type {
	exprTy := expr.Type()
	if !exprTy.IsNilable() {
		a.panic(elseOp.Span(), "`else` can only be used on nilables")
	}
	a.handleExpr(scope, &elseOp.Value)
	a.Matches(exprTy.NilableInnerType(), elseOp.Value.Type(), elseOp.Value.Span())
	return elseOp.Value.Type()
}

func (a *Analysis) handleUnwrapNilable(_ *Scope, unwrapOp *ast.UnwrapNilable, expr *ast.Expr) Type {
	exprTy := expr.Type()
	if !exprTy.IsNilable() {
		a.panic(unwrapOp.Span(), "`?` can only be used on nilables")
	}
	return exprTy.NilableInnerType()
}

func (a *Analysis) handleRunRaw(scope *Scope, runRaw *ast.ExprRunRaw) Type {
	code := runRaw.Code.Raw
	if code == "" {
		a.panic(runRaw.Span(), "run raw expression cannot be empty")
	}

	matches := runRaw.GetArgRegex().FindAllStringSubmatch(code, -1)
	maxArgUsed := 0
	usedArgs := make(map[int]bool)

	for _, match := range matches {
		argNum, err := strconv.Atoi(match[1])
		if err != nil {
			a.panicf(runRaw.Span(), "invalid argument number in placeholder: %s", match[0])
		}
		if argNum < 1 {
			a.panicf(runRaw.Span(), "argument numbers must start from 1")
		}
		if usedArgs[argNum] {
			a.panicf(runRaw.Span(), "argument {@%d@} is used more than once", argNum)
		}
		usedArgs[argNum] = true
		if argNum > maxArgUsed {
			maxArgUsed = argNum
		}
	}

	for i := 1; i <= maxArgUsed; i++ {
		if !usedArgs[i] {
			a.panicf(runRaw.Span(), "argument {@%d@} is missing - all arguments from 1 to %d must be used", i, maxArgUsed)
		}
	}

	actualArgs := len(runRaw.Args)
	if actualArgs != maxArgUsed {
		if maxArgUsed == 0 {
			a.panicf(runRaw.Span(), "no argument placeholders found in code, but %d arguments provided", actualArgs)
		} else {
			a.panicf(runRaw.Span(), "expected %d arguments based on placeholders, but got %d", maxArgUsed, actualArgs)
		}
	}

	returnMatches := runRaw.GetReturnRegex().FindAllStringSubmatch(code, -1)
	if len(returnMatches) > 1 {
		a.panicf(runRaw.Span(), "can't have more than one {@RETURN@} placeholder")
	}

	for i := range runRaw.Args {
		a.handleExpr(scope, &runRaw.Args[i])
	}

	returnType := a.nilType()
	if runRaw.ReturnType != nil {
		returnType = a.resolveType(scope, *runRaw.ReturnType)
	}
	return returnType
}

func (a *Analysis) handleVecInit(scope *Scope, vecInit *ast.ExprVecInit) Type {
	var ty Type
	generics := vecInit.Generics
	if len(generics) > 1 {
		a.panic(vecInit.Span(), "vector only accepts up to 1 generic")
	}
	if len(generics) > 0 {
		ty = a.resolveType(scope, generics[0])
	}
	for i := range vecInit.Values {
		val := &vecInit.Values[i]
		a.handleExpr(scope, val)
		if !ty.IsValid() {
			ty = val.Type()
		} else {
			a.Matches(ty, val.Type(), val.Span())
		}
	}
	if !ty.IsValid() {
		a.panic(vecInit.Span(), "cannot infer type of empty vector")
	}
	return a.vecType(ty, vecInit.Span())
}

func (a *Analysis) handleMapInit(scope *Scope, mapInit *ast.ExprMapInit) Type {
	var keyTy, valueTy Type
	generics := mapInit.Generics
	if len(generics) > 2 {
		a.panic(mapInit.Span(), "map only accepts up to 2 generics")
	}
	if len(generics) > 0 {
		keyTy = a.resolveType(scope, generics[0])
	}
	if len(generics) > 1 {
		valueTy = a.resolveType(scope, generics[1])
	}
	for i := range mapInit.Entries {
		field := &mapInit.Entries[i]
		a.handleExpr(scope, &field.Key)
		if !keyTy.IsValid() {
			keyTy = field.Key.Type()
		} else {
			a.Matches(keyTy, field.Key.Type(), field.Key.Span())
		}
		a.handleExpr(scope, &field.Value)
		if !valueTy.IsValid() {
			valueTy = field.Value.Type()
		} else {
			a.Matches(valueTy, field.Value.Type(), field.Value.Span())
		}
	}
	if !keyTy.IsValid() {
		a.panic(mapInit.Span(), "cannot infer types of empty map")
	}
	return a.mapType(keyTy, valueTy, mapInit.Span())
}
