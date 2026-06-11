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

package sema

import (
	"github.com/onflow/cadence/common/persistent"
)

// TODO: rename to e.g. ControlFlowInfo

// ReturnInfo tracks control-flow information
type ReturnInfo struct {
	// LoopJumpOffsets contains the offsets of all jumps targeting a loop
	// (`continue` statements, and `break` statements whose innermost
	// enclosing control construct is a loop), potential or definite.
	//
	// Kept separate from SwitchJumpOffsets because the two kinds of jumps
	// have different lifetimes: a loop-targeting jump is consumed by the
	// loop (`WithLoop` scopes this set), while a switch-targeting `break`
	// is consumed by the switch (`WithSwitch` scopes SwitchJumpOffsets).
	// In particular, a `continue` inside a switch case targets the
	// enclosing loop, so its offset must survive past the switch boundary
	// up to the loop — it must not be discarded when the switch ends.
	LoopJumpOffsets *persistent.OrderedSet[int]
	// SwitchJumpOffsets contains the offsets of all `break` statements
	// targeting a switch, potential or definite.
	//
	// Scoped to the switch by `WithSwitch`: once the switch has been
	// checked, breaks targeting it are consumed and are irrelevant
	// to the code that follows.
	SwitchJumpOffsets *persistent.OrderedSet[int]
	// MaybeReturned indicates that (the branch of) the function
	// contains a potentially taken return statement
	MaybeReturned bool
	// DefinitelyReturned indicates that (the branch of) the function
	// contains a definite return statement
	DefinitelyReturned bool
	// DefinitelyHalted indicates that (the branch of) the function
	// contains a definite halt (a function call with a Never return type)
	DefinitelyHalted bool
	// DefinitelyExited indicates that (the branch of) the function
	// definitely terminated control flow on every path — by
	// - `return`,
	// - halt (a function call with a `Never` return type),
	// - `break` (targeting a loop or a switch), or
	// - `continue`.
	//
	// This is the generic "every path terminated" flag and is the one
	// observed by `IsUnreachable()` to decide whether subsequent
	// statements are reachable.
	//
	// NOTE: It is intentionally NOT just `DefinitelyReturned || DefinitelyHalted`,
	// because AND-merging each kind-specific flag separately would lose
	// the case where both branches of an if-else terminate but via different kinds.
	// For example:
	//
	//   if ... {
	//       return
	//
	//       // DefinitelyReturned = true
	//       // DefinitelyHalted = false
	//       // DefinitelyExited = true
	//   } else {
	//       panic(...)
	//
	//       // DefinitelyReturned = false
	//       // DefinitelyHalted = true
	//       // DefinitelyExited = true
	//   }
	//   // (AND-merges of flags in both branches)
	//   // DefinitelyReturned = false
	//   // DefinitelyHalted = false
	//   // DefinitelyExited = true
	//
	// The same logic applies to `if break else return`,
	// `if continue else return`, etc.
	//
	// Because `DefinitelyExited` is broadened to also include break and
	// continue, it can be true even when the function itself has not
	// exited (the break/continue exits only the enclosing loop/switch).
	// Sites that care specifically about "did the function exit" (notably
	// `checkResourceLoss`) recover that by checking
	// `DefinitelyExited && !MaybeJumped()` — see `checkResourceLoss`
	// for the rationale.
	//
	// At a case body's terminal state and at a loop body's terminal state,
	// `DefinitelyExited` is cleared alongside `DefinitelyReturned` and `DefinitelyHalted`
	// when a corresponding `MaybeJumped*` is true,
	// because the maybe-jumping path escapes the construct without terminating the function.
	DefinitelyExited bool
	// MaybeJumpedLoop indicates that some path within the current loop
	// reached a jump targeting that loop (a `break` whose target is the
	// loop, or a `continue`).
	//
	// OR-merged across branches and not cleared by subsequent statements
	// within the loop. Scoped to the loop: cleared by `WithLoop` on exit,
	// since such jumps are consumed by the loop and are irrelevant
	// outside it.
	//
	// At a loop body's terminal state, used to clear DR/DH/DE so that the
	// surrounding scope's "definitely returns/halts/exits" claim is not
	// over-claimed (the jumping path falls past the loop, not the function).
	MaybeJumpedLoop bool
	// MaybeJumpedSwitch indicates that some path within the current switch
	// reached a `break` statement targeting that switch.
	//
	// OR-merged across branches and not cleared by subsequent statements
	// within the switch. Scoped to the switch: cleared by `WithSwitch` on
	// exit, since such breaks are consumed by the switch and are irrelevant
	// outside it.
	//
	// At a case body's terminal state, used to clear DR/DH/DE so that the
	// switch-level merge does not over-claim definite-return — the break
	// path means the switch as a whole can still fall through to the code
	// after it. For example:
	//
	//   case x:
	//       if cond { break }
	//       return ...
	MaybeJumpedSwitch bool
}

func NewReturnInfo() *ReturnInfo {
	return &ReturnInfo{
		LoopJumpOffsets:   persistent.NewOrderedSet[int](nil),
		SwitchJumpOffsets: persistent.NewOrderedSet[int](nil),
	}
}

