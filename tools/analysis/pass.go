/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

// Pass provides information to the Analyzer.Run function,
// which applies a specific analyzer to a single location.
type Pass struct {
	Program *Program

	// Report reports a Diagnostic, a finding about a specific location
	// in the analyzed source code such as a potential mistake.
	// It may be called by the Analyzer.Run function.
	Report func(Diagnostic)

	// ResultOf provides the inputs to this analysis pass,
	// which are the corresponding results of its prerequisite analyzers.
	// The map keys are the elements of Analyzer.Requires.
	ResultOf map[*Analyzer]interface{}
}
