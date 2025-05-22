// Package common provides common diagnostic objects.
package common

import (
	protocol "github.com/gluax-lang/lsp"
)

type (
	dSeverity  = protocol.DiagnosticSeverity
	diagnostic = protocol.Diagnostic
)

func adjustN(n uint32) uint32 {
	if n <= 1 {
		return 0
	}
	return n - 1
}

func spanToRange(s Span) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      adjustN(s.LineStart),
			Character: adjustN(s.ColumnStart),
		},
		End: protocol.Position{
			Line:      adjustN(s.LineEnd),
			Character: s.ColumnEnd,
		},
	}
}

func NewDiagnostic(severity dSeverity, message string, span Span) *diagnostic {
	return &protocol.Diagnostic{
		Severity: &severity,
		Message:  message,
		Range:    spanToRange(span),
	}
}

func ErrorDiag(msg string, span Span) *diagnostic {
	return NewDiagnostic(protocol.DiagnosticSeverityError,
		msg, span)
}

func PanicDiag(msg string, span Span) {
	panic(ErrorDiag(msg, span))
}

func WarningDiag(msg string, span Span) *diagnostic {
	return NewDiagnostic(protocol.DiagnosticSeverityWarning,
		msg, span)
}
