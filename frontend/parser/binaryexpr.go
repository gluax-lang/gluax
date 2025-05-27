package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

type associativity uint8

const (
	assocLeft associativity = iota
	assocRight
)

func getBinaryOperatorPrecedence(op string) (int, associativity, ast.BinaryOp, bool) {
	switch op {
	// Logical
	case "||":
		return 1, assocLeft, ast.BinaryOpLogicalOr, true
	case "&&":
		return 2, assocLeft, ast.BinaryOpLogicalAnd, true

	// Relational
	case "<":
		return 3, assocLeft, ast.BinaryOpLess, true
	case ">":
		return 3, assocLeft, ast.BinaryOpGreater, true
	case "<=":
		return 3, assocLeft, ast.BinaryOpLessEqual, true
	case ">=":
		return 3, assocLeft, ast.BinaryOpGreaterEqual, true
	case "==":
		return 3, assocLeft, ast.BinaryOpEqual, true
	case "!=":
		return 3, assocLeft, ast.BinaryOpNotEqual, true

	// Bitwise
	case "|":
		return 4, assocLeft, ast.BinaryOpBitwiseOr, true
	case "^":
		return 5, assocLeft, ast.BinaryOpBitwiseXor, true
	case "&":
		return 6, assocLeft, ast.BinaryOpBitwiseAnd, true

	// Shifts
	case "<<":
		return 7, assocLeft, ast.BinaryOpBitwiseLeftShift, true
	case ">>":
		return 7, assocLeft, ast.BinaryOpBitwiseRightShift, true

	// Concatenation
	case "..":
		return 8, assocRight, ast.BinaryOpConcat, true

	// Add / sub
	case "+":
		return 9, assocLeft, ast.BinaryOpAdd, true
	case "-":
		return 9, assocLeft, ast.BinaryOpSub, true

	// Mul / div / mod
	case "*":
		return 10, assocLeft, ast.BinaryOpMul, true
	case "/":
		return 10, assocLeft, ast.BinaryOpDiv, true
	case "%":
		return 10, assocLeft, ast.BinaryOpMod, true

	// Exponentiation (right-assoc)
	case "**":
		return 12, assocRight, ast.BinaryOpExponent, true
	}

	return 0, assocLeft, 0, false
}

func (p *parser) parseBinaryExpr(ctx ExprCtx, minPrec int) ast.Expr {
	left := p.parseUnsafeCast(ctx)

	for {
		isShiftRight := p.Token.Is(">") && p.peek().Is(">")
		isShiftLeft := p.Token.Is("<") && p.peek().Is("<")

		var opStr string
		if isShiftRight {
			opStr = ">>"
		} else if isShiftLeft {
			opStr = "<<"
		} else {
			opStr = p.Token.AsString()
		}

		prec, assoc, binOp, ok := getBinaryOperatorPrecedence(opStr)
		if !ok || prec < minPrec {
			// Not an operator we care about, or lower precedence -> done.
			break
		}

		if isShiftRight || isShiftLeft {
			p.advance() // first '>' or '<'
			p.advance() // second '>' or '<'
		} else {
			p.advance() // single-token operator
		}

		nextMinPrec := prec
		if assoc == assocLeft {
			nextMinPrec = prec + 1
		}
		right := p.parseBinaryExpr(ctx, nextMinPrec)

		span := SpanFrom(left.Span(), right.Span())
		left = ast.NewBinaryExpr(left, binOp, right, span)
	}

	return left
}

func (p *parser) parseUnsafeCast(ctx ExprCtx) ast.Expr {
	expr := p.parseUnaryExpr(ctx)
	for p.Token.Is("unsafe_cast_as") {
		spanStart := expr.Span()
		p.advance() // consume 'unsafe_cast_as'
		ty := p.parseType()
		span := SpanFrom(spanStart, p.prevSpan())
		expr = ast.NewUnsafeCast(expr, ty, span)
	}
	return expr
}
