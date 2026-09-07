// Package luatable is a minimal parser for Lua table *literals*, the kind
// WoW writes into SavedVariables files: nested tables of strings, numbers,
// booleans and nil, addressed by either a bracketed key ([1], ["Name"]) or a
// bare identifier (Name = ...). It deliberately does not evaluate arbitrary
// Lua — SavedVariables files never contain function calls or expressions,
// only literal data, so a small recursive-descent parser is enough and
// avoids embedding a real Lua VM.
package luatable

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Value is one of: nil, bool, float64, string, or map[any]Value (a Lua table,
// keyed by either a string or a float64 — Lua doesn't distinguish array vs
// hash tables, so callers that expect an array should look up integer keys
// 1..n themselves).
type Value interface{}

// Table is the concrete type returned for `{ ... }` literals.
type Table map[interface{}]Value

// ParseAssignment parses `Name = { ... }` (optionally followed by more such
// assignments) and returns the value bound to varName, as SavedVariables
// files are a sequence of top-level global assignments.
func ParseAssignment(src string, varName string) (Value, error) {
	p := &parser{src: src}
	for {
		p.skipSpaceAndComments()
		if p.pos >= len(p.src) {
			break
		}
		name, ok := p.readIdentifier()
		if !ok {
			return nil, fmt.Errorf("expected identifier at byte %d", p.pos)
		}
		p.skipSpaceAndComments()
		if !p.consume('=') {
			return nil, fmt.Errorf("expected '=' after %q at byte %d", name, p.pos)
		}
		p.skipSpaceAndComments()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if name == varName {
			return val, nil
		}
		p.skipSpaceAndComments()
		p.consume(';') // optional statement separator some serializers emit
	}
	return nil, fmt.Errorf("variable %q not found", varName)
}

type parser struct {
	src string
	pos int
}

func (p *parser) skipSpaceAndComments() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			p.pos++
			continue
		}
		if strings.HasPrefix(p.src[p.pos:], "--") {
			nl := strings.IndexByte(p.src[p.pos:], '\n')
			if nl < 0 {
				p.pos = len(p.src)
			} else {
				p.pos += nl + 1
			}
			continue
		}
		break
	}
}

func (p *parser) consume(b byte) bool {
	if p.pos < len(p.src) && p.src[p.pos] == b {
		p.pos++
		return true
	}
	return false
}

func (p *parser) readIdentifier() (string, bool) {
	start := p.pos
	for p.pos < len(p.src) {
		c := rune(p.src[p.pos])
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' {
			p.pos++
			continue
		}
		break
	}
	if p.pos == start {
		return "", false
	}
	return p.src[start:p.pos], true
}

func (p *parser) parseValue() (Value, error) {
	p.skipSpaceAndComments()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of input")
	}
	switch c := p.src[p.pos]; {
	case c == '{':
		return p.parseTable()
	case c == '"' || c == '\'':
		return p.parseString()
	case c == '-' || (c >= '0' && c <= '9'):
		return p.parseNumber()
	default:
		word, ok := p.readIdentifier()
		if !ok {
			return nil, fmt.Errorf("unexpected character %q at byte %d", p.src[p.pos], p.pos)
		}
		switch word {
		case "nil":
			return nil, nil
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return nil, fmt.Errorf("unexpected identifier %q at byte %d", word, p.pos)
		}
	}
}

func (p *parser) parseString() (string, error) {
	quote := p.src[p.pos]
	p.pos++
	var sb strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == quote {
			p.pos++
			return sb.String(), nil
		}
		if c == '\\' && p.pos+1 < len(p.src) {
			next := p.src[p.pos+1]
			switch next {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '"', '\'', '\\':
				sb.WriteByte(next)
			default:
				sb.WriteByte(next)
			}
			p.pos += 2
			continue
		}
		sb.WriteByte(c)
		p.pos++
	}
	return "", fmt.Errorf("unterminated string starting near byte %d", p.pos)
}

func (p *parser) parseNumber() (float64, error) {
	start := p.pos
	if p.src[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-' {
			p.pos++
			continue
		}
		break
	}
	return strconv.ParseFloat(p.src[start:p.pos], 64)
}

func (p *parser) parseTable() (Table, error) {
	if !p.consume('{') {
		return nil, fmt.Errorf("expected '{' at byte %d", p.pos)
	}
	table := Table{}
	arrayIndex := 1.0
	for {
		p.skipSpaceAndComments()
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unterminated table starting before byte %d", p.pos)
		}
		if p.consume('}') {
			return table, nil
		}

		var key interface{}
		if p.src[p.pos] == '[' {
			p.pos++
			p.skipSpaceAndComments()
			keyVal, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			p.skipSpaceAndComments()
			if !p.consume(']') {
				return nil, fmt.Errorf("expected ']' at byte %d", p.pos)
			}
			key = keyVal
			p.skipSpaceAndComments()
			if !p.consume('=') {
				return nil, fmt.Errorf("expected '=' after key at byte %d", p.pos)
			}
		} else if isIdentStart(p.src[p.pos]) {
			save := p.pos
			name, _ := p.readIdentifier()
			p.skipSpaceAndComments()
			if p.consume('=') {
				key = name
			} else {
				// Not actually `name = value`; it was a bare value (e.g. a bare
				// identifier value, which SavedVariables never emits, but be safe).
				p.pos = save
				key = arrayIndex
				arrayIndex++
			}
		} else {
			key = arrayIndex
			arrayIndex++
		}

		p.skipSpaceAndComments()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		table[key] = val

		p.skipSpaceAndComments()
		if p.consume(',') || p.consume(';') {
			continue
		}
		p.skipSpaceAndComments()
		if p.consume('}') {
			return table, nil
		}
		return nil, fmt.Errorf("expected ',' or '}' at byte %d", p.pos)
	}
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// GetTable returns v[key] as a Table, if present and of the right type.
func (t Table) GetTable(key interface{}) (Table, bool) {
	v, ok := t[key]
	if !ok || v == nil {
		return nil, false
	}
	sub, ok := v.(Table)
	return sub, ok
}

// GetString returns v[key] as a string, if present and of the right type.
func (t Table) GetString(key interface{}) (string, bool) {
	v, ok := t[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetNumber returns v[key] as a float64, if present and of the right type.
func (t Table) GetNumber(key interface{}) (float64, bool) {
	v, ok := t[key]
	if !ok || v == nil {
		return 0, false
	}
	n, ok := v.(float64)
	return n, ok
}
