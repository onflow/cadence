package trivia

import (
	"testing"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/parser"
)

func parse(t *testing.T, source string) *ast.Program {
	t.Helper()
	program, err := parser.ParseProgram(nil, []byte(source), parser.Config{})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return program
}

func attachComments(t *testing.T, source string) (*ast.Program, *CommentMap) {
	t.Helper()
	program := parse(t, source)
	src := []byte(source)
	comments := Scan(src)
	groups := Group(comments, src)
	cm := Attach(program, groups, src)
	return program, cm
}

func TestAttach_FileHeader(t *testing.T) {
	source := `// copyright 2024
// all rights reserved

access(all) fun main() {}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	if len(cm.Header) != 1 {
		t.Fatalf("expected 1 header group, got %d", len(cm.Header))
	}
	if cm.Header[0].Comments[0].Text != "// copyright 2024" {
		t.Errorf("header text = %q", cm.Header[0].Comments[0].Text)
	}

	// No leading on the function (header is separated by blank line)
	leading := cm.Leading[decls[0]]
	if len(leading) != 0 {
		t.Errorf("expected no leading on decl, got %d", len(leading))
	}
}

func TestAttach_FileFooter(t *testing.T) {
	// No blank line before comment → trailing of last decl, not footer
	source := `access(all) fun main() {}
// trailing comment
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	if len(cm.Footer) != 0 {
		t.Errorf("expected no footer, got %d", len(cm.Footer))
	}
	trailing := cm.Trailing[decls[0]]
	if len(trailing) != 1 {
		t.Fatalf("expected 1 trailing group, got %d", len(trailing))
	}
	if trailing[0].Comments[0].Text != "// trailing comment" {
		t.Errorf("trailing text = %q", trailing[0].Comments[0].Text)
	}
}

func TestAttach_FooterWithBlankLine(t *testing.T) {
	// Blank line before comment → true footer
	source := `access(all) fun main() {}

// footer comment
`
	_, cm := attachComments(t, source)

	if len(cm.Footer) != 1 {
		t.Fatalf("expected 1 footer group, got %d", len(cm.Footer))
	}
	if cm.Footer[0].Comments[0].Text != "// footer comment" {
		t.Errorf("footer text = %q", cm.Footer[0].Comments[0].Text)
	}
}

func TestAttach_LeadingOnFunction(t *testing.T) {
	source := `// this function does something
access(all) fun main() {}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	if len(cm.Header) != 0 {
		t.Errorf("expected no header, got %d", len(cm.Header))
	}

	leading := cm.Leading[decls[0]]
	if len(leading) != 1 {
		t.Fatalf("expected 1 leading group, got %d", len(leading))
	}
	if leading[0].Comments[0].Text != "// this function does something" {
		t.Errorf("leading text = %q", leading[0].Comments[0].Text)
	}
}

func TestAttach_TrailingAfterFunction(t *testing.T) {
	source := `access(all) fun a() {}
// after a
access(all) fun b() {}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	// "// after a" is right below a() with no blank line → trailing of a
	trailing := cm.Trailing[decls[0]]
	if len(trailing) != 1 {
		t.Fatalf("expected 1 trailing group on a, got %d", len(trailing))
	}
	if trailing[0].Comments[0].Text != "// after a" {
		t.Errorf("trailing text = %q", trailing[0].Comments[0].Text)
	}
}

func TestAttach_SameLine(t *testing.T) {
	source := `let x = 1 // inline
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	sl := cm.SameLine[decls[0]]
	if sl == nil {
		t.Fatal("expected same-line comment on declaration")
	}
	if sl.Comments[0].Text != "// inline" {
		t.Errorf("same-line text = %q", sl.Comments[0].Text)
	}
}

func TestAttach_BetweenDeclarations(t *testing.T) {
	source := `access(all) fun a() {}

// belongs to b (blank line above)
access(all) fun b() {}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	// Blank line between a and comment → leading of b
	leading := cm.Leading[decls[1]]
	if len(leading) != 1 {
		t.Fatalf("expected 1 leading group on b, got %d", len(leading))
	}
	if leading[0].Comments[0].Text != "// belongs to b (blank line above)" {
		t.Errorf("leading text = %q", leading[0].Comments[0].Text)
	}
}

