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
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

type noopReferenceTracker struct{}

func (noopReferenceTracker) ClearReferencedResourceKindedValues(_ atree.ValueID) {
	// NO-OP
}

func (noopReferenceTracker) ReferencedResourceKindedValues(_ atree.ValueID) map[*interpreter.EphemeralReferenceValue]struct{} {
	// NO-OP
	return nil
}

func (noopReferenceTracker) MaybeTrackReferencedResourceKindedValue(_ *interpreter.EphemeralReferenceValue) {
	// NO-OP
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
	) (Invokable, error) {
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
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
					CapabilityBorrowHandler: handlers.borrow,
					CapabilityCheckHandler:  handlers.check,
				},
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
			sema.StringType,
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

func TestInterpretIDCapabilityUntyped(t *testing.T) {

	t.Parallel()

	const id = 99

	type handlers struct {
		borrow interpreter.CapabilityBorrowHandlerFunc
		check  interpreter.CapabilityCheckHandlerFunc
	}

	test := func(
		t *testing.T,
		code string,
		referencedType interpreter.StaticType,
		handlers handlers,
	) (Invokable, error) {

		// Capability value has static type `Capability`,
		// but can be borrowed as `auth(Mutate) &U`,
		// where U is the referenced type

		entitlements := orderedmap.New[sema.TypeIDOrderedSet](1)
		entitlements.Set(sema.MutateType.ID(), struct{}{})

		value := stdlib.StandardLibraryValue{
			// Static type is `Capability` without type argument
			Type: &sema.CapabilityType{},
			Value: interpreter.NewUnmeteredCapabilityValue(
				id,
				interpreter.AddressValue{0x42},
				// Referenced type is `auth(Mutate) &U`, where U is the referenced type
				&interpreter.ReferenceStaticType{
					Authorization: interpreter.EntitlementSetAuthorization{
						Entitlements: entitlements,
						SetKind:      sema.Conjunction,
					},
					ReferencedType: referencedType,
				},
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
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
					CapabilityBorrowHandler: handlers.borrow,
					CapabilityCheckHandler:  handlers.check,
				},
			},
		)
	}

	t.Run("transfer", func(t *testing.T) {

		t.Parallel()

		_, err := test(t,
			`
              fun f(_ cap: Capability): Capability? {
                  return cap
              }

	          let capOpt: Capability? = f(cap)
            `,
			interpreter.PrimitiveStaticTypeString,
			handlers{},
		)
		require.NoError(t, err)
	})

	t.Run("borrow, without entitlements, struct", func(t *testing.T) {

		t.Parallel()

		var mockReference *interpreter.EphemeralReferenceValue

		inter, err := test(t,
			`
              fun test(): &String? {
                  return cap.borrow<&String>()
              }
            `,
			interpreter.PrimitiveStaticTypeString,
			handlers{
				borrow: func(
					_ interpreter.BorrowCapabilityControllerContext,
					address interpreter.AddressValue,
					capabilityID interpreter.UInt64Value,
					wantedBorrowType *sema.ReferenceType,
					_ *sema.ReferenceType,
				) interpreter.ReferenceValue {

					assert.Equal(t, interpreter.AddressValue{0x42}, address)
					assert.Equal(t, interpreter.UInt64Value(id), capabilityID)

					mockReference = interpreter.NewUnmeteredEphemeralReferenceValue(
						noopReferenceTracker{},
						interpreter.ConvertSemaAccessToStaticAuthorization(nil, wantedBorrowType.Authorization),
						interpreter.NewUnmeteredStringValue("mock"),
						sema.StringType,
					)

					return mockReference
				},
			},
		)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredSomeValueNonCopying(mockReference), res)
	})

	t.Run("borrow, without entitlements, resource", func(t *testing.T) {

		t.Parallel()

		var mockReference *interpreter.EphemeralReferenceValue

		const rTypeQualifiedIdentifier = "R"
		rTypeID := TestLocation.TypeID(nil, rTypeQualifiedIdentifier)
		rStaticType := &interpreter.CompositeStaticType{
			Location:            TestLocation,
			QualifiedIdentifier: rTypeQualifiedIdentifier,
			TypeID:              rTypeID,
		}

		inter, err := test(t,
			`
              resource R {}

              fun test(): &R? {
                  return cap.borrow<&R>()
              }
            `,
			rStaticType,
			handlers{
				borrow: func(
					context interpreter.BorrowCapabilityControllerContext,
					address interpreter.AddressValue,
					capabilityID interpreter.UInt64Value,
					wantedBorrowType *sema.ReferenceType,
					capabilityBorrowType *sema.ReferenceType,
				) interpreter.ReferenceValue {

					assert.Equal(t, interpreter.AddressValue{0x42}, address)
					assert.Equal(t, interpreter.UInt64Value(id), capabilityID)

					rSemaType, err := context.GetCompositeType(TestLocation, rTypeQualifiedIdentifier, rTypeID)
					require.NoError(t, err)

					assert.True(t, capabilityBorrowType.Type.Equal(rSemaType))

					mockReference = interpreter.NewUnmeteredEphemeralReferenceValue(
						noopReferenceTracker{},
						interpreter.ConvertSemaAccessToStaticAuthorization(nil, wantedBorrowType.Authorization),
						interpreter.NewCompositeValue(
							context,
							TestLocation,
							rTypeQualifiedIdentifier,
							common.CompositeKindResource,
							nil,
							common.ZeroAddress,
						),
						rSemaType,
					)

					return mockReference
				},
			},
		)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredSomeValueNonCopying(mockReference), res)
	})

	t.Run("borrow, with entitlements, struct", func(t *testing.T) {

		t.Parallel()

		var mockReference *interpreter.EphemeralReferenceValue

		inter, err := test(t,
			`
              fun test(): auth(Mutate) &String? {
                  return cap.borrow<auth(Mutate) &String>()
              }
            `,
			interpreter.PrimitiveStaticTypeString,
			handlers{
				borrow: func(
					_ interpreter.BorrowCapabilityControllerContext,
					address interpreter.AddressValue,
					capabilityID interpreter.UInt64Value,
					wantedBorrowType *sema.ReferenceType,
					_ *sema.ReferenceType,
				) interpreter.ReferenceValue {

					assert.Equal(t, interpreter.AddressValue{0x42}, address)
					assert.Equal(t, interpreter.UInt64Value(id), capabilityID)

					mockReference = interpreter.NewUnmeteredEphemeralReferenceValue(
						noopReferenceTracker{},
						interpreter.ConvertSemaAccessToStaticAuthorization(nil, wantedBorrowType.Authorization),
						interpreter.NewUnmeteredStringValue("mock"),
						sema.StringType,
					)

					return mockReference
				},
			},
		)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredSomeValueNonCopying(mockReference), res)
	})

	t.Run("borrow, with entitlements, resource", func(t *testing.T) {

		t.Parallel()

		var mockReference *interpreter.EphemeralReferenceValue

		const rTypeQualifiedIdentifier = "R"
		rTypeID := TestLocation.TypeID(nil, rTypeQualifiedIdentifier)
		rStaticType := &interpreter.CompositeStaticType{
			Location:            TestLocation,
			QualifiedIdentifier: rTypeQualifiedIdentifier,
			TypeID:              rTypeID,
		}

		inter, err := test(t,
			`
              resource R {}

              fun test(): auth(Mutate) &R? {
                  return cap.borrow<auth(Mutate) &R>()
              }
            `,
			rStaticType,
			handlers{
				borrow: func(
					context interpreter.BorrowCapabilityControllerContext,
					address interpreter.AddressValue,
					capabilityID interpreter.UInt64Value,
					wantedBorrowType *sema.ReferenceType,
					capabilityBorrowType *sema.ReferenceType,
				) interpreter.ReferenceValue {

					assert.Equal(t, interpreter.AddressValue{0x42}, address)
					assert.Equal(t, interpreter.UInt64Value(id), capabilityID)

					rSemaType, err := context.GetCompositeType(TestLocation, rTypeQualifiedIdentifier, rTypeID)
					require.NoError(t, err)

					assert.True(t, wantedBorrowType.Type.Equal(rSemaType))

					mockReference = interpreter.NewUnmeteredEphemeralReferenceValue(
						noopReferenceTracker{},
						interpreter.ConvertSemaAccessToStaticAuthorization(nil, wantedBorrowType.Authorization),
						interpreter.NewCompositeValue(
							context,
							TestLocation,
							rTypeQualifiedIdentifier,
							common.CompositeKindResource,
							nil,
							common.ZeroAddress,
						),
						rSemaType,
					)

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
                  return cap.check<&String>()
              }
            `,
			interpreter.PrimitiveStaticTypeString,
			handlers{
				check: func(
					_ interpreter.CheckCapabilityControllerContext,
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
			interpreter.PrimitiveStaticTypeString,
			handlers{},
		)
		require.NoError(t, err)

		res, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.UInt64Value(id), res)
	})
}
