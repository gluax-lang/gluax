package ast

import "strings"

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
	LSPString() string
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

func (v Value) LSPString() string {
	if v.data == nil {
		return "<nil>"
	}
	return v.data.LSPString()
}

func (v Value) CanShadow(other Value) bool {
	if v.Kind() == ValFunction || other.Kind() == ValFunction {
		return false
	}
	if v.Kind() == ValVariable && v.Variable().Def.IsItem {
		return false
	}
	if other.Kind() == ValVariable && other.Variable().Def.IsItem {
		return false
	}
	return true
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

func SetValueTo[T valueData](v *Value, data T) {
	if v == nil {
		panic("nil Value pointer")
	}
	v.data = data
}

type Variable struct {
	Def  Let
	N    int // number of the variable in the let statement, Def.Names[N]
	Type SemType
}

func NewVariable(def Let, n int, ty SemType) Variable {
	return Variable{Def: def, N: n, Type: ty}
}

func (v Variable) LSPString() string {
	var sb strings.Builder
	if v.Def.Public {
		sb.WriteString("pub ")
	}
	sb.WriteString("let ")
	sb.WriteString(v.Def.Names[v.N].Raw)
	sb.WriteString(": ")
	sb.WriteString(v.Type.String())
	return sb.String()
}

type SemFunctionParam struct {
	Def  FunctionParam
	Type SemType
}

func NewSemFunctionParam(def FunctionParam, ty SemType) SemFunctionParam {
	return SemFunctionParam{Def: def, Type: ty}
}

func (p SemFunctionParam) LSPString() string {
	return p.Def.Name.Raw + ": " + p.Type.String()
}

type SingleVariable struct {
	Name string
	Ty   SemType
}

func NewSingleVariable(name string, ty SemType) SingleVariable {
	return SingleVariable{Name: name, Ty: ty}
}

func (v SingleVariable) LSPString() string {
	return v.Name + ": " + v.Ty.String()
}
