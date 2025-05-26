package common

import (
	"fmt"
	"sync/atomic"

	protocol "github.com/gluax-lang/lsp"
)

var globalSpanID uint64

func nextSpanID() uint64 {
	return atomic.AddUint64(&globalSpanID, 1)
}

// Span represents a range in a source file.
type Span struct {
	ID                     uint64
	LineStart, LineEnd     uint32
	ColumnStart, ColumnEnd uint32
	Source                 string // nil == unknown
}

func adjustN(n uint32) uint32 {
	if n <= 1 {
		return 0
	}
	return n - 1
}

func (s Span) ToRange() protocol.Range {
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

func (s Span) String() string {
	return fmt.Sprintf("%d:%d-%d:%d (%s)", s.LineStart, s.ColumnStart, s.LineEnd, s.ColumnEnd, s.Source)
}

// NewDefault Default span (1:1).
func SpanDefault() Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   1,
		LineEnd:     1,
		ColumnStart: 1,
		ColumnEnd:   1,
	}
}

func SpanNew(lineStart, lineEnd, columnStart, columnEnd uint32) Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   lineStart,
		LineEnd:     lineEnd,
		ColumnStart: columnStart,
		ColumnEnd:   columnEnd,
	}
}

func SpanSrc(src string) Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   1,
		LineEnd:     1,
		ColumnStart: 1,
		ColumnEnd:   1,
		Source:      src,
	}
}

// From joins the outer bounds of two spans.
func SpanFrom(start, end Span) Span {
	return Span{
		ID:          nextSpanID(),
		LineStart:   start.LineStart,
		LineEnd:     end.LineEnd,
		ColumnStart: start.ColumnStart,
		ColumnEnd:   end.ColumnEnd,
		Source:      start.Source,
	}
}
