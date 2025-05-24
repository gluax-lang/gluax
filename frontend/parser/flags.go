package parser

import "strings"

type Flags uint16

const (
	FlagFuncParamNamed Flags = 1 << iota
	FlagFuncParamVarArg
	FlagFuncParamSelf
	FlagTypeTuple
	FlagTypeVarArg
	FlagFuncReturnUnreachable

	FlagAllowUnderscore
)

// Has reports whether f includes all bits in mask.
func (f Flags) Has(mask Flags) bool {
	return f&mask == mask
}

// Set turns on the bits in mask.
func (f Flags) Set(mask Flags) Flags {
	f |= mask
	return f
}

// Clear turns off the bits in mask.
func (f Flags) Clear(mask Flags) Flags {
	f &^= mask
	return f
}

func (f Flags) String() string {
	if f == 0 {
		return "0"
	}
	var parts []string
	if f.Has(FlagFuncParamNamed) {
		parts = append(parts, "FuncParamNamed")
	}
	if f.Has(FlagFuncParamVarArg) {
		parts = append(parts, "FuncParamVarArg")
	}
	if f.Has(FlagFuncParamSelf) {
		parts = append(parts, "FuncParamSelf")
	}
	if f.Has(FlagTypeTuple) {
		parts = append(parts, "TypeTuple")
	}
	if f.Has(FlagTypeVarArg) {
		parts = append(parts, "TypeVarArg")
	}
	if f.Has(FlagAllowUnderscore) {
		parts = append(parts, "AllowUnderscore")
	}
	return strings.Join(parts, "|")
}
