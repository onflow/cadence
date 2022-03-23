/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/ast"
)

type Stop struct {
	Interpreter *Interpreter
	Statement   ast.Statement
}

type Debugger struct {
	pauseRequested uint32
	stops          chan Stop
	continues      chan struct{}
}

func NewDebugger() *Debugger {
	return &Debugger{
		stops:     make(chan Stop),
		continues: make(chan struct{}),
	}
}

func (d *Debugger) Stops() <-chan Stop {
	return d.stops
}

func (d *Debugger) onStatement(interpreter *Interpreter, statement ast.Statement) {
	if !d.PauseRequested() {
		return
	}

	d.stops <- Stop{
		Interpreter: interpreter,
		Statement:   statement,
	}

	d.resetPauseRequest()

	<-d.continues
}

func (d *Debugger) PauseRequested() bool {
	return atomic.LoadUint32(&d.pauseRequested) == 1
}

func (d *Debugger) resetPauseRequest() {
	atomic.StoreUint32(&d.pauseRequested, 0)
}

func (d *Debugger) RequestPause() {
	atomic.StoreUint32(&d.pauseRequested, 1)
}

func (d *Debugger) Continue() bool {
	select {
	case d.continues <- struct{}{}:
		return true
	default:
		return false
	}
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
