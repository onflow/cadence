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
	// JumpOffsets contains the offsets of all jumps
	// (break or continue statements), potential or definite.
	//
	// If non-empty, indicates that (the branch of) the function
	// contains a potential break or continue statement
	JumpOffsets *persistent.OrderedSet[int]
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
	// This is the generic "every path terminated" flag
	// and is the one observed by `IsUnreachable()`.
	//
	// NOTE: It is intentionally NOT just
	// `DefinitelyReturned || DefinitelyHalted || DefinitelyJumpedLoop || DefinitelyJumpedSwitch`,
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
	// At a case body's terminal state and at a loop body's terminal state,
	// `DefinitelyExited` is cleared alongside `DefinitelyReturned` and `DefinitelyHalted`
	// when a corresponding `MaybeJumped*` is true,
	// because the maybe-jumping path escapes the construct without terminating the function.
	DefinitelyExited bool
	// DefinitelyJumpedLoop indicates that (the branch of) the function
	// contains a definite jump to an enclosing loop
	// (a `break` targeting a loop, or a `continue`).
	DefinitelyJumpedLoop bool
	// MaybeJumped indicates that some path within the current loop
	// reached a jump targeting that loop (a `break` whose target is the
	// loop, or a `continue`).
	//
	// This is the "Maybe" counterpart to `DefinitelyJumpedLoop`, OR-merged
	// across branches and not cleared by subsequent statements within
	// the loop. It mirrors `MaybeJumpedSwitch` and is scoped to the loop:
	// cleared by `WithLoop` on exit, since such jumps are consumed by
	// the loop and are irrelevant outside it.
	MaybeJumpedLoop bool
	// DefinitelyJumpedSwitch indicates that (the branch of) the function
	// contains a definite `break` whose target is a switch.
	//
	// Tracked separately from DefinitelyJumpedLoop because a `break` inside a switch
	// only exits the switch; it must not leak past the switch boundary as a
	// jump to an outer loop. The flag is observed for unreachability inside the
	// switch and is cleared by WithSwitch on exit.
	DefinitelyJumpedSwitch bool
	// MaybeJumpedSwitch indicates that some path within the current switch
	// reached a `break` statement targeting that switch.
	//
	// Unlike `DefinitelyJumpedSwitch`, this is OR-merged across branches and
	// is not cleared by subsequent statements within the switch. It is used
	// at a case body's terminal state to detect the pattern in which only
	// some paths reach the case's trailing termination, e.g.
	//
	//   case x:
	//       if cond { break }
	//       return ...
	//
	// Such a case must not be treated as "definitely returns" when merging
	// case results at the switch level, because the break path means the
	// switch as a whole can still fall through to the code after it.
	//
	// Like `DefinitelyJumpedSwitch`, the flag is scoped to the switch and
	// cleared by WithSwitch on exit.
	MaybeJumpedSwitch bool
}

func NewReturnInfo() *ReturnInfo {
	return &ReturnInfo{
		JumpOffsets: persistent.NewOrderedSet[int](nil),
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

	ri.DefinitelyJumpedLoop = ri.DefinitelyJumpedLoop ||
		(thenReturnInfo.DefinitelyJumpedLoop &&
			elseReturnInfo.DefinitelyJumpedLoop)

	ri.DefinitelyJumpedSwitch = ri.DefinitelyJumpedSwitch ||
		(thenReturnInfo.DefinitelyJumpedSwitch &&
			elseReturnInfo.DefinitelyJumpedSwitch)

	ri.DefinitelyHalted = ri.DefinitelyHalted ||
		(thenReturnInfo.DefinitelyHalted &&
			elseReturnInfo.DefinitelyHalted)

	ri.DefinitelyExited = ri.DefinitelyExited ||
		(thenReturnInfo.DefinitelyExited &&
			elseReturnInfo.DefinitelyExited)
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
	return result
}

func (ri *ReturnInfo) IsUnreachable() bool {
	// NOTE: intentionally NOT DefinitelyReturned || DefinitelyHalted || DefinitelyJumpedLoop || DefinitelyJumpedSwitch,
	// see DefinitelyExited
	return ri.DefinitelyExited
}

func (ri *ReturnInfo) AddJumpOffset(offset int) {
	ri.JumpOffsets.Add(offset)
}

func (ri *ReturnInfo) WithNewJumpTarget(f func()) {
	ri.JumpOffsets = ri.JumpOffsets.Clone()
	f()
	ri.JumpOffsets = ri.JumpOffsets.Parent
}
