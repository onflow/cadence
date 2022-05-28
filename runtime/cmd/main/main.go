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

package main

import (
	"os"
	"os/signal"

	"github.com/onflow/cadence/runtime/cmd/execute"
	"github.com/onflow/cadence/runtime/interpreter"
)

func main() {
	if len(os.Args) > 1 {
		// TODO: also make the REPL support the interactive debugger

		signals := make(chan os.Signal, 1)

		signal.Notify(signals, os.Interrupt)

		debugger := interpreter.NewDebugger()

		go func() {
			for range signals {
				stop := debugger.Pause()
				execute.NewInteractiveDebugger(debugger, stop).Run()
				debugger.Continue()
			}
		}()

		execute.Execute(os.Args[1:], debugger)
	} else {
		execute.RunREPL()
	}
}
