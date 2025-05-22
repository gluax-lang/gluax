package ast

type ValueKind uint8

const (
	ValVariable ValueKind = iota
	ValSingleVariable
	ValParameter
	ValFunction
)

type valueData interface {
	ValueKind() ValueKind
	ValueType() SemType
}

func (v Variable) ValueKind() ValueKind { return ValVariable }
func (v Variable) ValueType() SemType   { return v.Type }

func (v SingleVariable) ValueKind() ValueKind { return ValSingleVariable }
func (v SingleVariable) ValueType() SemType   { return v.Ty }

func (f SemFunction) ValueKind() ValueKind { return ValFunction }
func (f SemFunction) ValueType() SemType   { return NewSemType(f, f.Def.Span()) }

func (p SemFunctionParam) ValueKind() ValueKind { return ValParameter }
func (p SemFunctionParam) ValueType() SemType   { return p.Type }

type Value struct {
	data valueData
}

func NewValue[T valueData](data T) Value {
	return Value{data: data}
}

func (v Value) Kind() ValueKind {
	return v.data.ValueKind()
}

func (v Value) CanShadow(other Value) bool {
	if v.Kind() != ValVariable || other.Kind() != ValVariable {
		return false
	}
	return !v.Variable().Def.IsItem && !other.Variable().Def.IsItem
}

func (v Value) Type() SemType {
	return v.data.ValueType()
}

func (v Value) IsVariable() bool {
	return v.Kind() == ValVariable
}

func (v Value) Variable() Variable {
	if v.Kind() != ValVariable {
		panic("not a variable")
	}
	return v.data.(Variable)
}

func (v Value) Parameter() SemFunctionParam {
	if v.Kind() != ValParameter {
		panic("not a parameter")
	}
	return v.data.(SemFunctionParam)
}

func (v Value) Function() SemFunction {
	if v.Kind() != ValFunction {
		panic("not a function")
	}
	return v.data.(SemFunction)
}

func (v Value) SingleVariable() SingleVariable {
	if v.Kind() != ValSingleVariable {
		panic("not a single variable")
	}
	return v.data.(SingleVariable)
}

type Variable struct {
	Def  Let
	N    int // number of the variable in the let statement, Def.Names[N]
	Type SemType
}

func NewVariable(def Let, n int, ty SemType) Variable {
	return Variable{Def: def, N: n, Type: ty}
}

type SemFunctionParam struct {
	Def  FunctionParam
	Type SemType
}

func NewSemFunctionParam(def FunctionParam, ty SemType) SemFunctionParam {
	return SemFunctionParam{Def: def, Type: ty}
}

type SingleVariable struct {
	Name string
	Ty   SemType
}

func NewSingleVariable(name string, ty SemType) SingleVariable {
	return SingleVariable{Name: name, Ty: ty}
}
