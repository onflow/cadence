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

package interpreter_test

import (
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/common_utils"
)

type noopReferenceTracker struct{}

func (_ noopReferenceTracker) ClearReferencedResourceKindedValues(_ atree.ValueID) {
	return
}

func (_ noopReferenceTracker) ReferencedResourceKindedValues(_ atree.ValueID) map[*interpreter.EphemeralReferenceValue]struct{} {
	return nil
}

func (_ noopReferenceTracker) MaybeTrackReferencedResourceKindedValue(_ *interpreter.EphemeralReferenceValue) {
	return
}

var _ interpreter.ReferenceTracker = noopReferenceTracker{}

func TestInterpretIDCapability(t *testing.T) {

	t.Parallel()

	const id = 99

	type handlers struct {
		borrow interpreter.CapabilityBorrowHandlerFunc
		check  interpreter.CapabilityCheckHandlerFunc
	}

	test := func(
		t *testing.T,
		code string,
		handlers handlers,
	) (common_utils.Invokable, error) {
		borrowType := &sema.ReferenceType{
			Type:          sema.StringType,
			Authorization: sema.UnauthorizedAccess,
		}

		borrowStaticType := interpreter.ConvertSemaToStaticType(nil, borrowType)

		value := stdlib.StandardLibraryValue{
			Type: &sema.CapabilityType{
				BorrowType: borrowType,
			},
			Value: interpreter.NewUnmeteredCapabilityValue(
				id,
				interpreter.AddressValue{0x42},
				borrowStaticType,
			),
			Name: "cap",
			Kind: common.DeclarationKindConstant,
		}

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(value)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, value)

		return parseCheckAndPrepareWithOptions(
			t,
			code,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
					CapabilityBorrowHandler: handlers.borrow,
					CapabilityCheckHandler:  handlers.check,
				},
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
				HandleCheckerError: nil,
			},
		)
	}

	t.Run("transfer", func(t *testing.T) {

		t.Parallel()

		_, err := test(t,
			`
              fun f(_ cap: Capability<&String>): Capability<&String>? {
                  return cap
              }

	          let capOpt: Capability<&String>? = f(cap)
            `,
			handlers{},
		)
		require.NoError(t, err)
	})

	t.Run("borrow", func(t *testing.T) {

		t.Parallel()

		mockReference := interpreter.NewUnmeteredEphemeralReferenceValue(
			noopReferenceTracker{},
			interpreter.UnauthorizedAccess,
			interpreter.NewUnmeteredStringValue("mock"),
			sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.StringType),
			interpreter.EmptyLocationRange,
		)

		inter, err := test(t,
			`
              fun test(): &String? {
                  return cap.borrow()
              }
            `,
			handlers{
				borrow: func(
					_ interpreter.BorrowCapabilityControllerContext,
					_ interpreter.LocationRange,
					address interpreter.AddressValue,
					capabilityID interpreter.UInt64Value,
					_ *sema.ReferenceType,
					_ *sema.ReferenceType,
				) interpreter.ReferenceValue {
					assert.Equal(t, interpreter.AddressValue{0x42}, address)
					assert.Equal(t, interpreter.UInt64Value(id), capabilityID)

					return mockReference
				},
			},
		)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredSomeValueNonCopying(mockReference), res)
	})

	t.Run("check", func(t *testing.T) {

		t.Parallel()

		var checked bool

		inter, err := test(t,
			`
              fun test(): Bool {
                  return cap.check()
              }
            `,
			handlers{
				check: func(
					_ interpreter.CheckCapabilityControllerContext,
					_ interpreter.LocationRange,
					address interpreter.AddressValue,
					capabilityID interpreter.UInt64Value,
					_ *sema.ReferenceType,
					_ *sema.ReferenceType,
				) interpreter.BoolValue {
					assert.Equal(t, interpreter.AddressValue{0x42}, address)
					assert.Equal(t, interpreter.UInt64Value(id), capabilityID)

					checked = true

					return interpreter.TrueValue
				},
			},
		)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.TrueValue, res)
		require.True(t, checked, "check handler was not called")
	})

	t.Run("id", func(t *testing.T) {

		t.Parallel()

		inter, err := test(t,
			`
              fun test(): UInt64 {
                  return cap.id
              }
            `,
			handlers{},
		)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.UInt64Value(id), res)
	})
}
