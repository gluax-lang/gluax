package lexer

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/common"
)

// TokString represents a string token.
type TokString struct {
	Raw       string
	span      common.Span
	Multiline bool
}

func (t TokString) isToken() {}

func (t TokString) Span() common.Span {
	return t.span
}

func (t TokString) String() string {
	return t.Raw
}

func (t TokString) Is(_ string) bool {
	return false
}

func (t TokString) AsString() string {
	return ""
}

func NewTokString(s string, span common.Span) TokString {
	return TokString{Raw: s, span: span}
}

/* Lexing */

// This is a mix between luajit and rust

func (lx *lexer) string() (Token, *diagnostic) {
	// A string can start with either a single or double quote.
	if !IsChr(lx.CurChr, '"') /* && !isChr(lx.curChr, '\'') */ {
		return nil, nil
	}
	delim := *lx.CurChr // Store the delimiter (' or ") to find the end.
	lx.Advance()        // Consume the opening delimiter.

	var sb strings.Builder

	// Loop until we find the matching closing delimiter.
	for {
		// Check for end of file or an unescaped newline.
		if lx.CurChr == nil {
			return nil, lx.Error("unterminated string literal")
		}

		// Found the closing delimiter, the string is complete.
		if *lx.CurChr == delim {
			lx.Advance() // Consume the closing delimiter.
			return NewTokString(sb.String(), lx.CurrentSpan()), nil
		}

		// Check for an escape sequence.
		if *lx.CurChr == '\\' {
			lx.Advance() // Consume '\'.
			if lx.CurChr == nil {
				return nil, lx.Error("unterminated string literal")
			}

			// If the backslash is followed by a newline, we treat it as a line continuation.
			// This means we skip the newline and any leading whitespace on the next line.
			if *lx.CurChr == '\n' || *lx.CurChr == '\r' {
				// Handle both LF and CRLF line endings.
				if *lx.CurChr == '\r' && IsChr(lx.Peek(), '\n') {
					lx.Advance() // Consume '\r'.
				}
				lx.Advance() // Consume '\n' (or standalone '\r').

				// Skip leading whitespace on the next line.
				for IsWsChr(lx.CurChr) {
					lx.Advance()
				}
				continue
			}

			// Now, determine which escape sequence we have.
			switch *lx.CurChr {
			case 'a':
				sb.WriteByte('\a')
				lx.Advance()
			case 'b':
				sb.WriteByte('\b')
				lx.Advance()
			case 'f':
				sb.WriteByte('\f')
				lx.Advance()
			case 'n':
				sb.WriteByte('\n')
				lx.Advance()
			case 'r':
				sb.WriteByte('\r')
				lx.Advance()
			case 't':
				sb.WriteByte('\t')
				lx.Advance()
			case 'v':
				sb.WriteByte('\v')
				lx.Advance()
			case '\\':
				sb.WriteByte('\\')
				lx.Advance()
			case '"':
				sb.WriteByte('"')
				lx.Advance()
			case '\'':
				sb.WriteByte('\'')
				lx.Advance()
			// The \z escape skips all subsequent whitespace.
			case 'z':
				lx.Advance() // Consume 'z'.
				for IsWsChr(lx.CurChr) {
					lx.Advance()
				}

			// Hexadecimal escape: \xXX (e.g., \x41 is 'A').
			case 'x':
				lx.Advance() // Consume 'x'.
				var val uint32
				if !isHexDigit(lx.CurChr) {
					return nil, lx.Error("malformed hexadecimal escape sequence")
				}
				val = hexValue(*lx.CurChr) << 4 // First hex digit.
				lx.Advance()

				if !isHexDigit(lx.CurChr) {
					return nil, lx.Error("malformed hexadecimal escape sequence")
				}
				val += hexValue(*lx.CurChr) // Second hex digit.
				lx.Advance()
				sb.WriteByte(byte(val))

			// Unicode escape: \u{...} (e.g., \u{1F60A} is 😊).
			case 'u':
				lx.Advance() // Consume 'u'.
				if !IsChr(lx.CurChr, '{') {
					return nil, lx.Error("malformed Unicode escape sequence, missing '{'")
				}
				lx.Advance() // Consume '{'.

				var val uint32
				digitCount := 0
				for {
					if IsChr(lx.CurChr, '}') {
						break
					}
					if !isHexDigit(lx.CurChr) {
						return nil, lx.Error("malformed Unicode escape sequence, invalid hex digit")
					}
					digitCount++
					val = (val << 4) | hexValue(*lx.CurChr)
					// Check if the codepoint is in the valid Unicode range.
					if val >= 0x110000 {
						return nil, lx.Error("invalid Unicode escape sequence, value out of range")
					}
					lx.Advance()
				}

				if digitCount == 0 {
					return nil, lx.Error("malformed Unicode escape sequence, empty braces")
				}
				lx.Advance() // Consume '}'.

				// Surrogate pairs are not valid in this context.
				if val >= 0xD800 && val < 0xE000 {
					return nil, lx.Error("invalid Unicode escape sequence, surrogate values are not allowed")
				}
				sb.WriteRune(rune(val))

			// Decimal escape: \ddd (e.g., \65 is 'A').
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				var val uint16
				// Read up to three decimal digits.
				for i := 0; i < 3 && isAsciiDigit(lx.CurChr); i++ {
					val = val*10 + uint16(*lx.CurChr-'0')
					lx.Advance()
				}

				if val > 255 {
					return nil, lx.Error("invalid decimal escape sequence, value out of range (0-255)")
				}
				sb.WriteByte(byte(val))

			default:
				return nil, lx.Error(fmt.Sprintf("invalid escape sequence: \\%c", *lx.CurChr))
			}
			continue // Continue the loop after processing the escape.
		}

		// This is just a regular character, add it to our string.
		sb.WriteRune(*lx.CurChr)
		lx.Advance()
	}
}

