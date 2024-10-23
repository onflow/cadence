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

package runtime_utils

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
)

type TestInterpreterRuntime struct {
	runtime.Runtime
}

var _ runtime.Runtime = TestInterpreterRuntime{}

func NewTestInterpreterRuntimeWithConfig(config runtime.Config) TestInterpreterRuntime {
	return TestInterpreterRuntime{
		Runtime: runtime.NewInterpreterRuntime(config),
	}
}

var DefaultTestInterpreterConfig = runtime.Config{
	AtreeValidationEnabled: true,
}

func NewTestInterpreterRuntime() TestInterpreterRuntime {
	return NewTestInterpreterRuntimeWithConfig(DefaultTestInterpreterConfig)
}

func (r TestInterpreterRuntime) ExecuteTransaction(script runtime.Script, context runtime.Context) error {
	i := context.Interface.(*TestRuntimeInterface)
	i.onTransactionExecutionStart()
	return r.Runtime.ExecuteTransaction(script, context)
}

func (r TestInterpreterRuntime) ExecuteScript(script runtime.Script, context runtime.Context) (cadence.Value, error) {
	i := context.Interface.(*TestRuntimeInterface)
	i.onScriptExecutionStart()
	value, err := r.Runtime.ExecuteScript(script, context)
	// If there was a return value, let's also ensure it can be encoded
	// TODO: also test CCF
	if value != nil && err == nil {
		_ = json.MustEncode(value)
	}
	return value, err
}
