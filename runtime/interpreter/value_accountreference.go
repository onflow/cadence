/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// AccountReferenceValue

type AccountReferenceValue struct {
	BorrowedType sema.Type
	_authAccount Value
	Path         PathValue
	Address      common.Address
}

var _ Value = &AccountReferenceValue{}
var _ EquatableValue = &AccountReferenceValue{}
var _ ValueIndexableValue = &AccountReferenceValue{}
var _ MemberAccessibleValue = &AccountReferenceValue{}
var _ ReferenceValue = &AccountReferenceValue{}

func NewUnmeteredAccountReferenceValue(
	address common.Address,
	path PathValue,
	borrowedType sema.Type,
) *AccountReferenceValue {
	return &AccountReferenceValue{
		Address:      address,
		Path:         path,
		BorrowedType: borrowedType,
	}
}

func NewAccountReferenceValue(
	memoryGauge common.MemoryGauge,
	address common.Address,
	path PathValue,
	borrowedType sema.Type,
) *AccountReferenceValue {
	common.UseMemory(memoryGauge, common.AccountReferenceValueMemoryUsage)
	return NewUnmeteredAccountReferenceValue(
		address,
		path,
		borrowedType,
	)
}

func (*AccountReferenceValue) isValue() {}

func (*AccountReferenceValue) isReference() {}

func (v *AccountReferenceValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitAccountReferenceValue(interpreter, v)
}

func (*AccountReferenceValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (*AccountReferenceValue) String() string {
	return format.AccountReference
}

func (v *AccountReferenceValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *AccountReferenceValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(memoryGauge, common.AccountReferenceValueStringMemoryUsage)
	return v.String()
}

func (v *AccountReferenceValue) StaticType(inter *Interpreter) StaticType {
	return NewReferenceStaticType(
		inter,
		false,
		ConvertSemaToStaticType(inter, v.BorrowedType),
		PrimitiveStaticTypeAuthAccount,
	)
}

func (*AccountReferenceValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v *AccountReferenceValue) checkLink(interpreter *Interpreter, locationRange LocationRange) {
	address := v.Address
	domain := v.Path.Domain.Identifier()
	identifier := v.Path.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	referenced := interpreter.ReadStored(address, domain, storageMapKey)
	if referenced == nil {
		panic(DereferenceError{
			Cause:         "no value is stored at this path",
			LocationRange: locationRange,
		})
	}

	if v.BorrowedType != nil {
		if !interpreter.IsSubTypeOfSemaType(
			PrimitiveStaticTypeAuthAccount,
			v.BorrowedType,
		) {
			panic(ForceCastTypeMismatchError{
				ExpectedType:  v.BorrowedType,
				ActualType:    sema.AuthAccountType,
				LocationRange: locationRange,
			})
		}
	}
}

func (v *AccountReferenceValue) GetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	v.checkLink(interpreter, locationRange)
	self := v.authAccount(interpreter)
	return interpreter.getMember(self, locationRange, name)
}

func (v *AccountReferenceValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	v.checkLink(interpreter, locationRange)
	self := v.authAccount(interpreter)
	return self.(MemberAccessibleValue).RemoveMember(interpreter, locationRange, name)
}

func (v *AccountReferenceValue) SetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
	value Value,
) bool {
	v.checkLink(interpreter, locationRange)
	self := v.authAccount(interpreter)
	return interpreter.setMember(self, locationRange, name, value)
}

func (v *AccountReferenceValue) GetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	v.checkLink(interpreter, locationRange)
	self := v.authAccount(interpreter)
	return self.(ValueIndexableValue).
		GetKey(interpreter, locationRange, key)
}

func (v *AccountReferenceValue) SetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	v.checkLink(interpreter, locationRange)
	self := v.authAccount(interpreter)
	self.(ValueIndexableValue).
		SetKey(interpreter, locationRange, key, value)
}

func (v *AccountReferenceValue) InsertKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	v.checkLink(interpreter, locationRange)
	self := v.authAccount(interpreter)
	self.(ValueIndexableValue).
		InsertKey(interpreter, locationRange, key, value)
}

func (v *AccountReferenceValue) RemoveKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	v.checkLink(interpreter, locationRange)
	self := v.authAccount(interpreter)
	return self.(ValueIndexableValue).
		RemoveKey(interpreter, locationRange, key)
}

func (v *AccountReferenceValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherReference, ok := other.(*AccountReferenceValue)
	if !ok ||
		v.Address != otherReference.Address ||
		v.Path != otherReference.Path {

		return false
	}

	if v.BorrowedType == nil {
		return otherReference.BorrowedType == nil
	} else {
		return v.BorrowedType.Equal(otherReference.BorrowedType)
	}
}

func (v *AccountReferenceValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	if !interpreter.IsSubTypeOfSemaType(
		PrimitiveStaticTypeAuthAccount,
		v.BorrowedType,
	) {
		return false
	}

	self := v.authAccount(interpreter)

	return self.ConformsToStaticType(
		interpreter,
		locationRange,
		results,
	)
}

func (*AccountReferenceValue) IsStorable() bool {
	return false
}

func (v *AccountReferenceValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: v}, nil
}

func (*AccountReferenceValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*AccountReferenceValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *AccountReferenceValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *AccountReferenceValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredAccountReferenceValue(
		v.Address,
		v.Path,
		v.BorrowedType,
	)
}

func (*AccountReferenceValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v *AccountReferenceValue) authAccount(interpreter *Interpreter) Value {
	if v._authAccount == nil {
		v._authAccount = interpreter.SharedState.Config.AuthAccountHandler(AddressValue(v.Address))
	}
	return v._authAccount
}
