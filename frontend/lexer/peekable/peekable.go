// Package peekable provides a peekable iterator over a string
package peekable

import (
	"unicode/utf8"
)

// EOF signals that there are no more runes in the iterator.
const EOF rune = 0

// noWidth is the width assigned when there is no next rune.
const noWidth = 0

// Chars is a peekable iterator over a string
// It normalises Windows line endings ("\r\n") into a single '\n'.
// Stand‑alone '\r' or '\n' runes are returned unchanged.
type Chars struct {
	input   string
	pos     int
	width   int
	next    rune
	hasNext bool
}

// NewPeekableChars creates a new PeekableChars iterator.
func NewPeekableChars(s string) *Chars {
	p := &Chars{input: s}
	p.advance()
	return p
}

// advance moves to the next rune (or sets EOF/noWidth).
// If the next two runes are "\r\n", they are consumed together and
// reported as a single '\n' rune whose width is the combined byte length
// of both runes. This makes "\r\n" appear indistinguishable from a lone
// '\n' to callers, simplifying cross‑platform newline handling.
func (p *Chars) advance() {
	if p.pos >= len(p.input) {
		p.hasNext = false
		p.next = EOF
		p.width = noWidth
		return
	}

	r, w := utf8.DecodeRuneInString(p.input[p.pos:])

	// Normalise Windows line endings: "\r\n" -> '\n'
	if r == '\r' {
		nextPos := p.pos + w
		if nextPos < len(p.input) {
			r2, w2 := utf8.DecodeRuneInString(p.input[nextPos:])
			if r2 == '\n' {
				r = '\n' // represent CRLF as a single LF
				w += w2  // consume both runes
			}
		}
	}

	p.next = r
	p.width = w
	p.hasNext = true
}

// Peek returns a copy of the next rune without consuming it.
// It returns nil if there is no next rune.
func (p *Chars) Peek() *rune {
	if !p.hasNext {
		return nil
	}
	r := p.next
	return &r
}

// Next consumes and returns a copy of the next rune.
// It returns nil if there is no next rune.
func (p *Chars) Next() *rune {
	if !p.hasNext {
		return nil
	}
	r := p.next
	p.pos += p.width
	p.advance()
	return &r
}

func (p *Chars) Pos() int {
	if !p.hasNext {
		return len(p.input) // EOF position is the end of the input
	}
	return p.pos
}

func (p *Chars) Width() int {
	if !p.hasNext {
		return noWidth
	}
	return p.width
}
