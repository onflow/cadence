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
	"github.com/onflow/cadence/runtime/sema"
)

// members

type CapabilityValue struct {
	Address    AddressValue
	Path       PathValue
	BorrowType StaticType
}

var _ Value = CapabilityValue{}

func NewCapabilityValue(address AddressValue, path PathValue, borrowType StaticType) CapabilityValue {
	return CapabilityValue{
		Address:    address,
		Path:       path,
		BorrowType: borrowType,
	}
}

func (CapabilityValue) isValue() {}

func (v CapabilityValue) StaticType(gauge common.MemoryGauge) StaticType {
	return interpreter.NewCapabilityStaticType(gauge, v.BorrowType)
}

func (v CapabilityValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v CapabilityValue) String() string {
	var borrowType string
	if v.BorrowType != nil {
		borrowType = v.BorrowType.String()
	}
	return format.Capability(
		borrowType,
		v.Address.String(),
		v.Path.String(),
	)
}

func init() {
	typeName := interpreter.PrimitiveStaticTypeCapability.String()

	// Capability.borrow
	RegisterTypeBoundFunction(typeName, sema.CapabilityTypeBorrowField, NativeFunctionValue{
		ParameterCount: 0,
		Function: func(config *Config, typeArguments []StaticType, args ...Value) Value {
			capabilityValue := args[0].(CapabilityValue)

			// NOTE: if a type argument is provided for the function,
			// use it *instead* of the type of the value (if any)
			var borrowType interpreter.ReferenceStaticType
			if len(typeArguments) > 0 {
				borrowType = typeArguments[0].(interpreter.ReferenceStaticType)
			} else {
				borrowType = capabilityValue.BorrowType.(interpreter.ReferenceStaticType)
			}

			address := capabilityValue.Address

			targetPath, authorized, err := getCapabilityFinalTargetPath(
				config.Storage,
				common.Address(address),
				capabilityValue.Path,
				borrowType,
			)
			if err != nil {
				panic(err)
			}

			reference := NewStorageReferenceValue(
				config.Storage,
				authorized,
				common.Address(address),
				targetPath,
				borrowType,
			)

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			value, err := reference.dereference(config.MemoryGauge)
			if err != nil {
				panic(err)
			}
			if value == nil {
				return NilValue{}
			}

			return NewSomeValueNonCopying(reference)
		},
	})
}

func getCapabilityFinalTargetPath(
	storage interpreter.Storage,
	address common.Address,
	path PathValue,
	wantedBorrowType interpreter.ReferenceStaticType,
) (
	finalPath PathValue,
	authorized bool,
	err error,
) {
	wantedReferenceType := wantedBorrowType

	seenPaths := map[PathValue]struct{}{}
	paths := []PathValue{path}

	for {
		// Detect cyclic links

		if _, ok := seenPaths[path]; ok {
			return EmptyPathValue, false, fmt.Errorf("cyclic link error")
		} else {
			seenPaths[path] = struct{}{}
		}

		value := ReadStored(
			nil,
			storage,
			address,
			path.Domain.Identifier(),
			path.Identifier,
		)

		if value == nil {
			return EmptyPathValue, false, nil
		}

		if link, ok := value.(LinkValue); ok {

			//allowedType := interpreter.MustConvertStaticToSemaType(link.Type)

			//if !sema.IsSubType(allowedType, wantedBorrowType) {
			//	return EmptyPathValue, false, nil
			//}

			targetPath := link.TargetPath
			paths = append(paths, targetPath)
			path = targetPath

		} else {
			return path, wantedReferenceType.Authorized, nil
		}
	}
}
