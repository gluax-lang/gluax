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

#[no__index]
#[sealed]
pub class vec<T> { _priv: nil }

#[no__index]
#[sealed]
pub class map<K, V> { _priv: nil }

#[no_metatable]
#[sealed]
pub class nilable<T> { _priv: nil }

#[no_metatable]
#[no_impl]
#[sealed]
pub class anyfunc { _priv: nil }

#[no_metatable]
#[sealed]
pub class table { _priv: nil }

pub trait Eq {
    func eq(self, other: Self) -> bool {
		@raw("{@RETURN {@1@} == {@2@} @}", self, other) -> bool
	}
}

pub trait Ord {
	func lt(self, other: Self) -> bool;
	func le(self, other: Self) -> bool;
}
`

var builtin = map[string]struct{}{
	"nil":     {},
	"any":     {},
	"bool":    {},
	"number":  {},
	"string":  {},
	"vec":     {},
	"map":     {},
	"nilable": {},
	"anyfunc": {},
	"table":   {},
}

func IsBuiltinType(name string) bool {
	_, exists := builtin[name]
	return exists
}
