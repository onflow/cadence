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

// WithLoop runs f within a new loop scope. `MaybeJumpedLoop` is
// save/restored — break/continue targeting this loop are consumed by it
// and must not leak. If any path jumped, the body's DR/DH/DE are
// cleared (they over-claim — the jumping path didn't terminate the
// function).
func (a *FunctionActivation) WithLoop(f func()) {
	a.pushControl(ControlKindLoop)
	savedMaybeJumpedLoop := a.ReturnInfo.MaybeJumpedLoop
	a.ReturnInfo.WithNewLoopJumpTarget(f)
	if a.ReturnInfo.MaybeJumpedLoop {
		a.ReturnInfo.clearDefiniteExits()
	}
	a.ReturnInfo.MaybeJumpedLoop = savedMaybeJumpedLoop
	a.popControl()
}

// WithSwitch runs f within a new switch scope. Mirrors `WithLoop`:
// `MaybeJumpedSwitch` is save/restored (break targeting this switch is
// consumed by it; continue targets the enclosing loop and propagates
// past via `MaybeJumpedLoop`). If any case body broke, clear DR/DH/DE
// — the break path falls past the switch without terminating the
// function, so the merged "every case definitely terminated" claim
// would over-promise.
func (a *FunctionActivation) WithSwitch(f func()) {
	// The switch jump-offset set is scoped to the whole switch:
	// breaks targeting the switch are consumed by it.
	// Isolation between sibling cases is provided by `ReturnInfo.Clone`
	// in `checkConditionalBranches` (each case is a branch with its own
	// cloned jump-offset sets). Loop jump offsets (`continue`) are NOT
	// scoped here — they target the enclosing loop and must survive
	// past the switch (see `WithNewSwitchJumpTarget`).
	a.pushControl(ControlKindSwitch)
	savedMaybeJumpedSwitch := a.ReturnInfo.MaybeJumpedSwitch
	a.ReturnInfo.WithNewSwitchJumpTarget(f)
	if a.ReturnInfo.MaybeJumpedSwitch {
		a.ReturnInfo.clearDefiniteExits()
	}
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
