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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func newRoundingModeArgument(rawValue uint8) cadence.Value {
	return cadence.NewEnum([]cadence.Value{
		cadence.UInt8(rawValue),
	}).WithType(cadence.NewEnumType(
		nil,
		sema.RoundingModeTypeName,
		cadence.UInt8Type,
		[]cadence.Field{
			{
				Identifier: sema.EnumRawValueFieldName,
				Type:       cadence.UInt8Type,
			},
		},
		nil,
	))
}

func TestRuntimeRoundingModeExport(t *testing.T) {

	t.Parallel()

	runtime := NewTestRuntime()
	runtimeInterface := &TestRuntimeInterface{}
	nextScriptLocation := NewScriptLocationGenerator()

	testRoundingMode := func(mode sema.NativeEnumCase) {
		script := fmt.Sprintf(`
              access(all) fun main(): RoundingMode {
                  return RoundingMode.%s
              }
            `,
			mode.Name(),
		)

		value, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
				UseVM:     *compile,
			},
		)

		require.NoError(t, err)

		require.IsType(t, cadence.Enum{}, value)
		enumValue := value.(cadence.Enum)

		fields := cadence.FieldsMappedByName(enumValue)
		require.Len(t, fields, 1)
		assert.Equal(t,
			cadence.NewUInt8(mode.RawValue()),
			fields[sema.EnumRawValueFieldName],
		)
	}

	for _, mode := range sema.RoundingModes {
		testRoundingMode(mode)
	}
}

func TestRuntimeRoundingModeImport(t *testing.T) {

	t.Parallel()

	runtime := NewTestRuntime()
	runtimeInterface := &TestRuntimeInterface{
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}
	nextScriptLocation := NewScriptLocationGenerator()

	const script = `
      access(all) fun main(mode: RoundingMode): UInt8 {
          return mode.rawValue
      }
    `

	testRoundingMode := func(mode sema.NativeEnumCase) {

		value, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					newRoundingModeArgument(mode.RawValue()),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
				UseVM:     *compile,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, cadence.UInt8(mode.RawValue()), value)
	}

	for _, mode := range sema.RoundingModes {
		testRoundingMode(mode)
	}
}

func TestRuntimeRoundingModeImportInvalid(t *testing.T) {

	t.Parallel()

	runtime := NewTestRuntime()
	runtimeInterface := &TestRuntimeInterface{
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}
	nextScriptLocation := NewScriptLocationGenerator()

	const script = `
      access(all) fun main(mode: RoundingMode): UInt8 {
          return mode.rawValue
      }
    `

	_, err := runtime.ExecuteScript(
		Script{
			Source: []byte(script),
			Arguments: encodeArgs([]cadence.Value{
				newRoundingModeArgument(99), // invalid raw value
			}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
			UseVM:     *compile,
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown RoundingMode")
}

func TestRuntimeFix64ConversionWithRoundingModeArgument(t *testing.T) {

	t.Parallel()

	runtime := NewTestRuntime()
	runtimeInterface := &TestRuntimeInterface{
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}
	nextScriptLocation := NewScriptLocationGenerator()

	t.Run("Fix128 to Fix64 with rounding", func(t *testing.T) {
		t.Parallel()

		const script = `
          access(all) fun main(mode: RoundingMode): Fix64 {
              let x: Fix128 = 1.000000005000000000000000
              return Fix64(x, rounding: mode)
          }
        `

		// towardZero should truncate
		value, err := runtime.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: encodeArgs([]cadence.Value{newRoundingModeArgument(0)}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
				UseVM:     *compile,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, cadence.Fix64(100000000), value) // 1.00000000
	})

	t.Run("UFix128 to UFix64 with rounding", func(t *testing.T) {
		t.Parallel()

		const script = `
          access(all) fun main(mode: RoundingMode): UFix64 {
              let x: UFix128 = 1.000000005000000000000000
              return UFix64(x, rounding: mode)
          }
        `

		// awayFromZero should round up
		value, err := runtime.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: encodeArgs([]cadence.Value{newRoundingModeArgument(1)}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
				UseVM:     *compile,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, cadence.UFix64(100000001), value) // 1.00000001
	})
}
