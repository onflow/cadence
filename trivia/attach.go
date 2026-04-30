package trivia

import (
	"fmt"
	"sort"

	"github.com/onflow/cadence/ast"
)

// CommentMap binds comment groups to AST nodes by position class.
// Take() removes and returns comments for a node — this ensures each
// comment is emitted exactly once during rendering. After rendering,
// the map should be empty; any leftovers indicate a bug.
type CommentMap struct {
	Header   []*CommentGroup // before first declaration
	Footer   []*CommentGroup // after last declaration
	Leading  map[ast.Element][]*CommentGroup
	Trailing map[ast.Element][]*CommentGroup
	SameLine map[ast.Element]*CommentGroup // at most one per node
}

// NewCommentMap creates an empty CommentMap with initialized maps.
func NewCommentMap() *CommentMap {
	return &CommentMap{
		Leading:  make(map[ast.Element][]*CommentGroup),
		Trailing: make(map[ast.Element][]*CommentGroup),
		SameLine: make(map[ast.Element]*CommentGroup),
	}
}

// Take removes and returns all comments associated with n.
func (cm *CommentMap) Take(n ast.Element) (leading []*CommentGroup, sameLine *CommentGroup, trailing []*CommentGroup) {
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
	h := cm.Header
	cm.Header = nil
	return h
}

// TakeFooter removes and returns footer comments.
func (cm *CommentMap) TakeFooter() []*CommentGroup {
	f := cm.Footer
	cm.Footer = nil
	return f
}

// IsEmpty returns true if no comments remain in the map.
func (cm *CommentMap) IsEmpty() bool {
	return len(cm.Header) == 0 &&
		len(cm.Footer) == 0 &&
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
	if len(groups) == 0 {
		return cm
	}

	decls := program.Declarations()
	elements := make([]ast.Element, len(decls))
	for i, d := range decls {
		elements[i] = d
	}

	remaining := attachLevel(cm, elements, groups, true)

	// Anything left over is footer
	cm.Footer = append(cm.Footer, remaining...)
	return cm
}

// attachLevel distributes comment groups among a sequence of sibling elements.
// It recurses into each element's children for groups that fall inside the element.
// Returns any groups not consumed (after the last sibling).
func attachLevel(cm *CommentMap, siblings []ast.Element, groups []*CommentGroup, isTopLevel bool) []*CommentGroup {
	if len(groups) == 0 {
		return nil
	}

	if len(siblings) == 0 {
		if isTopLevel {
			cm.Header = append(cm.Header, groups...)
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
				cm.Header = append(cm.Header, groups[gi])
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
		nodeEnd := node.EndPosition(nil)

		// Collect groups that fall inside this node (start after node start, end at or before node end)
		var inside []*CommentGroup
		for gi < len(groups) {
			g := groups[gi]
			gStart := g.StartPos()
			gEnd := g.EndPos()

			if gStart.Offset > nodeStart.Offset && gEnd.Offset <= nodeEnd.Offset {
				inside = append(inside, g)
				gi++
				continue
			}
			break
		}

		// Recursively handle inside groups
		if len(inside) > 0 {
			children := getChildren(node)
			leftover := attachLevel(cm, children, inside, false)
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
		lastEnd := lastNode.EndPosition(nil)
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

// getChildren returns the direct children of an AST element, sorted by position.
// Children from access modifier entitlement types are excluded so that comments
// between the access modifier and the declaration keyword are not misclassified
// as trailing on the NominalType inside access(X).
func getChildren(node ast.Element) []ast.Element {
	// Collect elements from the access modifier so we can exclude them.
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

// MoveTrailingLineCommentsToLeading transfers any line-comment groups from
// cm.Trailing[from] to the front of cm.Leading[to]. Block-comment groups stay
// where they are. Used by renderers that need to render an inter-token
// line comment as leading-of-next instead of trailing-of-prev so that the
// `//` lands on its own line in output.
func (cm *CommentMap) MoveTrailingLineCommentsToLeading(from, to ast.Element) {
	trailing := cm.Trailing[from]
	if len(trailing) == 0 {
		return
	}
	var keep []*CommentGroup
	var move []*CommentGroup
	for _, g := range trailing {
		if len(g.Comments) > 0 {
			last := g.Comments[len(g.Comments)-1]
			if last.Kind == KindLine || last.Kind == KindDocLine {
				move = append(move, g)
				continue
			}
		}
		keep = append(keep, g)
	}
	if len(move) == 0 {
		return
	}
	if len(keep) == 0 {
		delete(cm.Trailing, from)
	} else {
		cm.Trailing[from] = keep
	}
	cm.Leading[to] = append(move, cm.Leading[to]...)
}

// MoveSameLineLineCommentToLeading moves a `//`-style same-line comment from
// cm.SameLine[from] to the front of cm.Leading[to]. No-op if SameLine[from]
// is empty or is a block comment. Used so a sameLine line comment on a
// non-terminal child (e.g. TypeAnnotation inside VariableDeclaration) is
// re-rendered as leading of the next sibling, otherwise the wrapWithComments
// emit of `node  //` is followed by parent-emitted tokens on the same line
// and the comment swallows them.
func (cm *CommentMap) MoveSameLineLineCommentToLeading(from, to ast.Element) {
	g := cm.SameLine[from]
	if g == nil || len(g.Comments) == 0 {
		return
	}
	last := g.Comments[len(g.Comments)-1]
	if last.Kind != KindLine && last.Kind != KindDocLine {
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
