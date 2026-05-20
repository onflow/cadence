/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package trivia

import (
	"fmt"
	"sort"

	"github.com/onflow/cadence/ast"
)

// CommentMap binds comment groups to AST nodes by position class.
// TakeRaw() removes and returns comments for a node — this ensures each
// comment is emitted exactly once during rendering. After rendering,
// the map should be empty; any leftovers indicate a bug.
//
// CommentMap also implements ast.PrettyContext, so it can be passed directly
// to ast.Doc(ctx) and supply comment placement, blank-line preservation,
// and header/footer rendering. The Take/HasComments/BlankLineBetween/Header/
// Footer/Wrap methods (see render.go) are the PrettyContext implementation.
type CommentMap struct {
	HeaderComments []*CommentGroup // before first declaration
	FooterComments []*CommentGroup // after last declaration
	Leading        map[ast.Element][]*CommentGroup
	Trailing       map[ast.Element][]*CommentGroup
	SameLine       map[ast.Element]*CommentGroup // at most one per node
	Source         []byte                        // original source bytes, for blank-line detection
	Semicolons     map[ast.Element]bool          // elements that had a trailing semicolon in source
}

// NewCommentMap creates an empty CommentMap with initialized maps.
func NewCommentMap() *CommentMap {
	return &CommentMap{
		Leading:  make(map[ast.Element][]*CommentGroup),
		Trailing: make(map[ast.Element][]*CommentGroup),
		SameLine: make(map[ast.Element]*CommentGroup),
	}
}

// TakeRaw removes and returns all comments associated with n as raw comment
// groups. Use this when callers need to manipulate the groups directly;
// renderers should use Take (the PrettyContext method) which returns
// already-rendered prettier docs.
func (cm *CommentMap) TakeRaw(n ast.Element) (leading []*CommentGroup, sameLine *CommentGroup, trailing []*CommentGroup) {
	leading = cm.Leading[n]
	delete(cm.Leading, n)
	sameLine = cm.SameLine[n]
	delete(cm.SameLine, n)
	trailing = cm.Trailing[n]
	delete(cm.Trailing, n)
	return
}

// TakeHeader removes and returns header comments.
func (cm *CommentMap) TakeHeader() []*CommentGroup {
	h := cm.HeaderComments
	cm.HeaderComments = nil
	return h
}

// TakeFooter removes and returns footer comments.
func (cm *CommentMap) TakeFooter() []*CommentGroup {
	f := cm.FooterComments
	cm.FooterComments = nil
	return f
}

// IsEmpty returns true if no comments remain in the map.
func (cm *CommentMap) IsEmpty() bool {
	return len(cm.HeaderComments) == 0 &&
		len(cm.FooterComments) == 0 &&
		len(cm.Leading) == 0 &&
		len(cm.Trailing) == 0 &&
		len(cm.SameLine) == 0
}

// OrphanDetails returns a human-readable summary of remaining comments in the map.
func (cm *CommentMap) OrphanDetails() string {
	var details string
	for k, v := range cm.Leading {
		for _, g := range v {
			details += fmt.Sprintf("  Leading on %T at %s: %q\n", k, k.StartPosition(), g.Comments[0].Text)
		}
	}
	for k, v := range cm.Trailing {
		for _, g := range v {
			details += fmt.Sprintf("  Trailing on %T at %s: %q\n", k, k.StartPosition(), g.Comments[0].Text)
		}
	}
	for k, v := range cm.SameLine {
		details += fmt.Sprintf("  SameLine on %T at %s: %q\n", k, k.StartPosition(), v.Comments[0].Text)
	}
	return details
}

// Attach walks the AST and binds comment groups to nodes by position.
func Attach(program *ast.Program, groups []*CommentGroup, source []byte) *CommentMap {
	cm := NewCommentMap()
	cm.Source = source
	if len(groups) == 0 {
		return cm
	}

	decls := program.Declarations()
	elements := make([]ast.Element, len(decls))
	for i, d := range decls {
		elements[i] = d
	}

	remaining := attachLevel(cm, elements, groups, true, source)

	// Anything left over is footer
	cm.FooterComments = append(cm.FooterComments, remaining...)

	// Post-process: hoist comments between a variable declaration's type
	// annotation and its value to the leading position of the value. Without
	// this, a `// comment` after the type annotation would render on the same
	// line as the type, swallowing the `=` operator that follows.
	hoistVarDeclTypeComments(cm, program)

	// Post-process: hoist comments positionally inside `access(...)` parens
	// to be trailing of the last entitlement, so they render inline rather
	// than getting attached to the next AST element (e.g., TypeAnnotation).
	hoistAccessInlineComments(cm, program, source)
	return cm
}

