package ast

import "maps"

var MetaMethods = map[string]struct{}{
	"__tostring": {},
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
