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

package analysis

// Analyzer describes an analysis function and its options
type Analyzer struct {
	Description string

	Run func(*Pass) interface{}

	// Requires is a set of analyzers that must run before this one.
	// This analyzer may inspect the outputs produced by each analyzer in Requires.
	// The graph over analyzers implied by Requires edges must be acyclic
	Requires []*Analyzer
}
