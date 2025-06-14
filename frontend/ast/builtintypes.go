package ast

const BuiltinTypes = `
#[no_metatable]
pub class nil { _priv: nil }

#[no_metatable]
#[no_impl]
pub class any { _priv: nil }

#[no_metatable]
pub class bool { _priv: nil }

#[no_metatable]
pub class number { _priv: nil }

#[no_metatable]
pub class string { _priv: nil }

#[no__index]
pub class vec<T> { _priv: nil }

#[no__index]
pub class map<K, V> { _priv: nil }

#[no_metatable]
pub class option<T> { _priv: nil }

#[no_metatable]
#[no_impl]
pub class anyfunc { _priv: nil }

#[no_metatable]
#[no_impl]
pub class table { _priv: nil }
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