func TestAttach_DocComment(t *testing.T) {
	source := `/// This is a doc comment
/// for the function below
access(all) fun documented() {}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	leading := cm.Leading[decls[0]]
	if len(leading) != 1 {
		t.Fatalf("expected 1 leading group, got %d", len(leading))
	}
	if len(leading[0].Comments) != 2 {
		t.Fatalf("expected 2 comments in group, got %d", len(leading[0].Comments))
	}
	if leading[0].Comments[0].Kind != KindDocLine {
		t.Errorf("expected DocLine, got %s", leading[0].Comments[0].Kind)
	}
}

func TestAttach_HeaderAndLeading(t *testing.T) {
	source := `// file header

// leading on fun
access(all) fun main() {}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	if len(cm.Header) != 1 {
		t.Fatalf("expected 1 header group, got %d", len(cm.Header))
	}

	leading := cm.Leading[decls[0]]
	if len(leading) != 1 {
		t.Fatalf("expected 1 leading group, got %d", len(leading))
	}
}

func TestAttach_InsideFunctionBody(t *testing.T) {
	source := `access(all) fun main() {
    let x = 1
    // between statements
    let y = 2
}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	// The comment should be attached somewhere inside the function
	// Verify it's not at the top level
	if len(cm.Header) != 0 {
		t.Errorf("expected no header, got %d", len(cm.Header))
	}
	if len(cm.Footer) != 0 {
		t.Errorf("expected no footer, got %d", len(cm.Footer))
	}
	if len(cm.Leading[decls[0]]) != 0 {
		t.Errorf("expected no leading on function, got %d", len(cm.Leading[decls[0]]))
	}

	// The comment should be attached to some inner node
	totalComments := 0
	for _, groups := range cm.Leading {
		for _, g := range groups {
			totalComments += len(g.Comments)
		}
	}
	for _, groups := range cm.Trailing {
		for _, g := range groups {
			totalComments += len(g.Comments)
		}
	}
	if totalComments == 0 {
		t.Error("comment inside function body was not attached to any node")
	}
}

func TestAttach_EmptyMap(t *testing.T) {
	source := `access(all) fun main() {}
`
	_, cm := attachComments(t, source)

	if !cm.IsEmpty() {
		t.Error("expected empty comment map for source without comments")
	}
}

func TestAttach_Take(t *testing.T) {
	source := `// leading
access(all) fun main() {} // same-line
// trailing
access(all) fun other() {}
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	leading, sameLine, trailing := cm.Take(decls[0])

	if len(leading) != 1 {
		t.Errorf("Take leading: got %d, want 1", len(leading))
	}
	if sameLine == nil {
		t.Error("Take sameLine: got nil, want comment")
	}
	if len(trailing) != 1 {
		t.Errorf("Take trailing: got %d, want 1", len(trailing))
	}

	// After Take, the node should have no comments
	leading2, sameLine2, trailing2 := cm.Take(decls[0])
	if len(leading2) != 0 || sameLine2 != nil || len(trailing2) != 0 {
		t.Error("Take should return nothing on second call")
	}
}

func TestAttach_MultipleDecls(t *testing.T) {
	source := `// header

// doc for a
access(all) fun a() {} // inline a

// doc for b
access(all) fun b() {}

// footer
`
	program, cm := attachComments(t, source)
	decls := program.Declarations()

	if len(cm.Header) != 1 {
		t.Errorf("header: got %d, want 1", len(cm.Header))
	}

	if len(cm.Leading[decls[0]]) != 1 {
		t.Errorf("leading on a: got %d, want 1", len(cm.Leading[decls[0]]))
	}

	if cm.SameLine[decls[0]] == nil {
		t.Error("expected same-line on a")
	}

	if len(cm.Leading[decls[1]]) != 1 {
		t.Errorf("leading on b: got %d, want 1", len(cm.Leading[decls[1]]))
	}

	if len(cm.Footer) != 1 {
		t.Errorf("footer: got %d, want 1", len(cm.Footer))
	}
}
