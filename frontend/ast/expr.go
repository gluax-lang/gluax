package ast

import (
	"regexp"

	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type ExprKind uint8

const (
	_ ExprKind = iota
	ExprKindNil
	ExprKindBool
	ExprKindNumber
	ExprKindString
	ExprKindVararg
	ExprKindFunction
	ExprKindPath
	ExprKindBinary
	ExprKindUnary
	ExprKindPostfix
	ExprKindBlock
	ExprKindIf
	ExprKindWhile
	ExprKindLoop
	ExprKindParenthesized
	ExprKindStructInit
	ExprKindPathCall
	ExprKindTuple
	ExprKindUnsafeCast
	ExprKindRunRaw
	ExprKindVecInit
	ExprKindMapInit
)

func (k ExprKind) String() string {
	switch k {
	case ExprKindNil:
		return "nil"
	case ExprKindBool:
		return "bool"
	case ExprKindNumber:
		return "number"
	case ExprKindString:
		return "string"
	case ExprKindVararg:
		return "..."
	case ExprKindIf:
		return "if"
	case ExprKindWhile:
		return "while"
	case ExprKindLoop:
		return "loop"
	case ExprKindBlock:
		return "block"
	case ExprKindParenthesized:
		return "parenthesized"
	case ExprKindStructInit:
		return "struct init"
	case ExprKindPathCall:
		return "struct static call"
	case ExprKindFunction:
		return "function"
	case ExprKindPath:
		return "path"
	case ExprKindBinary:
		return "binary"
	case ExprKindUnary:
		return "unary"
	case ExprKindPostfix:
		return "postfix"
	case ExprKindTuple:
		return "tuple"
	case ExprKindUnsafeCast:
		return "unsafe cast"
	case ExprKindRunRaw:
		return "run lua"
	case ExprKindVecInit:
		return "vec init"
	case ExprKindMapInit:
		return "map init"
	default:
		panic("unreachable")
	}
}

type exprData interface {
	ExprKind() ExprKind
	Span() common.Span
}

type Expr struct {
	data exprData
	sem  SemType
	// indicates whether this expression is used as a condition
	// this is used for option types, to always evaluate them to bool
	// lua always returns the last value of a condition, nil or not (s = a and b)
	// so it will break our code because it will be expecting a bool
	AsCond bool
}

func NewExpr[T exprData](data T) Expr {
	return Expr{data: data}
}

func (e Expr) Kind() ExprKind {
	return e.data.ExprKind()
}

func (e Expr) Data() exprData {
	return e.data
}

func (e Expr) Type() SemType {
	return e.sem
}

func (e *Expr) SetType(sem SemType) {
	e.sem = sem
}

func (e Expr) Span() common.Span {
	return e.data.Span()
}

func (e *Expr) Path() *Path {
	if e.Kind() != ExprKindPath {
		panic("not a path")
	}
	return e.data.(*Path)
}

func (e *Expr) Postfix() *ExprPostfix {
	if e.Kind() != ExprKindPostfix {
		panic("not a postfix")
	}
	return e.data.(*ExprPostfix)
}

func (e *Expr) Tuple() *ExprTuple {
	if e.Kind() != ExprKindTuple {
		panic("not a tuple")
	}
	return e.data.(*ExprTuple)
}

func (e *Expr) Binary() *ExprBinary {
	if e.Kind() != ExprKindBinary {
		panic("not a binary")
	}
	return e.data.(*ExprBinary)
}

func (e *Expr) Unary() *ExprUnary {
	if e.Kind() != ExprKindUnary {
		panic("not a unary")
	}
	return e.data.(*ExprUnary)
}

func (e *Expr) Block() *Block {
	if e.Kind() != ExprKindBlock {
		panic("not a block")
	}
	return e.data.(*Block)
}

func (e *Expr) While() *ExprWhile {
	if e.Kind() != ExprKindWhile {
		panic("not a while")
	}
	return e.data.(*ExprWhile)
}

func (e *Expr) Loop() *ExprLoop {
	if e.Kind() != ExprKindLoop {
		panic("not a loop")
	}
	return e.data.(*ExprLoop)
}

func (e *Expr) If() *ExprIf {
	if e.Kind() != ExprKindIf {
		panic("not an if")
	}
	return e.data.(*ExprIf)
}

func (e *Expr) Parenthesized() *ExprParenthesized {
	if e.Kind() != ExprKindParenthesized {
		panic("not a parenthesized")
	}
	return e.data.(*ExprParenthesized)
}

func (e *Expr) Function() *Function {
	if e.Kind() != ExprKindFunction {
		panic("not a function")
	}
	return e.data.(*Function)
}

func (e *Expr) StructInit() *ExprStructInit {
	if e.Kind() != ExprKindStructInit {
		panic("not a struct init")
	}
	return e.data.(*ExprStructInit)
}

func (e *Expr) UnsafeCast() *UnsafeCast {
	if e.Kind() != ExprKindUnsafeCast {
		panic("not an unsafe cast")
	}
	return e.data.(*UnsafeCast)
}

func (e *Expr) Bool() bool {
	if e.Kind() != ExprKindBool {
		panic("not a bool")
	}
	return e.data.(*ExprBool).Value
}

func (e *Expr) Number() lexer.TokNumber {
	if e.Kind() != ExprKindNumber {
		panic("not a number")
	}
	return e.data.(*ExprNumber).Value
}

func (e *Expr) String() lexer.TokString {
	if e.Kind() != ExprKindString {
		panic("not a string")
	}
	return e.data.(*ExprString).Value
}

func (e *Expr) IsBlock() bool {
	switch e.Kind() {
	case ExprKindBlock, ExprKindLoop, ExprKindWhile, ExprKindIf:
		return true
	default:
		return false
	}
}

/* Nil */

type ExprNil struct {
	span common.Span
}

func NewNilExpr(span common.Span) Expr {
	return NewExpr(&ExprNil{span: span})
}

func (n *ExprNil) ExprKind() ExprKind { return ExprKindNil }

func (n *ExprNil) Span() common.Span {
	return n.span
}

/* Bool */

type ExprBool struct {
	Value bool
	span  common.Span
}

func NewBoolExpr(b lexer.Token) Expr {
	return NewExpr(&ExprBool{Value: b.AsString() == "true", span: b.Span()})
}

func (b *ExprBool) ExprKind() ExprKind { return ExprKindBool }

func (b *ExprBool) Span() common.Span {
	return b.span
}

/* Number */

type ExprNumber struct {
	Value lexer.TokNumber
}

func NewNumberExpr(n lexer.TokNumber) Expr {
	return NewExpr(&ExprNumber{Value: n})
}

func (n *ExprNumber) ExprKind() ExprKind { return ExprKindNumber }

func (n *ExprNumber) Span() common.Span {
	return n.Value.Span()
}

/* String */

type ExprString struct {
	Value lexer.TokString
}

func NewStringExpr(s lexer.TokString) Expr {
	return NewExpr(&ExprString{Value: s})
}

func (s *ExprString) ExprKind() ExprKind { return ExprKindString }

func (s *ExprString) Span() common.Span {
	return s.Value.Span()
}

/* Vararg */

type ExprVararg struct {
	span common.Span
}

func NewVarargExpr(span common.Span) Expr {
	return NewExpr(&ExprVararg{span: span})
}

func (v *ExprVararg) ExprKind() ExprKind { return ExprKindVararg }

func (v *ExprVararg) Span() common.Span {
	return v.span
}

/* If */

type GuardedBlock struct {
	Cond Expr
	Then Block
}

func NewGuardedBlock(cond Expr, then Block) GuardedBlock {
	return GuardedBlock{Cond: cond, Then: then}
}

type ExprIf struct {
	Main     GuardedBlock
	Branches []GuardedBlock
	Else     *Block
	span     common.Span
}

func NewIfExpr(main GuardedBlock, branches []GuardedBlock, elseBlock *Block, span common.Span) Expr {
	return NewExpr(&ExprIf{Main: main, Branches: branches, Else: elseBlock, span: span})
}

func (i *ExprIf) ExprKind() ExprKind { return ExprKindIf }

func (i *ExprIf) Span() common.Span {
	return i.span
}

/* While */

type ExprWhile struct {
	Label *Ident
	Cond  Expr
	Body  Block
	span  common.Span
}

func NewWhileExpr(label *Ident, cond Expr, body Block, span common.Span) Expr {
	return NewExpr(&ExprWhile{Label: label, Cond: cond, Body: body, span: span})
}

func (w *ExprWhile) ExprKind() ExprKind { return ExprKindWhile }

func (w *ExprWhile) Span() common.Span {
	return w.span
}

/* Loop */

type ExprLoop struct {
	Label *Ident
	Body  Block
	span  common.Span
}

func NewLoopExpr(label *Ident, body Block, span common.Span) Expr {
	return NewExpr(&ExprLoop{Label: label, Body: body, span: span})
}

func (l *ExprLoop) ExprKind() ExprKind { return ExprKindLoop }

func (l *ExprLoop) Span() common.Span {
	return l.span
}

func (l *ExprLoop) SetSem(bodySem SemType) {
	l.bodySem = bodySem
}

func (l ExprLoop) SemBody() SemType {
	return l.bodySem
}

/* Binary */

type ExprBinary struct {
	Op    BinaryOp
	Left  Expr
	Right Expr
	span  common.Span
}

func (b ExprBinary) IsShortCircuit() bool {
	switch b.Op {
	case BinaryOpLogicalOr, BinaryOpLogicalAnd:
		return true
	default:
		return false
	}
}

func NewBinaryExpr(left Expr, op BinaryOp, right Expr, span common.Span) Expr {
	return NewExpr(&ExprBinary{Left: left, Op: op, Right: right, span: span})
}

func (b *ExprBinary) ExprKind() ExprKind { return ExprKindBinary }

func (b *ExprBinary) Span() common.Span {
	return b.span
}

/* Unary */

type ExprUnary struct {
	Op    UnaryOp
	Value Expr
	span  common.Span
}

func NewUnaryExpr(op UnaryOp, value Expr, span common.Span) Expr {
	return NewExpr(&ExprUnary{Op: op, Value: value, span: span})
}

func (u *ExprUnary) ExprKind() ExprKind { return ExprKindUnary }

func (u *ExprUnary) Span() common.Span {
	return u.span
}

/* Postfix */

type ExprPostfix struct {
	Left Expr
	Op   PostfixOp
	span common.Span
}

func NewPostfixExpr(left Expr, op PostfixOp, span common.Span) Expr {
	return NewExpr(&ExprPostfix{Left: left, Op: op, span: span})
}

func (p *ExprPostfix) ExprKind() ExprKind { return ExprKindPostfix }

func (p *ExprPostfix) Span() common.Span {
	return p.span
}

/* Parenthesized */

type ExprParenthesized struct {
	Value Expr
	span  common.Span
}

func NewParenthesizedExpr(value Expr, span common.Span) Expr {
	return NewExpr(&ExprParenthesized{Value: value, span: span})
}

func (p *ExprParenthesized) ExprKind() ExprKind { return ExprKindParenthesized }

func (p *ExprParenthesized) Span() common.Span {
	return p.span
}

/* Tuple */

type ExprTuple struct {
	Values []Expr
	span   common.Span
}

func NewTupleExpr(values []Expr, span common.Span) Expr {
	return NewExpr(&ExprTuple{Values: values, span: span})
}

func (t *ExprTuple) ExprKind() ExprKind { return ExprKindTuple }

func (t *ExprTuple) Span() common.Span {
	return t.span
}

/* Struct Init */

type ExprStructField struct {
	Name  Ident
	Value Expr
}

type ExprStructInit struct {
	Name     Path
	Generics []Type
	Fields   []ExprStructField
	span     common.Span
}

func NewStructInit(name Path, generics []Type, fields []ExprStructField, span common.Span) Expr {
	return NewExpr(&ExprStructInit{Name: name, Generics: generics, Fields: fields, span: span})
}

func (s *ExprStructInit) ExprKind() ExprKind { return ExprKindStructInit }

func (s *ExprStructInit) Span() common.Span {
	return s.span
}

/* UnsafeCast (unsafe_cast) */

type UnsafeCast struct {
	Expr Expr
	Type Type
	span common.Span
}

func NewUnsafeCast(expr Expr, ty Type, span common.Span) Expr {
	return NewExpr(&UnsafeCast{Expr: expr, Type: ty, span: span})
}

func (a *UnsafeCast) ExprKind() ExprKind { return ExprKindUnsafeCast }

func (a *UnsafeCast) Span() common.Span {
	return a.span
}

/* Run Lua (@lua) */

type ExprRunRaw struct {
	Code       lexer.TokString
	Args       []Expr
	ReturnType Type
	span       common.Span
}

func NewRunRawExpr(code lexer.TokString, args []Expr, returnType Type, span common.Span) Expr {
	return NewExpr(&ExprRunRaw{Code: code, Args: args, ReturnType: returnType, span: span})
}

func (e *Expr) RunRaw() *ExprRunRaw {
	if e.Kind() != ExprKindRunRaw {
		panic("not a run lua expression")
	}
	return e.data.(*ExprRunRaw)
}

func (r *ExprRunRaw) ExprKind() ExprKind { return ExprKindRunRaw }

func (r *ExprRunRaw) Span() common.Span {
	return r.span
}

var runRawArgRegex = regexp.MustCompile(`\{@(\d+)@\}`)
var runRawTempRegex = regexp.MustCompile(`\{@TEMP(\d+)@\}`)
var runRawReturnRegex = regexp.MustCompile(`\{@RETURN\s+(.+?)@\}`)

func (r *ExprRunRaw) GetArgRegex() *regexp.Regexp    { return runRawArgRegex }
func (r *ExprRunRaw) GetTempRegex() *regexp.Regexp   { return runRawTempRegex }
func (r *ExprRunRaw) GetReturnRegex() *regexp.Regexp { return runRawReturnRegex }

/* Vec Init */
type ExprVecInit struct {
	Generics []Type
	Values   []Expr
	span     common.Span
}

func NewVecInitExpr(generics []Type, values []Expr, span common.Span) Expr {
	return NewExpr(&ExprVecInit{Generics: generics, Values: values, span: span})
}

func (e *Expr) VecInit() *ExprVecInit {
	if e.Kind() != ExprKindVecInit {
		panic("not a vec init expression")
	}
	return e.data.(*ExprVecInit)
}

func (v *ExprVecInit) ExprKind() ExprKind { return ExprKindVecInit }
func (v *ExprVecInit) Span() common.Span {
	return v.span
}

/* Map Init */

type ExprMapEntry struct {
	Key   Expr
	Value Expr
}

type ExprMapInit struct {
	Generics []Type
	Entries  []ExprMapEntry
	span     common.Span
}

func NewMapInitExpr(generics []Type, entries []ExprMapEntry, span common.Span) Expr {
	return NewExpr(&ExprMapInit{Generics: generics, Entries: entries, span: span})
}

func (e *Expr) MapInit() *ExprMapInit {
	if e.Kind() != ExprKindMapInit {
		panic("not a map init expression")
	}
	return e.data.(*ExprMapInit)
}

func (m *ExprMapInit) ExprKind() ExprKind { return ExprKindMapInit }

func (m *ExprMapInit) Span() common.Span {
	return m.span
}