// hoistAccessInlineComments walks each declaration with an access modifier,
// finds the closing `)` of `access(...)` in the source, and moves any
// comment whose start offset falls between the last entitlement's end and
// the `)` to be trailing of that last entitlement. Without this, comments
// like `access(A /* note */) let x` would attach to the next AST element
// (the TypeAnnotation, etc.) due to the deliberate exclusion of entitlement
// children in getChildren.
func hoistAccessInlineComments(cm *CommentMap, program *ast.Program, source []byte) {
	if len(source) == 0 {
		return
	}
	ast.Inspect(program, func(node ast.Element) bool {
		decl, ok := node.(ast.Declaration)
		if !ok {
			return true
		}
		access := decl.DeclarationAccess()
		if access == nil {
			return true
		}
		// Find the last entitlement element.
		var lastEntitlement ast.Element
		access.Walk(func(child ast.Element) {
			if child != nil {
				lastEntitlement = child
			}
		})
		if lastEntitlement == nil {
			return true
		}
		// Scan forward from the last entitlement's end position to find the
		// closing `)` of the access modifier.
		startScan := lastEntitlement.EndPosition(nil).Offset + 1
		closeOffset := -1
		for i := startScan; i < len(source); i++ {
			if source[i] == ')' {
				closeOffset = i
				break
			}
		}
		if closeOffset < 0 {
			return true
		}
		// Iterate the declaration's direct children's leading-comment maps and
		// hoist any comment positionally inside (lastEntitlement.End, closeOffset).
		decl.Walk(func(child ast.Element) {
			if child == nil {
				return
			}
			groups := cm.Leading[child]
			if len(groups) == 0 {
				return
			}
			keep := groups[:0]
			var hoisted []*CommentGroup
			for _, g := range groups {
				gStart := g.StartPos().Offset
				if gStart > lastEntitlement.EndPosition(nil).Offset && gStart < closeOffset {
					hoisted = append(hoisted, g)
				} else {
					keep = append(keep, g)
				}
			}
			if len(hoisted) > 0 {
				if len(keep) == 0 {
					delete(cm.Leading, child)
				} else {
					cm.Leading[child] = keep
				}
				// Place the first hoisted comment in the same-line slot so it
				// renders inline (e.g., `access(A  /* note */)`). Any additional
				// hoisted comments go to trailing.
				for _, g := range hoisted {
					if cm.SameLine[lastEntitlement] == nil {
						cm.SameLine[lastEntitlement] = g
					} else {
						cm.Trailing[lastEntitlement] = append(cm.Trailing[lastEntitlement], g)
					}
				}
			}
		})
		return true
	})
}

// hoistVarDeclTypeComments walks the program and, for every VariableDeclaration
// that has both a type annotation and a value, moves any trailing or same-line
// comment attached to the type annotation to the leading-comment slot of the
// value. This keeps the comment on its own line between `: T` and `= value`.
// It also hoists trailing comments on an invocation's InvokedExpression to the
// leading of the first argument when those comments fall after the opening
// paren (between `func(` and the first argument).
func hoistVarDeclTypeComments(cm *CommentMap, program *ast.Program) {
	ast.Inspect(program, func(node ast.Element) bool {
		switch d := node.(type) {
		case *ast.VariableDeclaration:
			if d.TypeAnnotation != nil && d.Value != nil {
				// Move trailing first, then same-line, so the resulting
				// leading order matches source order (same-line was before
				// trailing in source).
				cm.MoveTrailingToLeading(d.TypeAnnotation, d.Value)
				cm.MoveSameLineToLeading(d.TypeAnnotation, d.Value)
			}
		case *ast.PragmaDeclaration:
			// Comments inside an empty pragma `#()` get attached to the
			// VoidExpression. They'd render between `#` and `()`, producing
			// invalid Cadence. Hoist them to trailing of the pragma itself.
			if _, isVoid := d.Expression.(*ast.VoidExpression); isVoid {
				lead, same, trail := cm.TakeRaw(d.Expression)
				if same != nil {
					cm.Trailing[d] = append(cm.Trailing[d], same)
				}
				cm.Trailing[d] = append(cm.Trailing[d], lead...)
				cm.Trailing[d] = append(cm.Trailing[d], trail...)
			}
		case *ast.InvocationExpression:
			// Comments attached as trailing to the InvokedExpression that
			// fall after the opening paren (i.e., inside the argument list)
			// belong to the first argument's leading position.
			if len(d.Arguments) == 0 {
				return true
			}
			invoked := d.InvokedExpression
			trailing := cm.Trailing[invoked]
			if len(trailing) == 0 {
				return true
			}
			parenOffset := d.ArgumentsStartPos.Offset
			keep := trailing[:0]
			var hoist []*CommentGroup
			for _, g := range trailing {
				if g.StartPos().Offset > parenOffset {
					hoist = append(hoist, g)
				} else {
					keep = append(keep, g)
				}
			}
			if len(hoist) == 0 {
				return true
			}
			if len(keep) == 0 {
				delete(cm.Trailing, invoked)
			} else {
				cm.Trailing[invoked] = keep
			}
			firstArg := d.Arguments[0]
			cm.Leading[firstArg] = append(hoist, cm.Leading[firstArg]...)
		}
		return true
	})
}

