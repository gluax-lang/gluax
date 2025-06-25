package sema

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
	protocol "github.com/gluax-lang/lsp"
)

type Span = common.Span
type Diagnostic = protocol.Diagnostic
type InlayHint = protocol.InlayHint

type Symbol = ast.Symbol
type Value = ast.Value
type Type = ast.SemType

const SymValue = ast.SymValue
const SymType = ast.SymType
const SymImport = ast.SymImport
const SymTrait = ast.SymTrait

type ImportInfo = ast.SemImport

type SemClass = ast.SemClass
type SemFunction = ast.SemFunction
type SemTuple = ast.SemTuple
type SemVararg = ast.SemVararg
type SemGenericType = ast.SemGenericType

type ClassInstance = ast.ClassInstance

type LSPSymbol = ast.LSPSymbol
