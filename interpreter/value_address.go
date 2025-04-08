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

package interpreter

import (
	"encoding/binary"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// AddressValue
type AddressValue common.Address

func NewAddressValueFromBytes(memoryGauge common.MemoryGauge, constructor func() []byte) AddressValue {
	common.UseMemory(memoryGauge, common.AddressValueMemoryUsage)
	return NewUnmeteredAddressValueFromBytes(constructor())
}

func NewUnmeteredAddressValueFromBytes(b []byte) AddressValue {
	result := AddressValue{}
	copy(result[common.AddressLength-len(b):], b)
	return result
}

// NewAddressValue constructs an address-value from a `common.Address`.
//
// NOTE:
// This method must only be used if the `address` value is already constructed,
// and/or already loaded onto memory. This is a convenient method for better performance.
// If the `address` needs to be constructed, the `NewAddressValueFromConstructor` must be used.
func NewAddressValue(
	memoryGauge common.MemoryGauge,
	address common.Address,
) AddressValue {
	common.UseMemory(memoryGauge, common.AddressValueMemoryUsage)
	return NewUnmeteredAddressValueFromBytes(address[:])
}

func NewAddressValueFromConstructor(
	memoryGauge common.MemoryGauge,
	addressConstructor func() common.Address,
) AddressValue {
	common.UseMemory(memoryGauge, common.AddressValueMemoryUsage)
	address := addressConstructor()
	return NewUnmeteredAddressValueFromBytes(address[:])
}

func ConvertAddress(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) AddressValue {
	if address, ok := value.(AddressValue); ok {
		return address
	}

	converter := func() (result common.Address) {
		uint64Value := ConvertUInt64(memoryGauge, value, locationRange)

		binary.BigEndian.PutUint64(
			result[:common.AddressLength],
			uint64(uint64Value),
		)

		return
	}

	return NewAddressValueFromConstructor(memoryGauge, converter)
}

var _ Value = AddressValue{}
var _ atree.Storable = AddressValue{}
var _ EquatableValue = AddressValue{}
var _ HashableValue = AddressValue{}
var _ MemberAccessibleValue = AddressValue{}

func (AddressValue) IsValue() {}

func (v AddressValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitAddressValue(interpreter, v)
}

func (AddressValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (AddressValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeAddress)
}

func (AddressValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v AddressValue) String() string {
	return format.Address(common.Address(v))
}

func (v AddressValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v AddressValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(context, common.AddressValueStringMemoryUsage)
	return v.String()
}

func (v AddressValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherAddress, ok := other.(AddressValue)
	if !ok {
		return false
	}
	return v == otherAddress
}

// HashInput returns a byte slice containing:
// - HashInputTypeAddress (1 byte)
// - address (8 bytes)
func (v AddressValue) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	length := 1 + len(v)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeAddress)
	copy(buffer[1:], v[:])
	return buffer
}

func (v AddressValue) Hex() string {
	return v.ToAddress().Hex()
}

func (v AddressValue) ToAddress() common.Address {
	return common.Address(v)
}

func (v AddressValue) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ToStringFunctionType,
			func(v AddressValue, invocation Invocation) Value {
				interpreter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				memoryUsage := common.NewStringMemoryUsage(
					safeMul(common.AddressLength, 2, locationRange),
				)

				return NewStringValue(
					interpreter,
					memoryUsage,
					func() string {
						return v.String()
					},
				)
			},
		)

	case sema.AddressTypeToBytesFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.AddressTypeToBytesFunctionType,
			func(v AddressValue, invocation Invocation) Value {
				interpreter := invocation.InvocationContext
				address := common.Address(v)
				return ByteSliceToByteArrayValue(interpreter, address[:])
			},
		)
	}

	return nil
}

func (AddressValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Addresses have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (AddressValue) SetMember(_ MemberAccessibleContext, _ LocationRange, _ string, _ Value) bool {
	// Addresses have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v AddressValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (AddressValue) IsStorable() bool {
	return true
}

func (v AddressValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (AddressValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (AddressValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v AddressValue) Transfer(
	transferContext ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		RemoveReferencedSlab(transferContext, storable)
	}
	return v
}

func (v AddressValue) Clone(_ *Interpreter) Value {
	return v
}

func (AddressValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v AddressValue) ByteSize() uint32 {
	return values.CBORTagSize + values.GetBytesCBORSize(v.ToAddress().Bytes())
}

func (v AddressValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (AddressValue) ChildStorables() []atree.Storable {
	return nil
}

func AddressFromBytes(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.InvocationContext

	bytes, err := ByteArrayValueToByteSlice(inter, argument, invocation.LocationRange)
	if err != nil {
		panic(err)
	}

	return NewAddressValue(invocation.InvocationContext, common.MustBytesToAddress(bytes))
}

func AddressFromString(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	addr, err := common.HexToAddressAssertPrefix(argument.Str)
	if err != nil {
		return Nil
	}

	inter := invocation.InvocationContext
	return NewSomeValueNonCopying(inter, NewAddressValue(inter, addr))
}
