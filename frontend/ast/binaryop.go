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
