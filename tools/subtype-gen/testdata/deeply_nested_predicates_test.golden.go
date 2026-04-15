// Code generated from rules.yaml. DO NOT EDIT.
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

package sema

func CheckSubTypeWithoutEquality_gen(subType Type, superType Type) bool {
	if subType == NeverType {
		return true
	}

	switch typedSuperType := superType.(type) {
	case *IntersectionType:
		switch typedSuperType.LegacyType {
		case nil,
			AnyType,
			AnyStructType,
			AnyResourceType:
			switch subType {
			case AnyType,
				AnyStructType,
				AnyResourceType:
				return false
			}

			switch typedSubType := subType.(type) {
			case *IntersectionType:
				if typedSubType.LegacyType == nil &&
					IsIntersectionSubset(typedSuperType, typedSubType) {
					return true
				}

				switch typedSubType.LegacyType {
				case AnyType,
					AnyStructType,
					AnyResourceType:
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typedSubType.LegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typedSuperType, typedSubType)
				}

				return false
			}

			return false
		}

		switch typedSubType := subType.(type) {
		case *IntersectionType:
			switch typedSubType.LegacyType {
			case nil,
				AnyType,
				AnyStructType,
				AnyResourceType:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeType:
				return typedSubTypeLegacyType == typedSuperType.LegacyType
			}

			return false
		}

		return false

	}

	return false
}
