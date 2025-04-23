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

package interpreter

import (
	"sync/atomic"

	"github.com/bits-and-blooms/bitset"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

type Stop struct {
	Interpreter *Interpreter
	Statement   ast.Statement
}

type Debugger struct {
	stops          chan Stop
	continues      chan struct{}
	breakpoints    map[common.Location]*bitset.BitSet
	pauseRequested uint32
}

func NewDebugger() *Debugger {
	return &Debugger{
		stops:       make(chan Stop),
		continues:   make(chan struct{}),
		breakpoints: map[common.Location]*bitset.BitSet{},
	}
}

func (d *Debugger) Stops() <-chan Stop {
	return d.stops
}

func (d *Debugger) AddBreakpoint(location common.Location, line uint) {
	breakpoints, ok := d.breakpoints[location]
	if !ok {
		breakpoints = bitset.New(1024)
		d.breakpoints[location] = breakpoints
	}
	breakpoints.Set(line)
}

func (d *Debugger) RemoveBreakpoint(location common.Location, line uint) {
	breakpoints, ok := d.breakpoints[location]
	if !ok {
		return
	}
	breakpoints.Clear(line)
}

func (d *Debugger) ClearBreakpoints() {
	for location := range d.breakpoints { //nolint:maprange
		delete(d.breakpoints, location)
	}
}

func (d *Debugger) ClearBreakpointsForLocation(location common.Location) {
	delete(d.breakpoints, location)
}

func (d *Debugger) onStatement(interpreter *Interpreter, statement ast.Statement) {
	if !atomic.CompareAndSwapUint32(&d.pauseRequested, 1, 0) {
		breakpoints, ok := d.breakpoints[interpreter.Location]
		if !ok {
			return
		}

		startPosition := statement.StartPosition()
		if !breakpoints.Test(uint(startPosition.Line)) {
			return
		}
	}

	d.stops <- Stop{
		Interpreter: interpreter,
		Statement:   statement,
	}

	<-d.continues
}

func (d *Debugger) RequestPause() {
	atomic.StoreUint32(&d.pauseRequested, 1)
}

func (d *Debugger) Continue() {
	d.continues <- struct{}{}
}

func (d *Debugger) Pause() Stop {
	d.RequestPause()
	return <-d.Stops()
}

func (d *Debugger) Next() Stop {
	d.RequestPause()
	d.Continue()
	return <-d.Stops()
}

func (d *Debugger) CurrentActivation(interpreter *Interpreter) *VariableActivation {
	return interpreter.activations.Current()
}