// attachLevel distributes comment groups among a sequence of sibling elements.
// It recurses into each element's children for groups that fall inside the element.
// Returns any groups not consumed (after the last sibling).
func attachLevel(cm *CommentMap, siblings []ast.Element, groups []*CommentGroup, isTopLevel bool, source []byte) []*CommentGroup {
	if len(groups) == 0 {
		return nil
	}

	if len(siblings) == 0 {
		if isTopLevel {
			cm.HeaderComments = append(cm.HeaderComments, groups...)
			return nil
		}
		return groups
	}

	gi := 0 // index into groups

	// Groups before first sibling
	firstStart := siblings[0].StartPosition()
	for gi < len(groups) && groups[gi].EndPos().Offset < firstStart.Offset {
		if isTopLevel {
			// Check if this is the last group before the first decl
			nextGi := gi + 1
			isLastBefore := nextGi >= len(groups) || groups[nextGi].EndPos().Offset >= firstStart.Offset

			if !isLastBefore || blankLineBetween(groups[gi].EndPos(), firstStart) {
				cm.HeaderComments = append(cm.HeaderComments, groups[gi])
			} else {
				cm.Leading[siblings[0]] = append(cm.Leading[siblings[0]], groups[gi])
			}
		} else {
			cm.Leading[siblings[0]] = append(cm.Leading[siblings[0]], groups[gi])
		}
		gi++
	}

	// Process each sibling
	for si := 0; si < len(siblings); si++ {
		node := siblings[si]
		nodeStart := node.StartPosition()
		// nodeEndRaw is what the parser reports — used for the inside check
		// because comments physically inside the un-clipped span belong to
		// descendants of node.
		nodeEndRaw := node.EndPosition(nil)
		// nodeEnd is the syntactic end (last non-whitespace byte). Used for
		// sameLine and between-sibling decisions because some upstream
		// constructs report an end position past their closing token (e.g.
		// VoidExpression `()` whose EndPos is the start of the *next* token,
		// which on multi-line input lands on the next line's indent and pulls
		// a following comment into a spurious sameLine attachment).
		nodeEnd := trueEndPosition(nodeEndRaw, source)

		// Collect groups that fall inside this node (start after node start, end at or before node end)
		var inside []*CommentGroup
		for gi < len(groups) {
			g := groups[gi]
			gStart := g.StartPos()
			gEnd := g.EndPos()

			if gStart.Offset > nodeStart.Offset && gEnd.Offset <= nodeEndRaw.Offset {
				inside = append(inside, g)
				gi++
				continue
			}
			break
		}

		// Recursively handle inside groups
		if len(inside) > 0 {
			children := getChildren(node)
			leftover := attachLevel(cm, children, inside, false, source)
			// Leftover from inside = trailing of last child, or dangling
			if len(leftover) > 0 {
				if len(children) > 0 {
					lastChild := children[len(children)-1]
					cm.Trailing[lastChild] = append(cm.Trailing[lastChild], leftover...)
				} else {
					// Dangling: no children, attach as leading of this node
					cm.Leading[node] = append(cm.Leading[node], leftover...)
				}
			}
		}

		// Same-line comment: on same line as node end, after the node
		if gi < len(groups) {
			g := groups[gi]
			if g.StartPos().Line == nodeEnd.Line && g.StartPos().Offset > nodeEnd.Offset {
				// Make sure it's not inside the next sibling
				isBeforeNext := si+1 >= len(siblings) || g.EndPos().Offset < siblings[si+1].StartPosition().Offset
				if isBeforeNext {
					cm.SameLine[node] = g
					gi++
				}
			}
		}

		// Groups between this sibling and the next
		if si+1 < len(siblings) {
			nextStart := siblings[si+1].StartPosition()

			for gi < len(groups) && groups[gi].EndPos().Offset < nextStart.Offset {
				g := groups[gi]
				// Disambiguation heuristic for comments between siblings:
				// 1. Same-line comments are handled above
				// 2. Blank line between previous sibling end and comment → Leading of next
				// 3. No blank line (adjacent) → Trailing of previous
				if blankLineBetween(nodeEnd, g.StartPos()) {
					cm.Leading[siblings[si+1]] = append(cm.Leading[siblings[si+1]], g)
				} else {
					cm.Trailing[node] = append(cm.Trailing[node], g)
				}
				gi++
			}
		}
	}

	// Groups after the last sibling: mirror the "between siblings" heuristic.
	// Without this, comments on the next line after the last sibling are
	// left unconsumed and incorrectly classified as footer/header by the caller.
	if len(siblings) > 0 && gi < len(groups) {
		lastNode := siblings[len(siblings)-1]
		lastEnd := trueEndPosition(lastNode.EndPosition(nil), source)
		if sl := cm.SameLine[lastNode]; sl != nil {
			lastEnd = sl.EndPos()
		}
		for gi < len(groups) {
			g := groups[gi]
			if blankLineBetween(lastEnd, g.StartPos()) {
				break
			}
			cm.Trailing[lastNode] = append(cm.Trailing[lastNode], g)
			gi++
			lastEnd = g.EndPos()
		}
	}

	// Return unconsumed groups
	return groups[gi:]
}

