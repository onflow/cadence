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

package vm

import (
	"fmt"
	"github.com/onflow/atree"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/interpreter"
)

type StorageReferenceValue struct {
	Authorization        interpreter.Authorization
	TargetStorageAddress common.Address
	TargetPath           PathValue
	BorrowedType         interpreter.StaticType
}

var _ Value = &StorageReferenceValue{}
var _ MemberAccessibleValue = &StorageReferenceValue{}
var _ ReferenceValue = &StorageReferenceValue{}

func NewStorageReferenceValue(
	authorization interpreter.Authorization,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType interpreter.StaticType,
) *StorageReferenceValue {
	return &StorageReferenceValue{
		Authorization:        authorization,
		TargetStorageAddress: targetStorageAddress,
		TargetPath:           targetPath,
		BorrowedType:         borrowedType,
	}
}

func (*StorageReferenceValue) isValue() {}

func (v *StorageReferenceValue) isReference() {}

func (v *StorageReferenceValue) ReferencedValue(config *Config, errorOnFailedDereference bool) *Value {
	referenced, err := v.dereference(config)
	if err != nil && errorOnFailedDereference {
		panic(err)
	}

	return referenced
}

func (v *StorageReferenceValue) BorrowType() interpreter.StaticType {
	return v.BorrowedType
}

func (v *StorageReferenceValue) StaticType(config *Config) StaticType {
	referencedValue, err := v.dereference(config)
	if err != nil {
		panic(err)
	}

	memoryGauge := config.MemoryGauge

	return interpreter.NewReferenceStaticType(
		memoryGauge,
		v.Authorization,
		(*referencedValue).StaticType(config),
	)
}

func (v *StorageReferenceValue) dereference(config *Config) (*Value, error) {
	memoryGauge := config.MemoryGauge
	address := v.TargetStorageAddress
	domain := v.TargetPath.Domain.Identifier()
	identifier := v.TargetPath.Identifier

	vmReferencedValue := ReadStored(memoryGauge, config.Storage, address, domain, identifier)
	if vmReferencedValue == nil {
		return nil, nil
	}

	if _, isReference := vmReferencedValue.(ReferenceValue); isReference {
		panic(interpreter.NestedReferenceError{})
	}

	if v.BorrowedType != nil {
		staticType := vmReferencedValue.StaticType(config)

		if !IsSubType(config, staticType, v.BorrowedType) {
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

func (v *StorageReferenceValue) GetMember(config *Config, name string) Value {
	referencedValue, err := v.dereference(config)
	if err != nil {
		panic(err)
	}

	memberAccessibleValue := (*referencedValue).(MemberAccessibleValue)
	return memberAccessibleValue.GetMember(config, name)
}

func (v *StorageReferenceValue) SetMember(config *Config, name string, value Value) {
	referencedValue, err := v.dereference(config)
	if err != nil {
		panic(err)
	}

	memberAccessibleValue := (*referencedValue).(MemberAccessibleValue)
	memberAccessibleValue.SetMember(config, name, value)
}
