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
	doc := render.Program(program, cm, opts.LineWidth, indent)

	var buf bytes.Buffer
	prettier.Prettier(&buf, doc, opts.LineWidth, indent)

	result := buf.Bytes()

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
