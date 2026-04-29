package format

import (
	"bytes"
	"fmt"

	"github.com/janezpodhostnik/cadencefmt/internal/format/render"
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/onflow/cadence/parser"
	"github.com/turbolent/prettier"
)

// Format parses Cadence source and returns deterministically formatted output.
// filename is used for diagnostics only; the file need not exist on disk.
func Format(src []byte, filename string, opts Options) ([]byte, error) {
	program, err := parser.ParseProgram(nil, src, parser.Config{})
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Extract and attach comments
	comments := trivia.Scan(src)
	groups := trivia.Group(comments, src)
	cm := trivia.Attach(program, groups, src)

	indent := opts.Indent
	if opts.UseTabs {
		indent = "\t"
	}

	// Render AST with interleaved comments
	doc := render.Program(program, cm, opts.LineWidth, indent)

	var buf bytes.Buffer
	prettier.Prettier(&buf, doc, opts.LineWidth, indent)

	result := buf.Bytes()

	// Verify no orphaned comments remain
	if !cm.IsEmpty() {
		return result, fmt.Errorf("internal error: orphaned comments remain in CommentMap")
	}

	return result, nil
}
