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

package vm

import (
	"fmt"
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/interpreter"
)

type StorageReferenceValue struct {
	Authorized           bool
	TargetStorageAddress common.Address
	TargetPath           PathValue
	BorrowedType         interpreter.StaticType
	storage              interpreter.Storage
}

func NewStorageReferenceValue(
	storage interpreter.Storage,
	authorized bool,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType interpreter.StaticType,
) *StorageReferenceValue {
	return &StorageReferenceValue{
		Authorized:           authorized,
		TargetStorageAddress: targetStorageAddress,
		TargetPath:           targetPath,
		BorrowedType:         borrowedType,
		storage:              storage,
	}
}

var _ Value = &StorageReferenceValue{}

func (*StorageReferenceValue) isValue() {}

func (v *StorageReferenceValue) StaticType(gauge common.MemoryGauge) StaticType {
	referencedValue, err := v.dereference(gauge)
	if err != nil {
		panic(err)
	}

	return interpreter.NewReferenceStaticType(
		gauge,
		v.Authorized,
		v.BorrowedType,
		(*referencedValue).StaticType(gauge),
	)
}

func (v *StorageReferenceValue) dereference(gauge common.MemoryGauge) (*Value, error) {
	address := v.TargetStorageAddress
	domain := v.TargetPath.Domain.Identifier()
	identifier := v.TargetPath.Identifier

	accountStorage := v.storage.GetStorageMap(address, domain, false)
	if accountStorage == nil {
		return nil, nil
	}

	referenced := accountStorage.ReadValue(gauge, identifier)
	vmReferencedValue := InterpreterValueToVMValue(v.storage, referenced)

	if v.BorrowedType != nil {
		staticType := vmReferencedValue.StaticType(gauge)

		if !IsSubType(staticType, v.BorrowedType) {
			panic(fmt.Errorf("type mismatch: expected %s, found %s", v.BorrowedType, staticType))
			//semaType := interpreter.MustConvertStaticToSemaType(staticType)
			//
			//return nil, ForceCastTypeMismatchError{
			//	ExpectedType:  v.BorrowedType,
			//	ActualType:    semaType,
			//	LocationRange: locationRange,
			//}
		}
	}

	return &vmReferencedValue, nil
}

func (v *StorageReferenceValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v *StorageReferenceValue) String() string {
	return format.StorageReference
}
