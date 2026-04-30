package format

import (
	"bytes"
	"fmt"

	"github.com/janezpodhostnik/cadencefmt/internal/format/render"
	"github.com/janezpodhostnik/cadencefmt/internal/format/rewrite"
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/janezpodhostnik/cadencefmt/internal/format/verify"
	"github.com/onflow/cadence/parser"
	"github.com/turbolent/prettier"
)

// Format parses Cadence source and returns deterministically formatted output.
// filename is used for diagnostics only; the file need not exist on disk.
func Format(src []byte, filename string, opts Options) ([]byte, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	program, err := parser.ParseProgram(nil, src, parser.Config{})
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Extract and attach comments
	comments := trivia.Scan(src)
	groups := trivia.Group(comments, src)
	cm := trivia.Attach(program, groups, src)

	// Apply AST rewrites (import sorting, etc.)
	if err := rewrite.Apply(program, cm); err != nil {
		return nil, fmt.Errorf("rewrite error: %w", err)
	}

	indent := opts.Indent
	if opts.UseTabs {
		indent = "\t"
	}

	// Render AST with interleaved comments
	ctx := &render.Context{}
	if !opts.StripSemicolons {
		ctx.Semicolons = trivia.ScanSemicolons(src, program)
	}
	doc := render.Program(program, cm, ctx)

	var buf bytes.Buffer
	prettier.Prettier(&buf, doc, opts.LineWidth, indent)

	result := collapseBlankLines(
		rejoinStringInterpolations(stripTrailingLineWhitespace(buf.Bytes())),
		opts.KeepBlankLines,
	)

	// Verify no orphaned comments remain
	if !cm.IsEmpty() {
		details := cm.OrphanDetails()
		return result, fmt.Errorf("internal error: orphaned comments remain in CommentMap\n%s", details)
	}

	// Round-trip verification: re-parse and compare ASTs
	if !opts.SkipVerify {
		if err := verify.RoundTrip(src, result); err != nil {
			return result, fmt.Errorf("internal error: round-trip verification failed: %w", err)
		}
	}

	return result, nil
}

// rejoinStringInterpolations collapses line breaks inside string template
// interpolations \(...). The prettier library may break expressions inside
// interpolations across lines; this rejoins them into a single line.
// Tracks paren depth to find the matching ) for each \(.
func rejoinStringInterpolations(data []byte) []byte {
	result := make([]byte, 0, len(data))
	i := 0
	inString := false

	for i < len(data) {
		b := data[i]

		// Track string boundaries (handle escaped quotes)
		if b == '"' && !inString {
			inString = true
			result = append(result, b)
			i++
			continue
		}
		if b == '"' && inString {
			inString = false
			result = append(result, b)
			i++
			continue
		}

		// Handle escape sequences inside strings
		if inString && b == '\\' && i+1 < len(data) {
			if data[i+1] == '(' {
				// Start of interpolation \( — scan to matching )
				result = append(result, '\\', '(')
				i += 2
				depth := 1
				for i < len(data) && depth > 0 {
					c := data[i]
					if c == '(' {
						depth++
						result = append(result, c)
					} else if c == ')' {
						depth--
						result = append(result, c)
					} else if c == '\n' {
						// Collapse newline + following whitespace. The expression
						// content already has operators/dots that provide spacing.
						i++
						for i < len(data) && (data[i] == ' ' || data[i] == '\t') {
							i++
						}
						// Add a space unless the next char is . (member access)
						if i < len(data) && data[i] != '.' {
							result = append(result, ' ')
						}
						continue
					} else if c == '"' {
						// Nested string inside interpolation — copy until closing "
						result = append(result, c)
						i++
						for i < len(data) && data[i] != '"' {
							if data[i] == '\\' && i+1 < len(data) {
								result = append(result, data[i], data[i+1])
								i += 2
								continue
							}
							result = append(result, data[i])
							i++
						}
						if i < len(data) {
							result = append(result, data[i]) // closing "
						}
					} else {
						result = append(result, c)
					}
					i++
				}
				continue
			}
			// Other escape: copy both bytes
			result = append(result, b, data[i+1])
			i += 2
			continue
		}

		result = append(result, b)
		i++
	}

	return result
}

// collapseBlankLines limits consecutive blank lines to at most max.
func collapseBlankLines(data []byte, max int) []byte {
	lines := bytes.Split(data, []byte("\n"))
	result := make([][]byte, 0, len(lines))
	consecutive := 0
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			consecutive++
			if consecutive > max {
				continue
			}
		} else {
			consecutive = 0
		}
		result = append(result, line)
	}
	return bytes.Join(result, []byte("\n"))
}

// stripTrailingLineWhitespace strips indent whitespace from blank lines.
// The prettier library emits indent prefixes on blank lines inside Indent
// blocks (e.g. "    \n" instead of "\n"); this cleans that up.
// Only whitespace-only lines are affected — content lines are not touched.
func stripTrailingLineWhitespace(data []byte) []byte {
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		if len(bytes.TrimRight(line, " \t")) == 0 {
			lines[i] = nil
		}
	}
	return bytes.Join(lines, []byte("\n"))
}
