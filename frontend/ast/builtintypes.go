package ast

var StdBuiltinTypes = map[string]SemType{}

func IsBuiltinType(name string) bool {
	_, ok := StdBuiltinTypes[name]
	return ok
}

func AddBuiltinType(name string, ty SemType) {
	StdBuiltinTypes[name] = ty
}
