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
	if !isChr(lx.curChr, '"') /* && !isChr(lx.curChr, '\'') */ {
		return nil, nil
	}
	delim := *lx.curChr // Store the delimiter (' or ") to find the end.
	lx.advance()        // Consume the opening delimiter.

	var sb strings.Builder

	// Loop until we find the matching closing delimiter.
	for {
		// Check for end of file or an unescaped newline.
		if lx.curChr == nil {
			return nil, lx.error("unterminated string literal")
		}

		// Found the closing delimiter, the string is complete.
		if *lx.curChr == delim {
			lx.advance() // Consume the closing delimiter.
			return NewTokString(sb.String(), lx.currentSpan()), nil
		}

		// Check for an escape sequence.
		if *lx.curChr == '\\' {
			lx.advance() // Consume '\'.
			if lx.curChr == nil {
				return nil, lx.error("unterminated string literal")
			}

			// If the backslash is followed by a newline, we treat it as a line continuation.
			// This means we skip the newline and any leading whitespace on the next line.
			if *lx.curChr == '\n' || *lx.curChr == '\r' {
				// Handle both LF and CRLF line endings.
				if *lx.curChr == '\r' && isChr(lx.peek(), '\n') {
					lx.advance() // Consume '\r'.
				}
				lx.advance() // Consume '\n' (or standalone '\r').

				// Skip leading whitespace on the next line.
				for isWsChr(lx.curChr) {
					lx.advance()
				}
				continue
			}

			// Now, determine which escape sequence we have.
			switch *lx.curChr {
			case 'a':
				sb.WriteByte('\a')
				lx.advance()
			case 'b':
				sb.WriteByte('\b')
				lx.advance()
			case 'f':
				sb.WriteByte('\f')
				lx.advance()
			case 'n':
				sb.WriteByte('\n')
				lx.advance()
			case 'r':
				sb.WriteByte('\r')
				lx.advance()
			case 't':
				sb.WriteByte('\t')
				lx.advance()
			case 'v':
				sb.WriteByte('\v')
				lx.advance()
			case '\\':
				sb.WriteByte('\\')
				lx.advance()
			case '"':
				sb.WriteByte('"')
				lx.advance()
			case '\'':
				sb.WriteByte('\'')
				lx.advance()
			// The \z escape skips all subsequent whitespace.
			case 'z':
				lx.advance() // Consume 'z'.
				for isWsChr(lx.curChr) {
					lx.advance()
				}

			// Hexadecimal escape: \xXX (e.g., \x41 is 'A').
			case 'x':
				lx.advance() // Consume 'x'.
				var val uint32
				if !isHexDigit(lx.curChr) {
					return nil, lx.error("malformed hexadecimal escape sequence")
				}
				val = hexValue(*lx.curChr) << 4 // First hex digit.
				lx.advance()

				if !isHexDigit(lx.curChr) {
					return nil, lx.error("malformed hexadecimal escape sequence")
				}
				val += hexValue(*lx.curChr) // Second hex digit.
				lx.advance()
				sb.WriteByte(byte(val))

			// Unicode escape: \u{...} (e.g., \u{1F60A} is 😊).
			case 'u':
				lx.advance() // Consume 'u'.
				if !isChr(lx.curChr, '{') {
					return nil, lx.error("malformed Unicode escape sequence, missing '{'")
				}
				lx.advance() // Consume '{'.

				var val uint32
				digitCount := 0
				for {
					if isChr(lx.curChr, '}') {
						break
					}
					if !isHexDigit(lx.curChr) {
						return nil, lx.error("malformed Unicode escape sequence, invalid hex digit")
					}
					digitCount++
					val = (val << 4) | hexValue(*lx.curChr)
					// Check if the codepoint is in the valid Unicode range.
					if val >= 0x110000 {
						return nil, lx.error("invalid Unicode escape sequence, value out of range")
					}
					lx.advance()
				}

				if digitCount == 0 {
					return nil, lx.error("malformed Unicode escape sequence, empty braces")
				}
				lx.advance() // Consume '}'.

				// Surrogate pairs are not valid in this context.
				if val >= 0xD800 && val < 0xE000 {
					return nil, lx.error("invalid Unicode escape sequence, surrogate values are not allowed")
				}
				sb.WriteRune(rune(val))

			// Decimal escape: \ddd (e.g., \65 is 'A').
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				var val uint16
				// Read up to three decimal digits.
				for i := 0; i < 3 && isAsciiDigit(lx.curChr); i++ {
					val = val*10 + uint16(*lx.curChr-'0')
					lx.advance()
				}

				if val > 255 {
					return nil, lx.error("invalid decimal escape sequence, value out of range (0-255)")
				}
				sb.WriteByte(byte(val))

			default:
				return nil, lx.error(fmt.Sprintf("invalid escape sequence: \\%c", *lx.curChr))
			}
			continue // Continue the loop after processing the escape.
		}

		// This is just a regular character, add it to our string.
		sb.WriteRune(*lx.curChr)
		lx.advance()
	}
}

func (lx *lexer) rawString() (Token, *diagnostic) {
	lx.advance() // Consume 'r'

	hashCount := 0
	for isChr(lx.curChr, '#') {
		hashCount++
		lx.advance()
	}

	if !isChr(lx.curChr, '"') {
		return nil, lx.error("expected '\"' to start raw string literal")
	}
	lx.advance() // Consume '\"'.

	var sb strings.Builder

	for {
		if lx.curChr == nil {
			return nil, lx.error("unterminated raw string literal")
		}

		// Check for a potential closing quote.
		if *lx.curChr == '"' {
			lx.advance() // Consume '\"'.

			// See if it's followed by the correct number of hashes.
			endHashCount := 0
			for isChr(lx.curChr, '#') {
				endHashCount++
				lx.advance()
			}

			// If the hash counts match, we've found the end.
			if endHashCount == hashCount {
				return NewTokString(sb.String(), lx.currentSpan()), nil
			}

			// It was just a quote and some hashes inside the string.
			// Append what we consumed to the string content.
			sb.WriteByte('"')
			sb.WriteString(strings.Repeat("#", endHashCount))
			continue // Continue the main loop.
		}

		// It's a regular character, add it to our string.
		sb.WriteRune(*lx.curChr)
		lx.advance()
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
