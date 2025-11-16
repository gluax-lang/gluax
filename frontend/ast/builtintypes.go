package ast

const BuiltinTypes = `
	#[no_metatable]
	#[sealed]
	pub class nil { _priv: nil }

	#[no_metatable]
	#[no_impl]
	#[sealed]
	pub class any { _priv: nil }

	#[no_metatable]
	#[sealed]
	pub class bool { _priv: nil }

	#[no_metatable]
	#[sealed]
	pub class number { _priv: nil }

	#[no_metatable]
	#[sealed]
	pub class string { _priv: nil }

	#[no_metatable]
	#[no_impl]
	#[sealed]
	pub class anyfunc { _priv: nil }

	#[no_metatable]
	#[sealed]
	pub class table { _priv: nil }

	#[no_metatable]
	#[no_impl]
	#[sealed]
	pub class vec_T { _priv: nil }

	#[no_metatable]
	#[no_impl]
	#[sealed]
	pub class map_K { _priv: nil }

	#[no_metatable]
	#[no_impl]
	#[sealed]
	pub class map_V { _priv: nil }
`

var builtin = map[string]struct{}{
	"nil":     {},
	"any":     {},
	"bool":    {},
	"number":  {},
	"string":  {},
	"anyfunc": {},
	"table":   {},
	"vec_T":   {},
	"map_K":   {},
	"map_V":   {},
}

func IsBuiltinType(name string) bool {
	_, exists := builtin[name]
	return exists
}