// trueEndPosition returns the position of the last non-whitespace byte at or
// before reportedEnd.Offset. Compensates for upstream parser quirks where
// some node EndPositions land on the next token's whitespace — notably
// VoidExpression `()`, whose EndPos is set to p.current.EndPos AFTER consuming
// `)`, which on multi-line input is the start of the next line's indent.
// Without this, a comment on the line after such a node attaches as SameLine,
// which differs from how a re-parse classifies the same comment after the
// formatter has normalized the layout, breaking idempotence.
//
// Column is left as-is since attach.go only reads Line and Offset for its
// decisions.
func trueEndPosition(reportedEnd ast.Position, source []byte) ast.Position {
	if len(source) == 0 {
		return reportedEnd
	}
	offset := reportedEnd.Offset
	if offset >= len(source) {
		offset = len(source) - 1
	}
	line := reportedEnd.Line
	for offset > 0 {
		b := source[offset]
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			break
		}
		// About to move offset → offset-1. The line decreases when the byte
		// we're moving onto is `\n` (the `\n` itself is on the prior line).
		if source[offset-1] == '\n' {
			line--
		}
		offset--
	}
	return ast.Position{Offset: offset, Line: line, Column: reportedEnd.Column}
}

// getChildren returns the direct children of an AST element, sorted by position.
// Children from access modifier entitlement types are excluded so that comments
// between the access modifier and the declaration keyword are not misclassified
// as trailing on the NominalType inside access(X). Comments that fall
// positionally inside `access(...)` are handled by hoistAccessInlineComments
// at post-attach time.
func getChildren(node ast.Element) []ast.Element {
	excluded := map[ast.Element]bool{}
	if decl, ok := node.(ast.Declaration); ok {
		access := decl.DeclarationAccess()
		if access != nil {
			access.Walk(func(child ast.Element) {
				if child != nil {
					excluded[child] = true
				}
			})
		}
	}

	var children []ast.Element
	node.Walk(func(child ast.Element) {
		if child != nil && !excluded[child] {
			children = append(children, child)
		}
	})
	sort.Slice(children, func(i, j int) bool {
		return children[i].StartPosition().Offset < children[j].StartPosition().Offset
	})
	return children
}

// HasTrailing returns true if the element has trailing comment groups.
func (cm *CommentMap) HasTrailing(n ast.Element) bool {
	return len(cm.Trailing[n]) > 0
}

// HasLeadingLineComment reports whether n has a leading `//`-style comment.
// Does not remove comments from the map.
func (cm *CommentMap) HasLeadingLineComment(n ast.Element) bool {
	for _, g := range cm.Leading[n] {
		for _, c := range g.Comments {
			if c.Kind == KindLine || c.Kind == KindDocLine {
				return true
			}
		}
	}
	return false
}

// MoveTrailingToLeading transfers all comment groups from cm.Trailing[from]
// to the front of cm.Leading[to]. Used by renderers that need to render an
// inter-token comment as leading-of-next instead of trailing-of-prev so the
// comment renders in a position that re-parses to the same attach key
// (avoids idempotence flips where a comment between two tokens of one
// declaration lands as Trailing on one pass and SameLine/Leading on another).
func (cm *CommentMap) MoveTrailingToLeading(from, to ast.Element) {
	trailing := cm.Trailing[from]
	if len(trailing) == 0 {
		return
	}
	delete(cm.Trailing, from)
	cm.Leading[to] = append(trailing, cm.Leading[to]...)
}

// MoveSameLineToLeading moves the same-line comment from cm.SameLine[from] to
// the front of cm.Leading[to]. No-op if SameLine[from] is empty. See
// MoveTrailingToLeading for rationale.
func (cm *CommentMap) MoveSameLineToLeading(from, to ast.Element) {
	g := cm.SameLine[from]
	if g == nil {
		return
	}
	delete(cm.SameLine, from)
	cm.Leading[to] = append([]*CommentGroup{g}, cm.Leading[to]...)
}

// blankLineBetween returns true if there is at least one blank line between
// positions a and b (i.e., the line gap is > 1).
func blankLineBetween(a, b ast.Position) bool {
	return b.Line-a.Line > 1
}
