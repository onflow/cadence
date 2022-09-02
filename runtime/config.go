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

package runtime

import (
	"github.com/onflow/cadence/runtime/interpreter"
)

// Config is a constant/read-only configuration of an environment.
//
type Config struct {
	Debugger *interpreter.Debugger
	// AtreeValidationEnabled configures if atree validation is enabled.
	AtreeValidationEnabled bool
	// TracingEnabled configures if tracing is enabled.
	TracingEnabled bool
	// ResourceOwnerChangeCallbackEnabled configures if the resource owner change callback is enabled.
	ResourceOwnerChangeHandlerEnabled bool
	// CoverageReportingEnabled configures if coverage reporting is enabled.
	CoverageReportingEnabled bool
	// StackDepthLimit specifies the maximum depth for call stacks.
	StackDepthLimit uint64
}
