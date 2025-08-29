//go:build !cadence_tracing

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

package runtime_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestRuntimeTracingDisabled(t *testing.T) {

	t.Parallel()

	tx := []byte(`
      transaction {
        prepare() {
          let dict: {String: Int} = {}
          let array: [Int] = [] 
        }
      }
    `)

	runtime := NewTestRuntime()

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnRecordTrace: func(_ string, _ time.Duration, _ []attribute.KeyValue) {
			t.Fatal("tracing should not be called when disabled")
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
			UseVM:     *compile,
		},
	)
	require.NoError(t, err)
}
