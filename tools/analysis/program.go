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

package analysis

import (
	"sync"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

type Program struct {
	Location    common.Location
	Code        string
	Program     *ast.Program
	Elaboration *sema.Elaboration
}

// Run runs the given DAG of analyzers in parallel
//
func (program *Program) Run(analyzers []*Analyzer, report func(Diagnostic)) {

	type action struct {
		once   sync.Once
		result interface{}
	}

	actions := make(map[*Analyzer]*action)

	var registerAnalyzer func(a *Analyzer)
	registerAnalyzer = func(a *Analyzer) {
		act, ok := actions[a]
		if ok {
			return
		}
		act = new(action)
		for _, req := range a.Requires {
			registerAnalyzer(req)
		}
		actions[a] = act
	}

	for _, a := range analyzers {
		registerAnalyzer(a)
	}

	var exec func(a *Analyzer) *action
	var execAll func(analyzers []*Analyzer)
	exec = func(a *Analyzer) *action {
		act := actions[a]
		act.once.Do(func() {
			// Prefetch dependencies in parallel
			execAll(a.Requires)

			// The inputs to this analysis are the
			// results of its prerequisites.
			inputs := make(map[*Analyzer]interface{})
			for _, req := range a.Requires {
				requirementAction := exec(req)
				inputs[req] = requirementAction.result
			}

			pass := &Pass{
				Program:  program,
				Report:   report,
				ResultOf: inputs,
			}

			act.result = a.Run(pass)
		})
		return act
	}
	execAll = func(analyzers []*Analyzer) {
		var wg sync.WaitGroup
		for _, a := range analyzers {
			wg.Add(1)
			go func(a *Analyzer) {
				_ = exec(a)
				wg.Done()
			}(a)
		}
		wg.Wait()
	}

	execAll(analyzers)
}
