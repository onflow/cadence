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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

var contractFunctionInvocationTestAddress = common.MustBytesToAddress([]byte{0x1})

var contractFunctionInvocationTestLocation = common.AddressLocation{
	Address: contractFunctionInvocationTestAddress,
	Name:    "Test",
}

const contractFunctionInvocationTestContract = `
    access(all) contract Test {

        access(all) struct Point {
            access(all) let x: UInt8

            init(x: UInt8) {
                self.x = x
            }
        }

        access(all) enum Tier: UInt8 {
            access(all) case basic
            access(all) case premium
        }

        access(all) fun takePoint(_ p: Point): UInt8 {
            return p.x
        }

        access(all) fun takeInt(_ n: Int): Int {
            return n
        }

        access(all) fun feeFor(_ tier: Tier): UInt8 {
            switch tier {
                case Tier.basic:
                    return 10
                case Tier.premium:
                    return 20
            }
            return 0
        }
    }
`

func invokeTestContractFunction(
	t *testing.T,
	functionName string,
	argument cadence.Value,
	argumentType sema.Type,
) error {
	address := contractFunctionInvocationTestAddress

	rt := NewTestRuntime()

	var accountCode []byte

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) ([]byte, error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnGetCode: func(_ Location) ([]byte, error) {
			return accountCode, nil
		},
		OnEmitEvent: func(_ cadence.Event) error {
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
				"Test",
				[]byte(contractFunctionInvocationTestContract),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
			UseVM:     *compile,
		},
	)
	require.NoError(t, err)

	_, err = rt.InvokeContractFunction(
		contractFunctionInvocationTestLocation,
		functionName,
		[]cadence.Value{argument},
		[]sema.Type{argumentType},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
			UseVM:     *compile,
		},
	)
	return err
}

func TestRuntimeInvokeContractFunctionEnumValidation(t *testing.T) {

	t.Parallel()

	makeTier := func(rawValue cadence.Value, rawType cadence.Type) cadence.Enum {
		return cadence.NewEnum([]cadence.Value{rawValue}).WithType(
			cadence.NewEnumType(
				contractFunctionInvocationTestLocation,
				"Test.Tier",
				rawType,
				[]cadence.Field{
					{Identifier: sema.EnumRawValueFieldName, Type: rawType},
				},
				nil,
			),
		)
	}

	tests := []struct {
		name          string
		arg           cadence.Value
		errorContains string
	}{
		{
			name: "valid case",
			arg:  makeTier(cadence.NewUInt8(1), cadence.UInt8Type),
		},
		{
			// Rejected by the enum case-membership check in importCompositeValue.
			name:          "forged out-of-range case",
			arg:           makeTier(cadence.NewUInt8(99), cadence.UInt8Type),
			errorContains: "is not a valid enum case",
		},
		{
			// Rejected by the exact-raw-type check in the enum importer.
			name:          "wrong raw type",
			arg:           makeTier(cadence.NewInt(1), cadence.IntType),
			errorContains: "raw value has type Int, expected UInt8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := invokeTestContractFunction(t, "feeFor", tc.arg, sema.AnyStructType)

			if tc.errorContains == "" {
				require.NoError(t, err)
			} else {
				RequireError(t, err)
				assert.ErrorContains(t, err, tc.errorContains)
			}
		})
	}
}

func TestRuntimeInvokeContractFunctionArgumentValidation(t *testing.T) {

	t.Parallel()

	makePoint := func(x cadence.Value, xType cadence.Type) cadence.Struct {
		return cadence.NewStruct([]cadence.Value{x}).WithType(
			cadence.NewStructType(
				contractFunctionInvocationTestLocation,
				"Test.Point",
				[]cadence.Field{
					{Identifier: "x", Type: xType},
				},
				nil,
			),
		)
	}

	tests := []struct {
		name          string
		functionName  string
		arg           cadence.Value
		argType       sema.Type
		errorContains string
	}{
		{
			name:         "valid struct",
			functionName: "takePoint",
			arg:          makePoint(cadence.NewUInt8(5), cadence.UInt8Type),
			argType:      sema.AnyStructType,
		},
		{
			// The struct's field `x` is declared as `UInt8`, but an `Int` is
			// injected. The malformed value is rejected by the
			// ConformsToStaticType check in validateImportedArgument.
			name:          "malformed struct field type",
			functionName:  "takePoint",
			arg:           makePoint(cadence.NewInt(5), cadence.IntType),
			argType:       sema.AnyStructType,
			errorContains: "does not conform to expected type",
		},
		{
			name:         "valid int",
			functionName: "takeInt",
			arg:          cadence.NewInt(5),
			argType:      sema.IntType,
		},
		{
			// A `String` value is passed where an `Int` is expected.
			// Rejected by the subtype check in validateImportedArgument.
			name:          "wrong argument type (String for Int)",
			functionName:  "takeInt",
			arg:           cadence.String("not an int"),
			argType:       sema.IntType,
			errorContains: "expected value of type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := invokeTestContractFunction(t, tc.functionName, tc.arg, tc.argType)

			if tc.errorContains == "" {
				require.NoError(t, err)
			} else {
				RequireError(t, err)
				assert.ErrorContains(t, err, tc.errorContains)
			}
		})
	}
}
