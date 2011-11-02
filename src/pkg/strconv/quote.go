// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strconv

import (
	"bytes"
	"strings"
	"unicode"
	"utf8"
)

const lowerhex = "0123456789abcdef"

func quoteWith(s string, quote byte, ASCIIonly bool) string {
	var buf bytes.Buffer
	buf.WriteByte(quote)
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf.WriteString(`\x`)
			buf.WriteByte(lowerhex[s[0]>>4])
			buf.WriteByte(lowerhex[s[0]&0xF])
			continue
		}
		if r == rune(quote) || r == '\\' { // always backslashed
			buf.WriteByte('\\')
			buf.WriteByte(byte(r))
			continue
		}
		if ASCIIonly {
			if r <= unicode.MaxASCII && unicode.IsPrint(r) {
				buf.WriteRune(r)
				continue
			}
		} else if unicode.IsPrint(r) {
			buf.WriteRune(r)
			continue
		}
		switch r {
		case '\a':
			buf.WriteString(`\a`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case '\v':
			buf.WriteString(`\v`)
		default:
			switch {
			case r < ' ':
				buf.WriteString(`\x`)
				buf.WriteByte(lowerhex[s[0]>>4])
				buf.WriteByte(lowerhex[s[0]&0xF])
			case r > unicode.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				buf.WriteString(`\u`)
				for s := 12; s >= 0; s -= 4 {
					buf.WriteByte(lowerhex[r>>uint(s)&0xF])
				}
			default:
				buf.WriteString(`\U`)
				for s := 28; s >= 0; s -= 4 {
					buf.WriteByte(lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	buf.WriteByte(quote)
	return buf.String()

}

// Quote returns a double-quoted Go string literal representing s.  The
// returned string uses Go escape sequences (\t, \n, \xFF, \u0100) for
// control characters and non-printable characters as defined by
// unicode.IsPrint.
func Quote(s string) string {
	return quoteWith(s, '"', false)
}

// QuoteToASCII returns a double-quoted Go string literal representing s.
// The returned string uses Go escape sequences (\t, \n, \xFF, \u0100) for
// non-ASCII characters and non-printable characters as defined by
// unicode.IsPrint.
func QuoteToASCII(s string) string {
	return quoteWith(s, '"', true)
}

// QuoteRune returns a single-quoted Go character literal representing the
// rune.  The returned string uses Go escape sequences (\t, \n, \xFF, \u0100)
// for control characters and non-printable characters as defined by
// unicode.IsPrint.
func QuoteRune(rune int) string {
	// TODO: avoid the allocation here.
	return quoteWith(string(rune), '\'', false)
}

// QuoteRuneToASCII returns a single-quoted Go character literal representing
// the rune.  The returned string uses Go escape sequences (\t, \n, \xFF,
// \u0100) for non-ASCII characters and non-printable characters as defined
// by unicode.IsPrint.
func QuoteRuneToASCII(rune int) string {
	// TODO: avoid the allocation here.
	return quoteWith(string(rune), '\'', true)
}

// CanBackquote returns whether the string s would be
// a valid Go string literal if enclosed in backquotes.
func CanBackquote(s string) bool {
	for i := 0; i < len(s); i++ {
		if (s[i] < ' ' && s[i] != '\t') || s[i] == '`' {
			return false
		}
	}
	return true
}

func unhex(b byte) (v rune, ok bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return
}

// UnquoteChar decodes the first character or byte in the escaped string
// or character literal represented by the string s.
// It returns four values:
//
//	1) value, the decoded Unicode code point or byte value;
//	2) multibyte, a boolean indicating whether the decoded character requires a multibyte UTF-8 representation;
//	3) tail, the remainder of the string after the character; and
//	4) an error that will be nil if the character is syntactically valid.
//
// The second argument, quote, specifies the type of literal being parsed
// and therefore which escaped quote character is permitted.
// If set to a single quote, it permits the sequence \' and disallows unescaped '.
// If set to a double quote, it permits \" and disallows unescaped ".
// If set to zero, it does not permit either escape and allows both quote characters to appear unescaped.
func UnquoteChar(s string, quote byte) (value rune, multibyte bool, tail string, err error) {
	// easy cases
	switch c := s[0]; {
	case c == quote && (quote == '\'' || quote == '"'):
		err = ErrSyntax
		return
	case c >= utf8.RuneSelf:
		r, size := utf8.DecodeRuneInString(s)
		return r, true, s[size:], nil
	case c != '\\':
		return rune(s[0]), false, s[1:], nil
	}

	// hard case: c is backslash
	if len(s) <= 1 {
		err = ErrSyntax
		return
	}
	c := s[1]
	s = s[2:]

	switch c {
	case 'a':
		value = '\a'
	case 'b':
		value = '\b'
	case 'f':
		value = '\f'
	case 'n':
		value = '\n'
	case 'r':
		value = '\r'
	case 't':
		value = '\t'
	case 'v':
		value = '\v'
	case 'x', 'u', 'U':
		n := 0
		switch c {
		case 'x':
			n = 2
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		var v rune
		if len(s) < n {
			err = ErrSyntax
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = ErrSyntax
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		if c == 'x' {
			// single-byte string, possibly not UTF-8
			value = v
			break
		}
		if v > unicode.MaxRune {
			err = ErrSyntax
			return
		}
		value = v
		multibyte = true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := rune(c) - '0'
		if len(s) < 2 {
			err = ErrSyntax
			return
		}
		for j := 0; j < 2; j++ { // one digit already; two more
			x := rune(s[j]) - '0'
			if x < 0 || x > 7 {
				return
			}
			v = (v << 3) | x
		}
		s = s[2:]
		if v > 255 {
			err = ErrSyntax
			return
		}
		value = v
	case '\\':
		value = '\\'
	case '\'', '"':
		if c != quote {
			err = ErrSyntax
			return
		}
		value = rune(c)
	default:
		err = ErrSyntax
		return
	}
	tail = s
	return
}

// Unquote interprets s as a single-quoted, double-quoted,
// or backquoted Go string literal, returning the string value
// that s quotes.  (If s is single-quoted, it would be a Go
// character literal; Unquote returns the corresponding
// one-character string.)
func Unquote(s string) (t string, err error) {
	n := len(s)
	if n < 2 {
		return "", ErrSyntax
	}
	quote := s[0]
	if quote != s[n-1] {
		return "", ErrSyntax
	}
	s = s[1 : n-1]

	if quote == '`' {
		if strings.Contains(s, "`") {
			return "", ErrSyntax
		}
		return s, nil
	}
	if quote != '"' && quote != '\'' {
		return "", ErrSyntax
	}
	if strings.Index(s, "\n") >= 0 {
		return "", ErrSyntax
	}

	// Is it trivial?  Avoid allocation.
	if strings.Index(s, `\`) < 0 && strings.IndexRune(s, rune(quote)) < 0 {
		switch quote {
		case '"':
			return s, nil
		case '\'':
			r, size := utf8.DecodeRuneInString(s)
			if size == len(s) && (r != utf8.RuneError || size != 1) {
				return s, nil
			}
		}
	}

	var buf bytes.Buffer
	for len(s) > 0 {
		c, multibyte, ss, err := UnquoteChar(s, quote)
		if err != nil {
			return "", err
		}
		s = ss
		if c < utf8.RuneSelf || !multibyte {
			buf.WriteByte(byte(c))
		} else {
			buf.WriteString(string(c))
		}
		if quote == '\'' && len(s) != 0 {
			// single-quoted must be single character
			return "", ErrSyntax
		}
	}
	return buf.String(), nil
}
