package ast

const BuiltinTypes = `
pub struct nil 	{ _priv: nil }
pub struct any { _priv: nil }
pub struct bool { _priv: nil }
pub struct number { _priv: nil }
pub struct string { _priv: nil }
pub struct vec<T> { _priv: nil }
pub struct map<K, V> { _priv: nil }
pub struct option<T> { _priv: nil }
pub struct anyfunc { _priv: nil }
pub struct table { _priv: nil }
`

var builtin = map[string]struct{}{
	"nil":     {},
	"any":     {},
	"bool":    {},
	"number":  {},
	"string":  {},
	"vec":     {},
	"map":     {},
	"option":  {},
	"anyfunc": {},
	"table":   {},
}

func IsBuiltinType(name string) bool {
	_, exists := builtin[name]
	return exists
}
