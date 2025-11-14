package ast

import "maps"

// Not doing __len because it won't work with tables in lua

var MetaMethods = map[string]struct{}{
	"__tostring": {},
	"__unm":      {},
	"__eq":       {},
}

var ArithmeticMetaMethods = map[string]struct{}{
	"__add": {},
	"__sub": {},
	"__mul": {},
	"__div": {},
	"__mod": {},
	"__pow": {},
}

var ArithmeticMetaMethodsOps = map[BinaryOp]string{
	BinaryOpAdd:      "__add",
	BinaryOpSub:      "__sub",
	BinaryOpMul:      "__mul",
	BinaryOpDiv:      "__div",
	BinaryOpMod:      "__mod",
	BinaryOpExponent: "__pow",
}

func init() {
	maps.Copy(MetaMethods, ArithmeticMetaMethods)
}
