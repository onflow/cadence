package trivia

import (
	"testing"
)

func TestScan(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []struct {
			kind Kind
			text string
			line int
			col  int
		}
	}{
		{
			name:   "basic line comment",
			source: "// hello",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// hello", 1, 0},
			},
		},
		{
			name:   "basic block comment",
			source: "/* hello */",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindBlock, "/* hello */", 1, 0},
			},
		},
		{
			name:   "doc-line comment",
			source: "/// doc comment",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindDocLine, "/// doc comment", 1, 0},
			},
		},
		{
			name:   "doc-block comment",
			source: "/** doc block */",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindDocBlock, "/** doc block */", 1, 0},
			},
		},
		{
			name:   "four slashes is regular line",
			source: "//// not doc",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "//// not doc", 1, 0},
			},
		},
		{
			name:   "empty block comment is regular",
			source: "/**/",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindBlock, "/**/", 1, 0},
			},
		},
		{
			name:   "triple star block is regular",
			source: "/*** stars ***/",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindBlock, "/*** stars ***/", 1, 0},
			},
		},
		{
			name:   "nested block comments",
			source: "/* outer /* inner */ outer */",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindBlock, "/* outer /* inner */ outer */", 1, 0},
			},
		},
		{
			name:   "comment-like inside string",
			source: `"// not a comment"`,
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{},
		},
		{
			name:   "comment-like inside string template",
			source: `"\(a /* not */ + b)"`,
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{},
		},
		{
			name:   "multiple comments",
			source: "let x = 1 // first\nlet y = 2 // second",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// first", 1, 10},
				{KindLine, "// second", 2, 10},
			},
		},
		{
			name:   "mixed comment kinds",
			source: "// line\n/* block */\n/// doc",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// line", 1, 0},
				{KindBlock, "/* block */", 2, 0},
				{KindDocLine, "/// doc", 3, 0},
			},
		},
		{
			name:   "empty input",
			source: "",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{},
		},
		{
			name:   "comment at EOF without newline",
			source: "let x = 1 // trailing",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// trailing", 1, 10},
			},
		},
		{
			name:   "multiline block comment",
			source: "/* line 1\n   line 2 */",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindBlock, "/* line 1\n   line 2 */", 1, 0},
			},
		},
		{
			name:   "string with escaped quote",
			source: `"escaped \" quote" // real comment`,
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// real comment", 1, 19},
			},
		},
		{
			name:   "string with backslash at end",
			source: `"test\\" // comment`,
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// comment", 1, 9},
			},
		},
		{
			name:   "nested string in template",
			source: `"\("inner")" // outer`,
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				// "\("inner")" is a string with template containing "inner"
				// then // outer is a real comment
				{KindLine, "// outer", 1, 13},
			},
		},
		{
			name:   "comment with only slashes",
			source: "//",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "//", 1, 0},
			},
		},
		{
			name:   "doc line at end of file",
			source: "///",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindDocLine, "///", 1, 0},
			},
		},
		{
			name:   "comment after code",
			source: "access(all) fun main() {} // end",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// end", 1, 26},
			},
		},
		{
			name:   "deeply nested block comments",
			source: "/* a /* b /* c */ b */ a */",
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindBlock, "/* a /* b /* c */ b */ a */", 1, 0},
			},
		},
		{
			name:   "template with nested parens",
			source: `"\(f(g(x)))" // after`,
			expected: []struct {
				kind Kind
				text string
				line int
				col  int
			}{
				{KindLine, "// after", 1, 13},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Scan([]byte(tt.source))

			if len(got) != len(tt.expected) {
				t.Fatalf("got %d comments, want %d.\ngot: %v", len(got), len(tt.expected), got)
			}

			for i, exp := range tt.expected {
				c := got[i]
				if c.Kind != exp.kind {
					t.Errorf("comment[%d].Kind = %s, want %s", i, c.Kind, exp.kind)
				}
				if c.Text != exp.text {
					t.Errorf("comment[%d].Text = %q, want %q", i, c.Text, exp.text)
				}
				if c.Start.Line != exp.line {
					t.Errorf("comment[%d].Start.Line = %d, want %d", i, c.Start.Line, exp.line)
				}
				if c.Start.Column != exp.col {
					t.Errorf("comment[%d].Start.Column = %d, want %d", i, c.Start.Column, exp.col)
				}
			}
		})
	}
}

func TestGroup(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []int // number of comments in each group
	}{
		{
			name:     "single comment",
			source:   "// one",
			expected: []int{1},
		},
		{
			name:     "two adjacent line comments",
			source:   "// first\n// second",
			expected: []int{2},
		},
		{
			name:     "two comments with blank line",
			source:   "// first\n\n// second",
			expected: []int{1, 1},
		},
		{
			name:     "three comments: two grouped then one",
			source:   "// a\n// b\n\n// c",
			expected: []int{2, 1},
		},
		{
			name:     "block then line on next line",
			source:   "/* block */\n// line",
			expected: []int{2},
		},
		{
			name:     "block then line with blank line",
			source:   "/* block */\n\n// line",
			expected: []int{1, 1},
		},
		{
			name:     "empty input",
			source:   "",
			expected: nil,
		},
		{
			name:     "multiline block then line adjacent",
			source:   "/* line1\nline2 */\n// after",
			expected: []int{2},
		},
		{
			name:     "multiline block then line with blank",
			source:   "/* line1\nline2 */\n\n// after",
			expected: []int{1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments := Scan([]byte(tt.source))
			groups := Group(comments)

			if tt.expected == nil {
				if groups != nil {
					t.Fatalf("expected nil groups, got %d", len(groups))
				}
				return
			}

			if len(groups) != len(tt.expected) {
				t.Fatalf("got %d groups, want %d", len(groups), len(tt.expected))
			}

			for i, expCount := range tt.expected {
				if len(groups[i].Comments) != expCount {
					t.Errorf("group[%d] has %d comments, want %d",
						i, len(groups[i].Comments), expCount)
				}
			}
		})
	}
}
