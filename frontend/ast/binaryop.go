package ast

type BinaryOp int

const (
	BinaryOpInvalid BinaryOp = iota
	// BinaryOpLogicalOr is `||`
	BinaryOpLogicalOr
	// BinaryOpLogicalAnd is `&&`
	BinaryOpLogicalAnd

	// BinaryOpLess is `<`
	BinaryOpLess
	// BinaryOpGreater is `>`
	BinaryOpGreater
	// BinaryOpLessEqual is `<=`
	BinaryOpLessEqual
	// BinaryOpGreaterEqual is `>=`
	BinaryOpGreaterEqual
	// BinaryOpEqual is `==`
	BinaryOpEqual
	// BinaryOpNotEqual is `!=`
	BinaryOpNotEqual

	// BinaryOpBitwiseOr is `|`
	BinaryOpBitwiseOr
	// BinaryOpBitwiseAnd is `&`
	BinaryOpBitwiseAnd
	// BinaryOpBitwiseXor is `^`
	BinaryOpBitwiseXor
	// BinaryOpBitwiseLeftShift is `<<`
	BinaryOpBitwiseLeftShift
	// BinaryOpBitwiseRightShift is `>>`
	BinaryOpBitwiseRightShift

	// BinaryOpConcat is `..`
	BinaryOpConcat

	// BinaryOpAdd is `+`
	BinaryOpAdd
	// BinaryOpSub is `-`
	BinaryOpSub
	// BinaryOpMul is `*`
	BinaryOpMul
	// BinaryOpDiv is `/`
	BinaryOpDiv
	// BinaryOpMod is `%`
	BinaryOpMod
	// BinaryOpExponent is `**`
	BinaryOpExponent
)

func (op BinaryOp) String() string {
	switch op {
	case BinaryOpInvalid:
		return "invalid"

	case BinaryOpLogicalOr:
		return "||"
	case BinaryOpLogicalAnd:
		return "&&"

	case BinaryOpLess:
		return "<"
	case BinaryOpGreater:
		return ">"
	case BinaryOpLessEqual:
		return "<="
	case BinaryOpGreaterEqual:
		return ">="
	case BinaryOpEqual:
		return "=="
	case BinaryOpNotEqual:
		return "!="

	case BinaryOpBitwiseOr:
		return "|"
	case BinaryOpBitwiseAnd:
		return "&"
	case BinaryOpBitwiseXor:
		return "^"
	case BinaryOpBitwiseLeftShift:
		return "<<"
	case BinaryOpBitwiseRightShift:
		return ">>"

	case BinaryOpConcat:
		return ".."

	case BinaryOpAdd:
		return "+"
	case BinaryOpSub:
		return "-"
	case BinaryOpMul:
		return "*"
	case BinaryOpDiv:
		return "/"
	case BinaryOpMod:
		return "%"
	case BinaryOpExponent:
		return "**"
	}

	return "invalid"
}