func (lx *lexer) rawString() (Token, *diagnostic) {
	lx.Advance() // Consume 'r'

	hashCount := 0
	for IsChr(lx.CurChr, '#') {
		hashCount++
		lx.Advance()
	}

	if !IsChr(lx.CurChr, '"') {
		return nil, lx.Error("expected '\"' to start raw string literal")
	}
	lx.Advance() // Consume '\"'.

	var sb strings.Builder

	for {
		if lx.CurChr == nil {
			return nil, lx.Error("unterminated raw string literal")
		}

		// Check for a potential closing quote.
		if *lx.CurChr == '"' {
			lx.Advance() // Consume '\"'.
			if hashCount == 0 {
				return NewTokString(sb.String(), lx.CurrentSpan()), nil
			}

			// See if it's followed by the correct number of hashes.
			endHashCount := 0
			for IsChr(lx.CurChr, '#') {
				endHashCount++
				lx.Advance()
			}

			// If the hash counts match, we've found the end.
			if endHashCount == hashCount {
				return NewTokString(sb.String(), lx.CurrentSpan()), nil
			}

			// It was just a quote and some hashes inside the string.
			// Append what we consumed to the string content.
			sb.WriteByte('"')
			sb.WriteString(strings.Repeat("#", endHashCount))
			continue // Continue the main loop.
		}

		// It's a regular character, add it to our string.
		sb.WriteRune(*lx.CurChr)
		lx.Advance()
	}
}

// isHexDigit checks if a rune is a hexadecimal digit.
func isHexDigit(r *rune) bool {
	if r == nil {
		return false
	}
	return ('0' <= *r && *r <= '9') || ('a' <= *r && *r <= 'f') || ('A' <= *r && *r <= 'F')
}

// hexValue returns the integer value of a hex digit rune.
// It assumes the rune has already been verified as a hex digit.
func hexValue(r rune) uint32 {
	switch {
	case '0' <= r && r <= '9':
		return uint32(r - '0')
	case 'a' <= r && r <= 'f':
		return uint32(r - 'a' + 10)
	case 'A' <= r && r <= 'F':
		return uint32(r - 'A' + 10)
	}
	return 0 // Should not be reached
}
