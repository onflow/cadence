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

package analyzers

import (
	"fmt"
	"regexp"

	"github.com/onflow/cadence/tools/analysis"
)

var Analyzers = map[string]*analysis.Analyzer{}

var analyzerNamePattern = regexp.MustCompile(`\w+`)

func registerAnalyzer(name string, analyzer *analysis.Analyzer) {
	if _, ok := Analyzers[name]; ok {
		panic(fmt.Errorf("analyzer already exists: %s", name))
	}

	if !analyzerNamePattern.MatchString(name) {
		panic(fmt.Errorf("invalid analyzer name: %s", name))

	}

	Analyzers[name] = analyzer
}
