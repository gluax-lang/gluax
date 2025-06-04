package ast

const BuiltinTypes = `
#[no_metatable]
pub struct nil 	{ _priv: nil }
#[no_metatable]
pub struct any { _priv: nil }
#[no_metatable]
pub struct bool { _priv: nil }
#[no_metatable]
pub struct number { _priv: nil }
#[no_metatable]
pub struct string { _priv: nil }
#[no_metatable]
pub struct vec<T> { _priv: nil }
#[no_metatable]
pub struct map<K, V> { _priv: nil }
#[no_metatable]
pub struct option<T> { _priv: nil }
#[no_metatable]
pub struct anyfunc { _priv: nil }
#[no_metatable]
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
