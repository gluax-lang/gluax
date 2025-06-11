// Package common provides common diagnostic objects.
package common

import (
	protocol "github.com/gluax-lang/lsp"
)

type (
	dSeverity  = protocol.DiagnosticSeverity
	diagnostic = protocol.Diagnostic
)

func NewDiagnostic(severity dSeverity, message string, span Span) *diagnostic {
	return &protocol.Diagnostic{
		Severity: &severity,
		Message:  message,
		Range:    span.ToRange(),
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
