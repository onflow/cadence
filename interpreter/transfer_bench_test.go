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

	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

func BenchmarkByteArrayTransfer(b *testing.B) {
	const elementCount = 32

	newOwner := common.Address{0x1}

	inter := newTestInterpreter(b)

	elements := make([]Value, elementCount)
	for i := range elements {
		elements[i] = NewUnmeteredUInt8Value(uint8(i))
	}

	array := NewArrayValue(
		inter,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeUInt8,
		},
		common.ZeroAddress,
		elements...,
	)

	var transferred Value

	for b.Loop() {
		transferred = array.Transfer(
			inter,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			true, // Array value is standalone.
		)
	}

	_ = transferred
}

func BenchmarkEMVAddressTransfer(b *testing.B) {
	const (
		EVMAddressLength                  = 20
		EVMAddressTypeQualifiedIdentifier = "EVM.EVMAddress"
		EVMAddressTypeBytesFieldName      = "bytes"
	)
	var (
		EVMAddressBytesType       = sema.NewConstantSizedType(nil, sema.UInt8Type, EVMAddressLength)
		EVMAddressBytesStaticType = ConvertSemaArrayTypeToStaticArrayType(nil, EVMAddressBytesType)
	)

	newOwner := common.Address{0x1}

	inter := newTestInterpreter(b)

	elements := make([]Value, EVMAddressLength)
	for i := range elements {
		elements[i] = NewUnmeteredUInt8Value(uint8(i))
	}

	evmAddress := NewCompositeValue(
		inter,
		common.StringLocation("test"),
		EVMAddressTypeQualifiedIdentifier,
		common.CompositeKindStructure,
		[]CompositeField{
			{
				Name: EVMAddressTypeBytesFieldName,
				Value: NewArrayValue(
					inter,
					EVMAddressBytesStaticType,
					common.ZeroAddress,
					elements...,
				),
			},
		},
		common.ZeroAddress,
	)

	var transferred Value

	for b.Loop() {
		transferred = evmAddress.Transfer(
			inter,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			true, // EMVAddress is standalone.
		)
	}

	_ = transferred
}

func BenchmarkEnumTransfer(b *testing.B) {
	newOwner := common.Address{0x1}

	inter := newTestInterpreter(b)

	fields := []CompositeField{
		{
			Name:  "rawValue",
			Value: NewUnmeteredUInt8Value(42),
		},
	}

	enumValue := NewCompositeValue(
		inter,
		common.StringLocation("test"),
		"Priority",
		common.CompositeKindEnum,
		fields,
		common.ZeroAddress,
	)

	var transferred Value

	for b.Loop() {
		transferred = enumValue.Transfer(
			inter,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			true, // Enum value is standalone.
		)
	}

	_ = transferred
}
