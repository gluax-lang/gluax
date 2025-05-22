package ast

var StdBuiltinTypes = map[string]SemType{}

func IsBuiltinType(name string) bool {
	_, ok := StdBuiltinTypes[name]
	return ok
}

func GetBuiltinType(name string) *SemType {
	if ty, ok := StdBuiltinTypes[name]; ok {
		return &ty
	}
	return nil
}

func AddBuiltinType(name string, ty SemType) {
	StdBuiltinTypes[name] = ty
}
