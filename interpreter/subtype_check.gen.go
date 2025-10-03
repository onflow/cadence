// Code generated from <no value>. DO NOT EDIT.
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

func checkSubTypeWithoutEquality_gen(typeConverter TypeConverter, subType StaticType, superType StaticType) bool {
	if subType == PrimitiveStaticTypeNever {
		return true
	}

	switch superType {
	}

	switch typedSuperType := superType.(type) {
	case *IntersectionStaticType:
		switch typedSuperType.LegacyType {
		case nil,
			PrimitiveStaticTypeAny,
			PrimitiveStaticTypeAnyStruct,
			PrimitiveStaticTypeAnyResource:
			switch subType {
			case PrimitiveStaticTypeAny,
				PrimitiveStaticTypeAnyStruct,
				PrimitiveStaticTypeAnyResource:
				return false
			}

			switch typedSubType := subType.(type) {
			case *IntersectionStaticType:
				return true

			case PrimitiveStaticTypeConforming:
				return true

			}

		}

		switch typedSubType := subType.(type) {
		case *IntersectionStaticType:
			switch typedSubType.LegacyType {
			case nil,
				PrimitiveStaticTypeAny,
				PrimitiveStaticTypeAnyStruct,
				PrimitiveStaticTypeAnyResource:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeStaticType:
				return typedSubTypeLegacyType == typedSuperType.LegacyType

			}

		case *CompositeStaticType:
			return IsSubType(typeConverter, typedSubType, typedSuperType.LegacyType)

		}

		switch subType {
		case PrimitiveStaticTypeAny,
			PrimitiveStaticTypeAnyStruct,
			PrimitiveStaticTypeAnyResource:
			return false
		}

		return false

	}

	return false
}
