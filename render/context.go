package render

import "github.com/onflow/cadence/ast"

// Context holds state shared across render functions.
type Context struct {
	Semicolons map[ast.Element]bool
	Source     []byte // original source bytes, used for blank line detection
}

// HasSemicolon reports whether elem had a trailing semicolon in the source.
func (c *Context) HasSemicolon(elem ast.Element) bool {
	return c != nil && c.Semicolons[elem]
}