func (ri *ReturnInfo) MergeBranches(thenReturnInfo *ReturnInfo, elseReturnInfo *ReturnInfo) {
	ri.MaybeReturned = ri.MaybeReturned ||
		thenReturnInfo.MaybeReturned ||
		elseReturnInfo.MaybeReturned

	ri.MaybeJumpedLoop = ri.MaybeJumpedLoop ||
		thenReturnInfo.MaybeJumpedLoop ||
		elseReturnInfo.MaybeJumpedLoop

	ri.MaybeJumpedSwitch = ri.MaybeJumpedSwitch ||
		thenReturnInfo.MaybeJumpedSwitch ||
		elseReturnInfo.MaybeJumpedSwitch

	ri.DefinitelyReturned = ri.DefinitelyReturned ||
		(thenReturnInfo.DefinitelyReturned &&
			elseReturnInfo.DefinitelyReturned)

	ri.DefinitelyHalted = ri.DefinitelyHalted ||
		(thenReturnInfo.DefinitelyHalted &&
			elseReturnInfo.DefinitelyHalted)

	ri.DefinitelyExited = ri.DefinitelyExited ||
		(thenReturnInfo.DefinitelyExited &&
			elseReturnInfo.DefinitelyExited)

	// Propagate jump offsets from both branches. Each branch's
	// `Clone()` gave it separate child jump-offset sets, so jumps in
	// one branch did not pollute the sibling's view of "did a jump
	// occur between this resource's declaration and its use?"
	// (see `maybeAddResourceInvalidation`). Now that both branches
	// are joining back into the parent, their jumps are potential
	// from the perspective of subsequent code.
	ri.addJumpOffsetsFrom(thenReturnInfo)
	ri.addJumpOffsetsFrom(elseReturnInfo)
}

func (ri *ReturnInfo) addJumpOffsetsFrom(other *ReturnInfo) {
	_ = other.LoopJumpOffsets.ForEach(func(offset int) error {
		ri.LoopJumpOffsets.Add(offset)
		return nil
	})
	_ = other.SwitchJumpOffsets.ForEach(func(offset int) error {
		ri.SwitchJumpOffsets.Add(offset)
		return nil
	})
}

func (ri *ReturnInfo) MergePotentiallyUnevaluated(temporaryReturnInfo *ReturnInfo) {
	ri.MaybeReturned = ri.MaybeReturned ||
		temporaryReturnInfo.MaybeReturned

	ri.MaybeJumpedLoop = ri.MaybeJumpedLoop ||
		temporaryReturnInfo.MaybeJumpedLoop

	ri.MaybeJumpedSwitch = ri.MaybeJumpedSwitch ||
		temporaryReturnInfo.MaybeJumpedSwitch

	// NOTE: the definitive return state does not change
}

func (ri *ReturnInfo) Clone() *ReturnInfo {
	result := NewReturnInfo()
	*result = *ri
	// Clone the jump-offset sets so that jumps recorded in this clone
	// do not leak into sibling clones (e.g. then vs. else branch).
	result.LoopJumpOffsets = ri.LoopJumpOffsets.Clone()
	result.SwitchJumpOffsets = ri.SwitchJumpOffsets.Clone()
	return result
}

func (ri *ReturnInfo) IsUnreachable() bool {
	// NOTE: intentionally NOT DefinitelyReturned || DefinitelyHalted,
	// see DefinitelyExited
	return ri.DefinitelyExited
}

// MaybeJumped reports whether some path within the current loop or
// switch reached a `break` or `continue`. It is the union of
// `MaybeJumpedLoop` and `MaybeJumpedSwitch`.
//
// Used to distinguish "function exited via return/halt" (no maybe-jump)
// from "construct exited via break/continue" (maybe-jump set), notably
// in `checkResourceLoss` (to decide whether to skip) and in
// `maybeAddResourceInvalidation` (to decide whether a member resource
// invalidation is only potential).
func (ri *ReturnInfo) MaybeJumped() bool {
	return ri.MaybeJumpedLoop || ri.MaybeJumpedSwitch
}

// clearDefiniteExits clears the kind-specific "definitely exited"
// flags. Used by `WithLoop`/`WithSwitch` when a path through the
// construct took a break/continue — the construct as a whole did NOT
// terminate the function on every path, so a "definitely returned/
// halted/exited" claim would over-promise.
func (ri *ReturnInfo) clearDefiniteExits() {
	ri.DefinitelyReturned = false
	ri.DefinitelyHalted = false
	ri.DefinitelyExited = false
}

func (ri *ReturnInfo) AddLoopJumpOffset(offset int) {
	ri.LoopJumpOffsets.Add(offset)
}

func (ri *ReturnInfo) AddSwitchJumpOffset(offset int) {
	ri.SwitchJumpOffsets.Add(offset)
}

// WithNewLoopJumpTarget runs f with a fresh child set of loop jump
// offsets, and discards the child set afterwards: jumps targeting the
// loop are consumed by it and are irrelevant to the code that follows.
// Switch jump offsets are NOT scoped here — a switch-targeting `break`
// is always consumed by its switch, which lies entirely inside or
// outside the loop, so its offsets can never escape into the loop's set.
func (ri *ReturnInfo) WithNewLoopJumpTarget(f func()) {
	ri.LoopJumpOffsets = ri.LoopJumpOffsets.Clone()
	f()
	ri.LoopJumpOffsets = ri.LoopJumpOffsets.Parent
}

// WithNewSwitchJumpTarget runs f with a fresh child set of switch jump
// offsets, and discards the child set afterwards: breaks targeting the
// switch are consumed by it and are irrelevant to the code that follows.
// Loop jump offsets are intentionally NOT scoped here — a `continue`
// inside a switch case targets the enclosing loop, so its offset must
// survive past the switch boundary up to the loop.
func (ri *ReturnInfo) WithNewSwitchJumpTarget(f func()) {
	ri.SwitchJumpOffsets = ri.SwitchJumpOffsets.Clone()
	f()
	ri.SwitchJumpOffsets = ri.SwitchJumpOffsets.Parent
}
