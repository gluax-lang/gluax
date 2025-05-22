package ast

type UnaryOp int

const (
	_ UnaryOp = iota
	// UnaryOpNegate is `-a`
	UnaryOpNegate
	// UnaryOpNot is `!a`
	UnaryOpNot
	// UnaryOpBitwiseNot is `~a`
	UnaryOpBitwiseNot
	// UnaryOpLength is `#a`
	UnaryOpLength
)
