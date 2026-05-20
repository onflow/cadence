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

import "slices"

// ControlKind identifies the kind of a control-flow construct
// that `break` or `continue` can target.
type ControlKind uint8

const (
	ControlKindNone ControlKind = iota
	ControlKindLoop
	ControlKindSwitch
)

type FunctionActivation struct {
	ReturnType           Type
	ReturnInfo           *ReturnInfo
	InitializationInfo   *InitializationInfo
	ControlStack         []ControlKind
	ValueActivationDepth int
}

func (a *FunctionActivation) InLoop() bool {
	return slices.Contains(a.ControlStack, ControlKindLoop)
}

// InnermostControl returns the kind of the innermost enclosing
// control-flow construct (loop or switch), if any.
// This is the target of a `break` statement.
func (a *FunctionActivation) InnermostControl() ControlKind {
	if len(a.ControlStack) == 0 {
		return ControlKindNone
	}
	return a.ControlStack[len(a.ControlStack)-1]
}

func (a *FunctionActivation) pushControl(kind ControlKind) {
	a.ControlStack = append(a.ControlStack, kind)
}

func (a *FunctionActivation) popControl() {
	lastIndex := len(a.ControlStack) - 1
	a.ControlStack = a.ControlStack[:lastIndex]
}

func (a *FunctionActivation) WithLoop(f func()) {
	a.pushControl(ControlKindLoop)
	// MaybeJumpedLoop is scoped to this loop: save on entry, restore on exit,
	// so a nested loop's break/continue does not affect an enclosing loop.
	// `break` and `continue` statements targeting this loop
	// are consumed by the loop and must not leak to an enclosing scope.
	savedMaybeJumpedLoop := a.ReturnInfo.MaybeJumpedLoop
	a.ReturnInfo.WithNewJumpTarget(f)
	// If any path within the loop body jumped (break/continue),
	// the body did not definitely return/halt/exit on every path:
	// clear the corresponding flags so subsequent reasoning sees an
	// accurate per-body result.
	if a.ReturnInfo.MaybeJumpedLoop {
		a.ReturnInfo.DefinitelyReturned = false
		a.ReturnInfo.DefinitelyHalted = false
		a.ReturnInfo.DefinitelyExited = false
	}
	a.ReturnInfo.MaybeJumpedLoop = savedMaybeJumpedLoop
	a.popControl()
}

func (a *FunctionActivation) WithSwitch(f func()) {
	// NOTE: new jump-offsets child-set for each case instead of whole switch
	a.pushControl(ControlKindSwitch)
	// `break` statements inside a switch only target the switch, not any enclosing loop.
	// MaybeJumpedSwitch is scoped to this switch: save on entry, restore on exit,
	// so a nested switch's break does not affect an enclosing switch.
	// (`continue` statements always target the enclosing loop,
	// so it sets MaybeJumpedLoop (not MaybeJumpedSwitch)
	// and therefore propagates past the switch.)
	savedMaybeJumpedSwitch := a.ReturnInfo.MaybeJumpedSwitch
	f()
	a.ReturnInfo.MaybeJumpedSwitch = savedMaybeJumpedSwitch
	a.popControl()
}

type FunctionActivations struct {
	activations []*FunctionActivation
}

func (a *FunctionActivations) IsLocal() bool {
	currentFunctionDepth := -1

	currentFunctionActivation := a.Current()
	if currentFunctionActivation != nil {
		currentFunctionDepth = currentFunctionActivation.ValueActivationDepth
	}

	return currentFunctionDepth > 0
}

func (a *FunctionActivations) EnterFunction(functionType *FunctionType, valueActivationDepth int) *FunctionActivation {
	activation := &FunctionActivation{
		ReturnType:           functionType.ReturnTypeAnnotation.Type,
		ValueActivationDepth: valueActivationDepth,
		ReturnInfo:           NewReturnInfo(),
	}
	a.activations = append(a.activations, activation)
	return activation
}

func (a *FunctionActivations) LeaveFunction() {
	lastIndex := len(a.activations) - 1
	a.activations[lastIndex] = nil
	a.activations = a.activations[:lastIndex]
}

func (a *FunctionActivations) WithFunction(
	functionType *FunctionType,
	valueActivationDepth int,
	f func(activation *FunctionActivation),
) {
	activation := a.EnterFunction(functionType, valueActivationDepth)
	f(activation)
	a.LeaveFunction()
}

func (a *FunctionActivations) Current() *FunctionActivation {
	lastIndex := len(a.activations) - 1
	if lastIndex < 0 {
		return nil
	}
	return a.activations[lastIndex]
}
