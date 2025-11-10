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

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/common"
)

// TestCheckSubTypeWithoutEquality tests all paths of checkSubTypeWithoutEquality function
func TestCheckSubTypeWithoutEquality(t *testing.T) {
	t.Parallel()

	t.Run("NeverType", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			superType Type
		}{
			{"Never <: Any", AnyType},
			{"Never <: AnyStruct", AnyStructType},
			{"Never <: AnyResource", AnyResourceType},
			{"Never <: Int", IntType},
			{"Never <: String", StringType},
			{"Never <: Bool", BoolType},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := checkSubTypeWithoutEquality(NeverType, tt.superType)
				assert.True(t, result, "NeverType should be a subtype of %v", tt.superType)
			})
		}
	})

	t.Run("AnyType", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			subType Type
		}{
			{"Int <: Any", IntType},
			{"String <: Any", StringType},
			{"Bool <: Any", BoolType},
			{"AnyStruct <: Any", AnyStructType},
			{"AnyResource <: Any", AnyResourceType},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := checkSubTypeWithoutEquality(tt.subType, AnyType)
				assert.True(t, result, "%v should be a subtype of AnyType", tt.subType)
			})
		}
	})

	t.Run("AnyStructType", func(t *testing.T) {
		t.Parallel()

		t.Run("struct types are subtypes of AnyStruct", func(t *testing.T) {
			tests := []struct {
				name    string
				subType Type
			}{
				{"Int <: AnyStruct", IntType},
				{"String <: AnyStruct", StringType},
				{"Bool <: AnyStruct", BoolType},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, AnyStructType)
					assert.True(t, result, "%v should be a subtype of AnyStructType", tt.subType)
				})
			}
		})

		t.Run("resource types are NOT subtypes of AnyStruct", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(AnyResourceType, AnyStructType)
			assert.False(t, result, "AnyResource should NOT be a subtype of AnyStruct")
		})

		t.Run("AnyType is NOT a subtype of AnyStruct", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(AnyType, AnyStructType)
			assert.False(t, result, "AnyType should NOT be a subtype of AnyStruct")
		})
	})

	t.Run("AnyResourceType", func(t *testing.T) {
		t.Parallel()

		t.Run("resource types are subtypes of AnyResource", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(AnyResourceType, AnyResourceType)
			assert.True(t, result, "AnyResource should be a subtype of AnyResource")
		})

		t.Run("struct types are NOT subtypes of AnyResource", func(t *testing.T) {
			tests := []Type{
				IntType,
				StringType,
				BoolType,
				AnyStructType,
			}

			for _, subType := range tests {
				result := checkSubTypeWithoutEquality(subType, AnyResourceType)
				assert.False(t, result, "%v should NOT be a subtype of AnyResource", subType)
			}
		})
	})

	t.Run("AttachmentTypes", func(t *testing.T) {
		t.Parallel()

		t.Run("AnyResourceAttachment", func(t *testing.T) {
			// Note: Testing with real attachment types would require more setup
			// These tests verify the basic structure
			t.Run("non-resource is not subtype", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(IntType, AnyResourceAttachmentType)
				assert.False(t, result)
			})

			t.Run("struct is not subtype", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(StringType, AnyResourceAttachmentType)
				assert.False(t, result)
			})

			t.Run("AnyStruct is not subtype", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(AnyStructType, AnyResourceAttachmentType)
				assert.False(t, result)
			})
		})

		t.Run("AnyStructAttachment", func(t *testing.T) {
			t.Run("resource is not subtype", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(AnyResourceType, AnyStructAttachmentType)
				assert.False(t, result)
			})

			t.Run("non-attachment struct is not subtype", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(IntType, AnyStructAttachmentType)
				assert.False(t, result)
			})

			t.Run("AnyResource is not subtype", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(AnyResourceType, AnyStructAttachmentType)
				assert.False(t, result)
			})
		})
	})

	t.Run("HashableStructType", func(t *testing.T) {
		t.Parallel()

		t.Run("hashable types are subtypes", func(t *testing.T) {
			tests := []Type{
				IntType,
				StringType,
				BoolType,
				TheAddressType,
			}

			for _, subType := range tests {
				result := checkSubTypeWithoutEquality(subType, HashableStructType)
				assert.True(t, result, "%v should be a subtype of HashableStruct", subType)
			}
		})
	})

	t.Run("PathTypes", func(t *testing.T) {
		t.Parallel()

		t.Run("PathType", func(t *testing.T) {
			t.Run("StoragePath <: Path", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(StoragePathType, PathType)
				assert.True(t, result)
			})

			t.Run("PrivatePath <: Path", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(PrivatePathType, PathType)
				assert.True(t, result)
			})

			t.Run("PublicPath <: Path", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(PublicPathType, PathType)
				assert.True(t, result)
			})

			t.Run("Int is NOT <: Path", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(IntType, PathType)
				assert.False(t, result)
			})
		})

		t.Run("CapabilityPathType", func(t *testing.T) {
			t.Run("PrivatePath <: CapabilityPath", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(PrivatePathType, CapabilityPathType)
				assert.True(t, result)
			})

			t.Run("PublicPath <: CapabilityPath", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(PublicPathType, CapabilityPathType)
				assert.True(t, result)
			})

			t.Run("StoragePath is NOT <: CapabilityPath", func(t *testing.T) {
				result := checkSubTypeWithoutEquality(StoragePathType, CapabilityPathType)
				assert.False(t, result)
			})
		})
	})

	t.Run("StorableType", func(t *testing.T) {
		t.Parallel()

		t.Run("storable types are subtypes", func(t *testing.T) {
			tests := []Type{
				IntType,
				StringType,
				BoolType,
				TheAddressType,
			}

			for _, subType := range tests {
				result := checkSubTypeWithoutEquality(subType, StorableType)
				assert.True(t, result, "%v should be a subtype of Storable", subType)
			}
		})
	})

	t.Run("NumberTypes", func(t *testing.T) {
		t.Parallel()

		t.Run("NumberType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  Type
				expected bool
			}{
				{"NumberType <: Number", NumberType, true},
				{"SignedNumberType <: Number", SignedNumberType, true},
				{"Int <: Number", IntType, true},
				{"Int8 <: Number", Int8Type, true},
				{"UInt <: Number", UIntType, true},
				{"Fix64 <: Number", Fix64Type, true},
				{"UFix64 <: Number", UFix64Type, true},
				{"String is NOT <: Number", StringType, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, NumberType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("SignedNumberType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  Type
				expected bool
			}{
				{"SignedNumberType <: SignedNumber", SignedNumberType, true},
				{"Int <: SignedNumber", IntType, true},
				{"Int8 <: SignedNumber", Int8Type, true},
				{"Fix64 <: SignedNumber", Fix64Type, true},
				{"UInt is NOT <: SignedNumber", UIntType, false},
				{"UFix64 is NOT <: SignedNumber", UFix64Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, SignedNumberType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("IntegerType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  Type
				expected bool
			}{
				{"IntegerType <: Integer", IntegerType, true},
				{"SignedIntegerType <: Integer", SignedIntegerType, true},
				{"FixedSizeUnsignedIntegerType <: Integer", FixedSizeUnsignedIntegerType, true},
				{"UIntType <: Integer", UIntType, true},
				{"Int <: Integer", IntType, true},
				{"UInt8 <: Integer", UInt8Type, true},
				{"Fix64 is NOT <: Integer", Fix64Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, IntegerType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("SignedIntegerType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  Type
				expected bool
			}{
				{"SignedIntegerType <: SignedInteger", SignedIntegerType, true},
				{"Int <: SignedInteger", IntType, true},
				{"Int8 <: SignedInteger", Int8Type, true},
				{"Int16 <: SignedInteger", Int16Type, true},
				{"Int32 <: SignedInteger", Int32Type, true},
				{"Int64 <: SignedInteger", Int64Type, true},
				{"Int128 <: SignedInteger", Int128Type, true},
				{"Int256 <: SignedInteger", Int256Type, true},
				{"UInt is NOT <: SignedInteger", UIntType, false},
				{"UInt8 is NOT <: SignedInteger", UInt8Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, SignedIntegerType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("FixedSizeUnsignedIntegerType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  Type
				expected bool
			}{
				{"UInt8 <: FixedSizeUnsignedInteger", UInt8Type, true},
				{"UInt16 <: FixedSizeUnsignedInteger", UInt16Type, true},
				{"UInt32 <: FixedSizeUnsignedInteger", UInt32Type, true},
				{"UInt64 <: FixedSizeUnsignedInteger", UInt64Type, true},
				{"UInt128 <: FixedSizeUnsignedInteger", UInt128Type, true},
				{"UInt256 <: FixedSizeUnsignedInteger", UInt256Type, true},
				{"Word8 <: FixedSizeUnsignedInteger", Word8Type, true},
				{"Word16 <: FixedSizeUnsignedInteger", Word16Type, true},
				{"Word32 <: FixedSizeUnsignedInteger", Word32Type, true},
				{"Word64 <: FixedSizeUnsignedInteger", Word64Type, true},
				{"Word128 <: FixedSizeUnsignedInteger", Word128Type, true},
				{"Word256 <: FixedSizeUnsignedInteger", Word256Type, true},
				{"UInt is NOT <: FixedSizeUnsignedInteger", UIntType, false},
				{"Int is NOT <: FixedSizeUnsignedInteger", IntType, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, FixedSizeUnsignedIntegerType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("FixedPointType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  Type
				expected bool
			}{
				{"FixedPointType <: FixedPoint", FixedPointType, true},
				{"SignedFixedPointType <: FixedPoint", SignedFixedPointType, true},
				{"UFix64 <: FixedPoint", UFix64Type, true},
				{"UFix128 <: FixedPoint", UFix128Type, true},
				{"Fix64 <: FixedPoint", Fix64Type, true},
				{"Fix128 <: FixedPoint", Fix128Type, true},
				{"Int is NOT <: FixedPoint", IntType, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, FixedPointType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("SignedFixedPointType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  Type
				expected bool
			}{
				{"SignedFixedPointType <: SignedFixedPoint", SignedFixedPointType, true},
				{"Fix64 <: SignedFixedPoint", Fix64Type, true},
				{"Fix128 <: SignedFixedPoint", Fix128Type, true},
				{"UFix64 is NOT <: SignedFixedPoint", UFix64Type, false},
				{"UFix128 is NOT <: SignedFixedPoint", UFix128Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeWithoutEquality(tt.subType, SignedFixedPointType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})
	})

	t.Run("OptionalType", func(t *testing.T) {
		t.Parallel()

		t.Run("T <: T?", func(t *testing.T) {
			optionalInt := &OptionalType{Type: IntType}
			result := checkSubTypeWithoutEquality(IntType, optionalInt)
			assert.True(t, result, "Int should be a subtype of Int?")
		})

		t.Run("T? <: U? when T <: U", func(t *testing.T) {
			optionalNumber := &OptionalType{Type: NumberType}
			optionalInt := &OptionalType{Type: IntType}
			result := checkSubTypeWithoutEquality(optionalInt, optionalNumber)
			assert.True(t, result, "Int? should be a subtype of Number?")
		})

		t.Run("T? is NOT <: U? when T is NOT <: U", func(t *testing.T) {
			optionalInt := &OptionalType{Type: IntType}
			optionalString := &OptionalType{Type: StringType}
			result := checkSubTypeWithoutEquality(optionalInt, optionalString)
			assert.False(t, result, "Int? should NOT be a subtype of String?")
		})
	})

	t.Run("DictionaryType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in key and value types", func(t *testing.T) {
			dict1 := &DictionaryType{
				KeyType:   IntType,
				ValueType: IntType,
			}
			dict2 := &DictionaryType{
				KeyType:   NumberType,
				ValueType: NumberType,
			}
			result := checkSubTypeWithoutEquality(dict1, dict2)
			assert.True(t, result, "{Int: Int} should be a subtype of {Number: Number}")
		})

		t.Run("not subtype when key types don't match", func(t *testing.T) {
			dict1 := &DictionaryType{
				KeyType:   IntType,
				ValueType: IntType,
			}
			dict2 := &DictionaryType{
				KeyType:   StringType,
				ValueType: IntType,
			}
			result := checkSubTypeWithoutEquality(dict1, dict2)
			assert.False(t, result, "{Int: Int} should NOT be a subtype of {String: Int}")
		})

		t.Run("not subtype when value types don't match", func(t *testing.T) {
			dict1 := &DictionaryType{
				KeyType:   IntType,
				ValueType: IntType,
			}
			dict2 := &DictionaryType{
				KeyType:   IntType,
				ValueType: StringType,
			}
			result := checkSubTypeWithoutEquality(dict1, dict2)
			assert.False(t, result, "{Int: Int} should NOT be a subtype of {Int: String}")
		})

		t.Run("non-dictionary is not subtype", func(t *testing.T) {
			dict := &DictionaryType{
				KeyType:   IntType,
				ValueType: StringType,
			}
			result := checkSubTypeWithoutEquality(IntType, dict)
			assert.False(t, result, "Int should NOT be a subtype of {Int: String}")
		})
	})

	t.Run("VariableSizedType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in element type", func(t *testing.T) {
			arr1 := &VariableSizedType{Type: IntType}
			arr2 := &VariableSizedType{Type: NumberType}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.True(t, result, "[Int] should be a subtype of [Number]")
		})

		t.Run("not subtype when element types don't match", func(t *testing.T) {
			arr1 := &VariableSizedType{Type: IntType}
			arr2 := &VariableSizedType{Type: StringType}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.False(t, result, "[Int] should NOT be a subtype of [String]")
		})

		t.Run("non-array is not subtype", func(t *testing.T) {
			arr := &VariableSizedType{Type: IntType}
			result := checkSubTypeWithoutEquality(IntType, arr)
			assert.False(t, result, "Int should NOT be a subtype of [Int]")
		})
	})

	t.Run("ConstantSizedType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in element type with same size", func(t *testing.T) {
			arr1 := &ConstantSizedType{Type: IntType, Size: 5}
			arr2 := &ConstantSizedType{Type: NumberType, Size: 5}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.True(t, result, "[Int; 5] should be a subtype of [Number; 5]")
		})

		t.Run("not subtype when sizes differ", func(t *testing.T) {
			arr1 := &ConstantSizedType{Type: IntType, Size: 5}
			arr2 := &ConstantSizedType{Type: IntType, Size: 10}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.False(t, result, "[Int; 5] should NOT be a subtype of [Int; 10]")
		})

		t.Run("not subtype when element types don't match", func(t *testing.T) {
			arr1 := &ConstantSizedType{Type: IntType, Size: 5}
			arr2 := &ConstantSizedType{Type: StringType, Size: 5}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.False(t, result, "[Int; 5] should NOT be a subtype of [String; 5]")
		})

		t.Run("non-array is not subtype", func(t *testing.T) {
			arr := &ConstantSizedType{Type: IntType, Size: 5}
			result := checkSubTypeWithoutEquality(IntType, arr)
			assert.False(t, result, "Int should NOT be a subtype of [Int; 5]")
		})
	})

	t.Run("ReferenceType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in referenced type with compatible authorization", func(t *testing.T) {
			ref1 := &ReferenceType{
				Type:          IntType,
				Authorization: UnauthorizedAccess,
			}
			ref2 := &ReferenceType{
				Type:          NumberType,
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(ref1, ref2)
			assert.True(t, result, "&Int should be a subtype of &Number")
		})

		t.Run("not subtype when authorization doesn't permit", func(t *testing.T) {
			entitlement := NewEntitlementType(nil, common.NewStringLocation(nil, "test"), "E")
			auth := NewEntitlementSetAccess([]*EntitlementType{entitlement}, Disjunction)

			ref1 := &ReferenceType{
				Type:          IntType,
				Authorization: UnauthorizedAccess,
			}
			ref2 := &ReferenceType{
				Type:          IntType,
				Authorization: auth,
			}
			result := checkSubTypeWithoutEquality(ref1, ref2)
			assert.False(t, result, "unauthorized reference should NOT be a subtype of authorized reference")
		})

		t.Run("not subtype when referenced types don't match", func(t *testing.T) {
			ref1 := &ReferenceType{
				Type:          IntType,
				Authorization: UnauthorizedAccess,
			}
			ref2 := &ReferenceType{
				Type:          StringType,
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(ref1, ref2)
			assert.False(t, result, "&Int should NOT be a subtype of &String")
		})

		t.Run("non-reference is not subtype", func(t *testing.T) {
			ref := &ReferenceType{
				Type:          IntType,
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(IntType, ref)
			assert.False(t, result, "Int should NOT be a subtype of &Int")
		})

		t.Run("reference to resource type", func(t *testing.T) {
			// &AnyResource <: &AnyResource
			ref1 := &ReferenceType{
				Type:          AnyResourceType,
				Authorization: UnauthorizedAccess,
			}
			ref2 := &ReferenceType{
				Type:          AnyResourceType,
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(ref1, ref2)
			assert.True(t, result, "&AnyResource should be a subtype of &AnyResource")
		})

		t.Run("reference to optional type", func(t *testing.T) {
			// &Int <: &Int?
			refToInt := &ReferenceType{
				Type:          IntType,
				Authorization: UnauthorizedAccess,
			}
			refToOptInt := &ReferenceType{
				Type:          &OptionalType{Type: IntType},
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(refToInt, refToOptInt)
			assert.True(t, result, "&Int should be a subtype of &Int?")
		})

		t.Run("reference with same authorization different types", func(t *testing.T) {
			// &String is NOT <: &Int even with same auth
			ref1 := &ReferenceType{
				Type:          StringType,
				Authorization: UnauthorizedAccess,
			}
			ref2 := &ReferenceType{
				Type:          IntType,
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(ref1, ref2)
			assert.False(t, result, "&String should NOT be a subtype of &Int")
		})
	})

	t.Run("FunctionType", func(t *testing.T) {
		t.Parallel()

		t.Run("view function is subtype of impure function", func(t *testing.T) {
			viewFunc := &FunctionType{
				Purity: FunctionPurityView,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			impureFunc := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			result := checkSubTypeWithoutEquality(viewFunc, impureFunc)
			assert.True(t, result, "view function should be a subtype of impure function")
		})

		t.Run("impure function is NOT subtype of view function", func(t *testing.T) {
			viewFunc := &FunctionType{
				Purity: FunctionPurityView,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			impureFunc := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			result := checkSubTypeWithoutEquality(impureFunc, viewFunc)
			assert.False(t, result, "impure function should NOT be a subtype of view function")
		})

		t.Run("contravariant in parameter types", func(t *testing.T) {
			// fun(Number): Int  <:  fun(Int): Int
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(NumberType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.True(t, result, "fun(Number): Int should be a subtype of fun(Int): Int")
		})

		t.Run("covariant in return type", func(t *testing.T) {
			// fun(Int): Int  <:  fun(Int): Number
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(NumberType),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.True(t, result, "fun(Int): Int should be a subtype of fun(Int): Number")
		})

		t.Run("not subtype when parameter arity differs", func(t *testing.T) {
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.False(t, result, "functions with different arities should NOT be subtypes")
		})

		t.Run("not subtype when constructor flags differ", func(t *testing.T) {
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				IsConstructor:        true,
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				IsConstructor:        false,
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.False(t, result, "constructor and non-constructor functions should NOT be subtypes")
		})

		t.Run("non-function is not subtype", func(t *testing.T) {
			fn := &FunctionType{
				Purity:               FunctionPurityImpure,
				Parameters:           []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			result := checkSubTypeWithoutEquality(IntType, fn)
			assert.False(t, result, "Int should NOT be a subtype of function")
		})

		t.Run("function with Void return type", func(t *testing.T) {
			// fun(Int): Void <: fun(Int): Void
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: VoidTypeAnnotation,
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: VoidTypeAnnotation,
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.True(t, result, "functions with Void return should be subtypes")
		})

		t.Run("function with different return types", func(t *testing.T) {
			// fun(Int): String is NOT <: fun(Int): Int
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(StringType),
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.False(t, result, "function with String return should NOT be subtype of Int return")
		})

		t.Run("function with different arity", func(t *testing.T) {
			// fun() is NOT <: fun(Int)
			func1 := &FunctionType{
				Purity:               FunctionPurityImpure,
				Parameters:           []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				Arity:                &Arity{Min: 0, Max: 0},
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				Arity:                &Arity{Min: 1, Max: 1},
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.False(t, result, "functions with different arity should NOT be subtypes")
		})

		t.Run("function with type parameters not equal", func(t *testing.T) {
			// fun<T: Int>() is NOT <: fun<T: String>()
			typeParam1 := &TypeParameter{
				Name:      "T",
				TypeBound: IntType,
			}
			typeParam2 := &TypeParameter{
				Name:      "T",
				TypeBound: StringType,
			}
			func1 := &FunctionType{
				Purity:               FunctionPurityImpure,
				Parameters:           []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				TypeParameters:       []*TypeParameter{typeParam1},
			}
			func2 := &FunctionType{
				Purity:               FunctionPurityImpure,
				Parameters:           []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				TypeParameters:       []*TypeParameter{typeParam2},
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.False(t, result, "functions with different type parameter bounds should NOT be subtypes")
		})

		t.Run("function with different type parameter count", func(t *testing.T) {
			// fun<T>() is NOT <: fun<T, U>()
			typeParam := &TypeParameter{
				Name:      "T",
				TypeBound: IntType,
			}
			func1 := &FunctionType{
				Purity:               FunctionPurityImpure,
				Parameters:           []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				TypeParameters:       []*TypeParameter{typeParam},
			}
			func2 := &FunctionType{
				Purity:               FunctionPurityImpure,
				Parameters:           []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
				TypeParameters:       []*TypeParameter{typeParam, typeParam},
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.False(t, result, "functions with different type parameter count should NOT be subtypes")
		})

		t.Run("function returning array", func(t *testing.T) {
			// fun(): [Int] <: fun(): [Number]
			func1 := &FunctionType{
				Purity:     FunctionPurityImpure,
				Parameters: []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(
					&VariableSizedType{Type: IntType},
				),
			}
			func2 := &FunctionType{
				Purity:     FunctionPurityImpure,
				Parameters: []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(
					&VariableSizedType{Type: NumberType},
				),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.True(t, result, "fun(): [Int] should be subtype of fun(): [Number]")
		})

		t.Run("function with array parameter", func(t *testing.T) {
			// fun([Number]): Void <: fun([Int]): Void (contravariance)
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(&VariableSizedType{Type: NumberType})},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(&VariableSizedType{Type: IntType})},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.True(t, result, "fun([Number]): Void should be subtype of fun([Int]): Void")
		})
	})

	t.Run("IntersectionType", func(t *testing.T) {
		t.Parallel()

		// Create test interfaces for intersection types
		location := common.NewStringLocation(nil, "test")

		interfaceType1 := &InterfaceType{
			Location:      location,
			Identifier:    "I1",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		interfaceType2 := &InterfaceType{
			Location:      location,
			Identifier:    "I2",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		t.Run("AnyResource intersection with nil subtype", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(
				AnyResourceType,
				&IntersectionType{
					LegacyType: AnyResourceType,
					Types:      []*InterfaceType{interfaceType1},
				},
			)
			assert.False(t, result, "AnyResource should NOT be a subtype of AnyResource{I}")
		})

		t.Run("AnyStruct intersection with nil subtype", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(
				AnyStructType,
				&IntersectionType{
					LegacyType: AnyStructType,
					Types:      []*InterfaceType{interfaceType1},
				},
			)
			assert.False(t, result, "AnyStruct should NOT be a subtype of AnyStruct{I}")
		})

		t.Run("Any intersection with nil subtype", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(
				AnyType,
				&IntersectionType{
					LegacyType: AnyType,
					Types:      []*InterfaceType{interfaceType1},
				},
			)
			assert.False(t, result, "Any should NOT be a subtype of Any{I}")
		})

		// Tests for IntersectionType subtype with nil LegacyType
		t.Run("intersection with nil legacy type as subtype", func(t *testing.T) {
			// {I1, I2} <: {I1} when I1 is subset of {I1, I2}
			subType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType1, interfaceType2},
			}
			superType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.True(t, result, "{I1, I2} should be a subtype of {I1}")
		})

		t.Run("intersection with nil legacy type not subtype when not subset", func(t *testing.T) {
			// {I1} is NOT <: {I2}
			subType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "{I1} should NOT be a subtype of {I2}")
		})

		// Tests for IntersectionType subtype with AnyResource LegacyType
		t.Run("AnyResource intersection subtype with matching interfaces", func(t *testing.T) {
			interfaceType2 := &InterfaceType{
				Location:      location,
				Identifier:    "I2",
				CompositeKind: common.CompositeKindResource,
				Members:       &StringMemberOrderedMap{},
			}

			// AnyResource{I1, I2} <: AnyResource{I1}
			subType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{interfaceType1, interfaceType2},
			}
			superType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.True(t, result, "AnyResource{I1, I2} should be a subtype of AnyResource{I1}")
		})

		t.Run("AnyResource intersection not subtype of AnyStruct intersection", func(t *testing.T) {
			subType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "AnyResource{I} should NOT be a subtype of AnyStruct{I}")
		})

		// Tests for IntersectionType subtype with CompositeType LegacyType
		t.Run("composite intersection subtype when interfaces match", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "R",
				Kind:       common.CompositeKindResource,
				Members:    &StringMemberOrderedMap{},
			}
			compositeType.ExplicitInterfaceConformances = []*InterfaceType{interfaceType1}

			// R{I1} <: AnyResource{} when R conforms to I1
			subType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.True(t, result, "R{I1} should be a subtype of AnyResource{}")
		})

		t.Run("composite intersection not subtype when composite doesn't conform", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "R",
				Kind:       common.CompositeKindResource,
				Members:    &StringMemberOrderedMap{},
			}
			// No conformance set

			interfaceType2 := &InterfaceType{
				Location:      location,
				Identifier:    "I2",
				CompositeKind: common.CompositeKindResource,
				Members:       &StringMemberOrderedMap{},
			}

			// R{I2} is NOT <: AnyResource{I1} when R doesn't conform to I1
			subType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType2},
			}
			superType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "R{I2} should NOT be a subtype of AnyResource{I1} when R doesn't conform")
		})

		// Tests for ConformingType (CompositeType) as subtype of IntersectionType
		t.Run("composite type subtype of intersection when conforming", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}
			compositeType.ExplicitInterfaceConformances = []*InterfaceType{interfaceType1}

			// S <: AnyStruct{I1} when S conforms to I1
			superType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(compositeType, superType)
			assert.True(t, result, "composite should be a subtype of intersection when conforming")
		})

		t.Run("composite type not subtype of intersection when not conforming", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}
			// No conformance

			interfaceType2 := &InterfaceType{
				Location:      location,
				Identifier:    "I2",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			// S is NOT <: AnyStruct{I2} when S doesn't conform to I2
			superType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(compositeType, superType)
			assert.False(t, result, "composite should NOT be a subtype of intersection when not conforming")
		})

		t.Run("composite type not subtype when wrong base type", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "R",
				Kind:       common.CompositeKindResource,
				Members:    &StringMemberOrderedMap{},
			}
			compositeType.ExplicitInterfaceConformances = []*InterfaceType{interfaceType1}

			// Resource R is NOT <: AnyStruct{I1} even if conforming
			superType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(compositeType, superType)
			assert.False(t, result, "resource should NOT be a subtype of struct intersection")
		})

		// Tests for IntersectionType supertype with non-Any* legacy type
		t.Run("intersection with composite legacy type", func(t *testing.T) {
			compositeType1 := &CompositeType{
				Location:   location,
				Identifier: "S1",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// S1{I1} <: S1{I2} when S1 == S1 (owner may restrict/unrestrict)
			subType := &IntersectionType{
				LegacyType: compositeType1,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: compositeType1,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.True(t, result, "S1{I1} should be a subtype of S1{I2} when same composite")
		})

		t.Run("intersection with different composite legacy types", func(t *testing.T) {
			compositeType1 := &CompositeType{
				Location:   location,
				Identifier: "S1",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			compositeType2 := &CompositeType{
				Location:   location,
				Identifier: "S2",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// S1{I1} is NOT <: S2{I1} when S1 != S2
			subType := &IntersectionType{
				LegacyType: compositeType1,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: compositeType2,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "S1{I1} should NOT be a subtype of S2{I1} when different composites")
		})

		t.Run("composite type subtype of composite intersection", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// S <: S{I1} (owner may freely restrict)
			superType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(compositeType, superType)
			assert.True(t, result, "composite should be a subtype of its own intersection type")
		})

		t.Run("intersection nil legacy type not subtype of composite intersection", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// {I1} is NOT <: S{I1} statically
			subType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "{I1} should NOT be statically a subtype of S{I1}")
		})

		t.Run("intersection Any* legacy type not subtype of composite intersection", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// AnyStruct{I1} is NOT <: S{I1} statically
			subType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "AnyStruct{I1} should NOT be statically a subtype of S{I1}")
		})

		// Tests for lines 7872-7880: AnyResourceType/AnyStructType/AnyType not subtype of composite intersection
		t.Run("AnyResourceType not subtype of composite intersection", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "R",
				Kind:       common.CompositeKindResource,
				Members:    &StringMemberOrderedMap{},
			}

			// AnyResource is NOT <: R{I1} statically
			superType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(AnyResourceType, superType)
			assert.False(t, result, "AnyResource should NOT be statically a subtype of R{I1}")
		})

		t.Run("AnyStructType not subtype of composite intersection", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// AnyStruct is NOT <: S{I1} statically
			superType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(AnyStructType, superType)
			assert.False(t, result, "AnyStruct should NOT be statically a subtype of S{I1}")
		})

		t.Run("AnyType not subtype of composite intersection", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// Any is NOT <: S{I1} statically
			superType := &IntersectionType{
				LegacyType: compositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			result := checkSubTypeWithoutEquality(AnyType, superType)
			assert.False(t, result, "Any should NOT be statically a subtype of S{I1}")
		})

		// Tests for lines 7812-7816: composite legacy type in subtype vs Any* legacy type in supertype
		t.Run("struct composite intersection not subtype of AnyResource intersection", func(t *testing.T) {
			structCompositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// S{I1} is NOT <: AnyResource{I2} because S is not a resource
			subType := &IntersectionType{
				LegacyType: structCompositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "S{I1} should NOT be subtype of AnyResource{I2} (struct vs resource)")
		})

		t.Run("resource composite intersection is subtype of AnyResource intersection with conformance", func(t *testing.T) {
			resourceCompositeType := &CompositeType{
				Location:                      location,
				Identifier:                    "R",
				Kind:                          common.CompositeKindResource,
				Members:                       &StringMemberOrderedMap{},
				ExplicitInterfaceConformances: []*InterfaceType{interfaceType1, interfaceType2},
			}

			// R{I1} <: AnyResource{I2} if R is a resource and R conforms to I2
			subType := &IntersectionType{
				LegacyType: resourceCompositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.True(t, result, "R{I1} should be subtype of AnyResource{I2} when R conforms to I2")
		})

		t.Run("resource composite intersection not subtype of AnyResource without conformance", func(t *testing.T) {
			resourceCompositeType := &CompositeType{
				Location:                      location,
				Identifier:                    "R",
				Kind:                          common.CompositeKindResource,
				Members:                       &StringMemberOrderedMap{},
				ExplicitInterfaceConformances: []*InterfaceType{interfaceType1},
			}

			// R{I1} is NOT <: AnyResource{I2} if R doesn't conform to I2
			subType := &IntersectionType{
				LegacyType: resourceCompositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyResourceType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "R{I1} should NOT be subtype of AnyResource{I2} when R doesn't conform to I2")
		})

		t.Run("struct composite intersection is subtype of AnyStruct intersection with conformance", func(t *testing.T) {
			structCompositeType := &CompositeType{
				Location:                      location,
				Identifier:                    "S",
				Kind:                          common.CompositeKindStructure,
				Members:                       &StringMemberOrderedMap{},
				ExplicitInterfaceConformances: []*InterfaceType{interfaceType1, interfaceType2},
			}

			// S{I1} <: AnyStruct{I2} if S is a struct and S conforms to I2
			subType := &IntersectionType{
				LegacyType: structCompositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.True(t, result, "S{I1} should be subtype of AnyStruct{I2} when S conforms to I2")
		})

		t.Run("resource composite intersection not subtype of AnyStruct intersection", func(t *testing.T) {
			resourceCompositeType := &CompositeType{
				Location:   location,
				Identifier: "R",
				Kind:       common.CompositeKindResource,
				Members:    &StringMemberOrderedMap{},
			}

			// R{I1} is NOT <: AnyStruct{I2} because R is a resource, not a struct
			subType := &IntersectionType{
				LegacyType: resourceCompositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.False(t, result, "R{I1} should NOT be subtype of AnyStruct{I2} (resource vs struct)")
		})

		t.Run("composite intersection is subtype of AnyType intersection with conformance", func(t *testing.T) {
			structCompositeType := &CompositeType{
				Location:                      location,
				Identifier:                    "S",
				Kind:                          common.CompositeKindStructure,
				Members:                       &StringMemberOrderedMap{},
				ExplicitInterfaceConformances: []*InterfaceType{interfaceType1, interfaceType2},
			}

			// S{I1} <: Any{I2} if S conforms to I2
			subType := &IntersectionType{
				LegacyType: structCompositeType,
				Types:      []*InterfaceType{interfaceType1},
			}
			superType := &IntersectionType{
				LegacyType: AnyType,
				Types:      []*InterfaceType{interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, superType)
			assert.True(t, result, "S{I1} should be subtype of Any{I2} when S conforms to I2")
		})

		// Test intersection subtype with IntersectionType interface supertype
		t.Run("intersection type subtype of interface type", func(t *testing.T) {
			// {I1, I2} <: I1 when I1 is in the intersection set
			subType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType1, interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, interfaceType1)
			assert.True(t, result, "{I1, I2} should be a subtype of I1")
		})

		t.Run("intersection type not subtype of interface not in set", func(t *testing.T) {
			interfaceType3 := &InterfaceType{
				Location:      location,
				Identifier:    "I3",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			// {I1, I2} is NOT <: I3 when I3 is not in the intersection set
			subType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType1, interfaceType2},
			}
			result := checkSubTypeWithoutEquality(subType, interfaceType3)
			assert.False(t, result, "{I1, I2} should NOT be a subtype of I3")
		})
	})

	t.Run("CompositeType", func(t *testing.T) {
		t.Parallel()

		location := common.NewStringLocation(nil, "test")

		composite1 := &CompositeType{
			Location:   location,
			Identifier: "S1",
			Kind:       common.CompositeKindStructure,
			Members:    &StringMemberOrderedMap{},
		}

		composite2 := &CompositeType{
			Location:   location,
			Identifier: "S2",
			Kind:       common.CompositeKindStructure,
			Members:    &StringMemberOrderedMap{},
		}

		t.Run("different composite types are not subtypes", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(composite1, composite2)
			assert.False(t, result, "different composite types should NOT be subtypes")
		})

		t.Run("non-composite is not subtype of composite", func(t *testing.T) {
			result := checkSubTypeWithoutEquality(IntType, composite1)
			assert.False(t, result, "Int should NOT be a subtype of composite")
		})

		// Tests for IntersectionType subtype with CompositeType supertype
		t.Run("intersection with nil legacy type not subtype of composite", func(t *testing.T) {
			interfaceType := &InterfaceType{
				Location:      location,
				Identifier:    "I",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			// {I} is NOT <: S statically
			subType := &IntersectionType{
				LegacyType: nil,
				Types:      []*InterfaceType{interfaceType},
			}
			result := checkSubTypeWithoutEquality(subType, composite1)
			assert.False(t, result, "{I} should NOT be statically a subtype of S")
		})

		t.Run("intersection with Any* legacy type not subtype of composite", func(t *testing.T) {
			interfaceType := &InterfaceType{
				Location:      location,
				Identifier:    "I",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			// AnyStruct{I} is NOT <: S statically
			subType := &IntersectionType{
				LegacyType: AnyStructType,
				Types:      []*InterfaceType{interfaceType},
			}
			result := checkSubTypeWithoutEquality(subType, composite1)
			assert.False(t, result, "AnyStruct{I} should NOT be statically a subtype of S")
		})

		t.Run("intersection with matching composite legacy type", func(t *testing.T) {
			interfaceType := &InterfaceType{
				Location:      location,
				Identifier:    "I",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			// S{I} <: S (owner may freely unrestrict)
			subType := &IntersectionType{
				LegacyType: composite1,
				Types:      []*InterfaceType{interfaceType},
			}
			result := checkSubTypeWithoutEquality(subType, composite1)
			assert.True(t, result, "S{I} should be a subtype of S (unrestrict)")
		})

		t.Run("intersection with different composite legacy type", func(t *testing.T) {
			interfaceType := &InterfaceType{
				Location:      location,
				Identifier:    "I",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			// S1{I} is NOT <: S2 when S1 != S2
			subType := &IntersectionType{
				LegacyType: composite2,
				Types:      []*InterfaceType{interfaceType},
			}
			result := checkSubTypeWithoutEquality(subType, composite1)
			assert.False(t, result, "S1{I} should NOT be a subtype of S2")
		})
	})

	t.Run("InterfaceType", func(t *testing.T) {
		t.Parallel()

		location := common.NewStringLocation(nil, "test")

		interfaceType := &InterfaceType{
			Location:      location,
			Identifier:    "I",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		t.Run("composite conforming to interface", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// Set up conformance
			compositeType.ExplicitInterfaceConformances = []*InterfaceType{interfaceType}

			result := checkSubTypeWithoutEquality(compositeType, interfaceType)
			assert.True(t, result, "composite conforming to interface should be a subtype")
		})

		t.Run("composite NOT conforming to interface", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
				Members:    &StringMemberOrderedMap{},
			}

			// No conformance set

			result := checkSubTypeWithoutEquality(compositeType, interfaceType)
			assert.False(t, result, "composite NOT conforming to interface should NOT be a subtype")
		})

		t.Run("composite with wrong kind", func(t *testing.T) {
			compositeType := &CompositeType{
				Location:   location,
				Identifier: "R",
				Kind:       common.CompositeKindResource, // Different kind
				Members:    &StringMemberOrderedMap{},
			}

			result := checkSubTypeWithoutEquality(compositeType, interfaceType)
			assert.False(t, result, "composite with different kind should NOT be a subtype")
		})

		t.Run("interface subtype of interface", func(t *testing.T) {
			interface1 := &InterfaceType{
				Location:      location,
				Identifier:    "I1",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			interface2 := &InterfaceType{
				Location:      location,
				Identifier:    "I2",
				CompositeKind: common.CompositeKindStructure,
				Members:       &StringMemberOrderedMap{},
			}

			// Set up conformance: I1 conforms to I2
			interface1.ExplicitInterfaceConformances = []*InterfaceType{interface2}

			result := checkSubTypeWithoutEquality(interface1, interface2)
			assert.True(t, result, "interface conforming to another interface should be a subtype")
		})
	})

	t.Run("ParameterizedType", func(t *testing.T) {
		t.Parallel()

		// This test uses CapabilityType which is a ParameterizedType

		t.Run("capability with matching borrow type", func(t *testing.T) {
			capability1 := &CapabilityType{
				BorrowType: IntType,
			}
			capability2 := &CapabilityType{
				BorrowType: NumberType,
			}

			result := checkSubTypeWithoutEquality(capability1, capability2)
			assert.True(t, result, "Capability<Int> should be a subtype of Capability<Number>")
		})

		t.Run("capability with non-matching borrow type", func(t *testing.T) {
			capability1 := &CapabilityType{
				BorrowType: IntType,
			}
			capability2 := &CapabilityType{
				BorrowType: StringType,
			}

			result := checkSubTypeWithoutEquality(capability1, capability2)
			assert.False(t, result, "Capability<Int> should NOT be a subtype of Capability<String>")
		})

		t.Run("non-capability is not subtype of capability", func(t *testing.T) {
			capability := &CapabilityType{
				BorrowType: IntType,
			}

			result := checkSubTypeWithoutEquality(IntType, capability)
			// This may pass or fail depending on IsParameterizedSubType fallback
			// The function checks if IntType's base type is a subtype of capability
			assert.False(t, result, "Int should NOT be a subtype of Capability<Int>")
		})

		// More comprehensive ParameterizedType tests
		t.Run("parameterized type with base types not matching", func(t *testing.T) {
			// This tests the case where base types don't match
			capability := &CapabilityType{
				BorrowType: IntType,
			}
			inclusiveRange := &InclusiveRangeType{
				MemberType: IntType,
			}

			// The base types need to be subtypes first
			result := checkSubTypeWithoutEquality(capability, inclusiveRange)
			assert.False(t, result, "base types must match for parameterized subtypes")
		})
	})

	t.Run("EdgeCases", func(t *testing.T) {
		t.Parallel()

		t.Run("nested optionals", func(t *testing.T) {
			// Int <: Int??
			doubleOptionalInt := &OptionalType{
				Type: &OptionalType{Type: IntType},
			}
			result := checkSubTypeWithoutEquality(IntType, doubleOptionalInt)
			assert.True(t, result, "Int should be a subtype of Int??")
		})

		t.Run("triple nested optionals", func(t *testing.T) {
			// Int <: Int???
			tripleOptionalInt := &OptionalType{
				Type: &OptionalType{
					Type: &OptionalType{Type: IntType},
				},
			}
			result := checkSubTypeWithoutEquality(IntType, tripleOptionalInt)
			assert.True(t, result, "Int should be a subtype of Int???")
		})

		t.Run("optional subtype with non-subtype base", func(t *testing.T) {
			// String? is NOT <: Int?
			optString := &OptionalType{Type: StringType}
			optInt := &OptionalType{Type: IntType}
			result := checkSubTypeWithoutEquality(optString, optInt)
			assert.False(t, result, "String? should NOT be a subtype of Int?")
		})

		t.Run("nested arrays", func(t *testing.T) {
			// [[Int]] <: [[Number]]
			arrInt := &VariableSizedType{
				Type: &VariableSizedType{Type: IntType},
			}
			arrNumber := &VariableSizedType{
				Type: &VariableSizedType{Type: NumberType},
			}
			result := checkSubTypeWithoutEquality(arrInt, arrNumber)
			assert.True(t, result, "[[Int]] should be a subtype of [[Number]]")
		})

		t.Run("dictionary with optional values", func(t *testing.T) {
			dict1 := &DictionaryType{
				KeyType:   StringType,
				ValueType: IntType,
			}
			dict2 := &DictionaryType{
				KeyType:   StringType,
				ValueType: &OptionalType{Type: IntType},
			}
			result := checkSubTypeWithoutEquality(dict1, dict2)
			assert.True(t, result, "{String: Int} should be a subtype of {String: Int?}")
		})

		t.Run("optional array", func(t *testing.T) {
			arr := &VariableSizedType{Type: IntType}
			optArr := &OptionalType{
				Type: &VariableSizedType{Type: IntType},
			}
			result := checkSubTypeWithoutEquality(arr, optArr)
			assert.True(t, result, "[Int] should be a subtype of [Int]?")
		})

		t.Run("reference to optional", func(t *testing.T) {
			refToInt := &ReferenceType{
				Type:          IntType,
				Authorization: UnauthorizedAccess,
			}
			refToOptInt := &ReferenceType{
				Type:          &OptionalType{Type: IntType},
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(refToInt, refToOptInt)
			assert.True(t, result, "&Int should be a subtype of &Int?")
		})
	})

	t.Run("SelfReferences", func(t *testing.T) {
		t.Parallel()

		// The function explicitly does NOT check equality,
		// so T <: T should not necessarily return true
		// (caller should check equality separately)

		t.Run("function does not guarantee T <: T returns true", func(t *testing.T) {
			// This is implementation-dependent. The function name suggests
			// it doesn't handle equality case, so we don't assert anything specific here.
			// Just document the behavior.
			_ = checkSubTypeWithoutEquality(IntType, IntType)
			// No assertion - documenting that this case is not guaranteed
		})
	})

	t.Run("ComplexScenarios", func(t *testing.T) {
		t.Parallel()

		t.Run("function returning optional", func(t *testing.T) {
			// fun(): Int  <:  fun(): Int?
			func1 := &FunctionType{
				Purity:               FunctionPurityImpure,
				Parameters:           []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(IntType),
			}
			func2 := &FunctionType{
				Purity:     FunctionPurityImpure,
				Parameters: []Parameter{},
				ReturnTypeAnnotation: NewTypeAnnotation(
					&OptionalType{Type: IntType},
				),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.True(t, result, "fun(): Int should be a subtype of fun(): Int?")
		})

		t.Run("function with optional parameter", func(t *testing.T) {
			// fun(Int?): Void  <:  fun(Int): Void (contravariance)
			func1 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(&OptionalType{Type: IntType})},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
			}
			func2 := &FunctionType{
				Purity: FunctionPurityImpure,
				Parameters: []Parameter{
					{TypeAnnotation: NewTypeAnnotation(IntType)},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
			}
			result := checkSubTypeWithoutEquality(func1, func2)
			assert.True(t, result, "fun(Int?): Void should be a subtype of fun(Int): Void")
		})

		t.Run("dictionary of arrays", func(t *testing.T) {
			dict1 := &DictionaryType{
				KeyType: StringType,
				ValueType: &VariableSizedType{
					Type: IntType,
				},
			}
			dict2 := &DictionaryType{
				KeyType: StringType,
				ValueType: &VariableSizedType{
					Type: NumberType,
				},
			}
			result := checkSubTypeWithoutEquality(dict1, dict2)
			assert.True(t, result, "{String: [Int]} should be a subtype of {String: [Number]}")
		})

		t.Run("constant array covariance with number types", func(t *testing.T) {
			arr1 := &ConstantSizedType{
				Type: Int8Type,
				Size: 3,
			}
			arr2 := &ConstantSizedType{
				Type: SignedIntegerType,
				Size: 3,
			}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.True(t, result, "[Int8; 3] should be a subtype of [SignedInteger; 3]")
		})

		t.Run("array of optionals", func(t *testing.T) {
			// [Int?] <: [Number?]
			arr1 := &VariableSizedType{
				Type: &OptionalType{Type: IntType},
			}
			arr2 := &VariableSizedType{
				Type: &OptionalType{Type: NumberType},
			}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.True(t, result, "[Int?] should be a subtype of [Number?]")
		})

		t.Run("dictionary with nested dictionaries", func(t *testing.T) {
			// {String: {Int: Int}} <: {String: {Number: Number}}
			dict1 := &DictionaryType{
				KeyType: StringType,
				ValueType: &DictionaryType{
					KeyType:   IntType,
					ValueType: IntType,
				},
			}
			dict2 := &DictionaryType{
				KeyType: StringType,
				ValueType: &DictionaryType{
					KeyType:   NumberType,
					ValueType: NumberType,
				},
			}
			result := checkSubTypeWithoutEquality(dict1, dict2)
			assert.True(t, result, "{String: {Int: Int}} should be a subtype of {String: {Number: Number}}")
		})

		t.Run("constant array of constant arrays", func(t *testing.T) {
			// [[Int; 2]; 3] <: [[Number; 2]; 3]
			arr1 := &ConstantSizedType{
				Type: &ConstantSizedType{
					Type: IntType,
					Size: 2,
				},
				Size: 3,
			}
			arr2 := &ConstantSizedType{
				Type: &ConstantSizedType{
					Type: NumberType,
					Size: 2,
				},
				Size: 3,
			}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.True(t, result, "[[Int; 2]; 3] should be subtype of [[Number; 2]; 3]")
		})

		t.Run("mixed variable and constant arrays", func(t *testing.T) {
			// [[Int; 2]] <: [[Number; 2]]
			arr1 := &VariableSizedType{
				Type: &ConstantSizedType{
					Type: IntType,
					Size: 2,
				},
			}
			arr2 := &VariableSizedType{
				Type: &ConstantSizedType{
					Type: NumberType,
					Size: 2,
				},
			}
			result := checkSubTypeWithoutEquality(arr1, arr2)
			assert.True(t, result, "[[Int; 2]] should be subtype of [[Number; 2]]")
		})

		t.Run("dictionary with references", func(t *testing.T) {
			// {String: &Int} <: {String: &Number}
			dict1 := &DictionaryType{
				KeyType: StringType,
				ValueType: &ReferenceType{
					Type:          IntType,
					Authorization: UnauthorizedAccess,
				},
			}
			dict2 := &DictionaryType{
				KeyType: StringType,
				ValueType: &ReferenceType{
					Type:          NumberType,
					Authorization: UnauthorizedAccess,
				},
			}
			result := checkSubTypeWithoutEquality(dict1, dict2)
			assert.True(t, result, "{String: &Int} should be subtype of {String: &Number}")
		})

		t.Run("optional dictionary", func(t *testing.T) {
			// {String: Int}? <: {String: Number}?
			dict1 := &DictionaryType{
				KeyType:   StringType,
				ValueType: IntType,
			}
			dict2 := &DictionaryType{
				KeyType:   StringType,
				ValueType: NumberType,
			}
			opt1 := &OptionalType{Type: dict1}
			opt2 := &OptionalType{Type: dict2}
			result := checkSubTypeWithoutEquality(opt1, opt2)
			assert.True(t, result, "{String: Int}? should be subtype of {String: Number}?")
		})

		t.Run("reference to array", func(t *testing.T) {
			// &[Int] <: &[Number]
			ref1 := &ReferenceType{
				Type:          &VariableSizedType{Type: IntType},
				Authorization: UnauthorizedAccess,
			}
			ref2 := &ReferenceType{
				Type:          &VariableSizedType{Type: NumberType},
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(ref1, ref2)
			assert.True(t, result, "&[Int] should be subtype of &[Number]")
		})

		t.Run("reference to dictionary", func(t *testing.T) {
			// &{String: Int} <: &{String: Number}
			ref1 := &ReferenceType{
				Type: &DictionaryType{
					KeyType:   StringType,
					ValueType: IntType,
				},
				Authorization: UnauthorizedAccess,
			}
			ref2 := &ReferenceType{
				Type: &DictionaryType{
					KeyType:   StringType,
					ValueType: NumberType,
				},
				Authorization: UnauthorizedAccess,
			}
			result := checkSubTypeWithoutEquality(ref1, ref2)
			assert.True(t, result, "&{String: Int} should be subtype of &{String: Number}")
		})
	})
}
