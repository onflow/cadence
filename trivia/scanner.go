package trivia

import "github.com/onflow/cadence/ast"

// Scan extracts all comments from Cadence source bytes.
// It correctly skips comment-like sequences inside string literals
// and string template interpolations.
func Scan(source []byte) []Comment {
	s := &scanner{source: source, line: 1}
	return s.scan()
}

type scanner struct {
	source []byte
	pos    int
	line   int
	col    int
}

func (s *scanner) position() ast.Position {
	return ast.Position{
		Offset: s.pos,
		Line:   s.line,
		Column: s.col,
	}
}

func (s *scanner) advance() {
	if s.pos >= len(s.source) {
		return
	}
	if s.source[s.pos] == '\n' {
		s.pos++
		s.line++
		s.col = 0
	} else {
		s.pos++
		s.col++
	}
}

func (s *scanner) peek() byte {
	if s.pos+1 < len(s.source) {
		return s.source[s.pos+1]
	}
	return 0
}

func (s *scanner) scan() []Comment {
	var comments []Comment
	for s.pos < len(s.source) {
		ch := s.source[s.pos]
		switch {
		case ch == '"':
			s.skipString()
		case ch == '/' && s.peek() == '/':
			comments = append(comments, s.scanLineComment())
		case ch == '/' && s.peek() == '*':
			comments = append(comments, s.scanBlockComment())
		default:
			s.advance()
		}
	}
	return comments
}

// scanLineComment consumes a line comment starting at the current position
// (which must be at '/'). Returns the comment with Kind set to either
// KindLine or KindDocLine.
func (s *scanner) scanLineComment() Comment {
	start := s.position()
	startOff := s.pos

	// Determine kind:
	// /// is doc-line only if 4th char is NOT /
	// //// is a regular line comment
	kind := KindLine
	if s.pos+2 < len(s.source) && s.source[s.pos+2] == '/' {
		if s.pos+3 >= len(s.source) || s.source[s.pos+3] != '/' {
			kind = KindDocLine
		}
	}

	// Consume until newline or EOF (newline is NOT part of the comment)
	for s.pos < len(s.source) && s.source[s.pos] != '\n' {
		s.advance()
	}

	text := string(s.source[startOff:s.pos])

	// End position: last character of the comment (same line as start)
	endOff := s.pos - 1
	if endOff < startOff {
		endOff = startOff
	}
	end := ast.Position{
		Offset: endOff,
		Line:   start.Line,
		Column: start.Column + (endOff - startOff),
	}

	return Comment{Kind: kind, Start: start, End: end, Text: text}
}

// scanBlockComment consumes a block comment starting at the current position
// (which must be at '/'). Handles nested block comments. Returns the comment
// with Kind set to either KindBlock or KindDocBlock.
func (s *scanner) scanBlockComment() Comment {
	start := s.position()
	startOff := s.pos

	// Determine kind:
	// /** is doc-block if char after /** is NOT * and NOT /
	// /**/ is regular empty block
	// /*** is regular block
	kind := KindBlock
	if s.pos+2 < len(s.source) && s.source[s.pos+2] == '*' {
		if s.pos+3 < len(s.source) && s.source[s.pos+3] != '*' && s.source[s.pos+3] != '/' {
			kind = KindDocBlock
		}
	}

	s.advance() // skip /
	s.advance() // skip *
	depth := 1

	var end ast.Position
	for s.pos < len(s.source) && depth > 0 {
		if s.source[s.pos] == '/' && s.peek() == '*' {
			depth++
			s.advance()
			s.advance()
		} else if s.source[s.pos] == '*' && s.peek() == '/' {
			depth--
			if depth == 0 {
				s.advance() // skip *
				end = s.position()
				s.advance() // skip /
				break
			}
			s.advance()
			s.advance()
		} else {
			s.advance()
		}
	}

	// Unterminated block comment
	if depth > 0 {
		end = s.position()
		if end.Offset > startOff {
			end.Offset--
			if end.Column > 0 {
				end.Column--
			}
		}
	}

	text := string(s.source[startOff:s.pos])
	return Comment{Kind: kind, Start: start, End: end, Text: text}
}

// skipString consumes a string literal starting at the current position
// (which must be at '"'). Handles escape sequences and string template
// interpolations \(expr).
func (s *scanner) skipString() {
	s.advance() // skip opening "
	for s.pos < len(s.source) {
		ch := s.source[s.pos]
		switch ch {
		case '"':
			s.advance()
			return
		case '\\':
			s.advance() // skip backslash
			if s.pos < len(s.source) {
				if s.source[s.pos] == '(' {
					s.advance() // skip (
					s.skipStringTemplate()
				} else {
					s.advance() // skip escaped character
				}
			}
		case '\n':
			// Invalid string termination; stop to avoid getting stuck
			return
		default:
			s.advance()
		}
	}
}

// skipStringTemplate consumes a string template interpolation expression.
// Called after \( has been consumed. Tracks nested parentheses and handles
// nested string literals within the expression. Comments inside templates
// are skipped (not extracted) since template expressions are preserved verbatim.
func (s *scanner) skipStringTemplate() {
	depth := 1
	for s.pos < len(s.source) && depth > 0 {
		ch := s.source[s.pos]
		switch ch {
		case '(':
			depth++
			s.advance()
		case ')':
			depth--
			s.advance()
		case '"':
			s.skipString()
		case '/':
			if s.peek() == '/' {
				// Line comment inside template — skip to end of line
				for s.pos < len(s.source) && s.source[s.pos] != '\n' {
					s.advance()
				}
			} else if s.peek() == '*' {
				s.skipNestedBlockComment()
			} else {
				s.advance()
			}
		default:
			s.advance()
		}
	}
}

// skipNestedBlockComment consumes a block comment without recording it.
// Used inside string templates where comments are preserved verbatim.
func (s *scanner) skipNestedBlockComment() {
	s.advance() // /
	s.advance() // *
	depth := 1
	for s.pos < len(s.source) && depth > 0 {
		if s.source[s.pos] == '/' && s.peek() == '*' {
			depth++
			s.advance()
			s.advance()
		} else if s.source[s.pos] == '*' && s.peek() == '/' {
			depth--
			s.advance()
			s.advance()
		} else {
			s.advance()
		}
	}
}
