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

package runtime

import (
	"testing"

	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
)

var location = common.NewStringLocation(nil, "test")

var interfaceType1 = &sema.InterfaceType{
	Location:      location,
	Identifier:    "I1",
	CompositeKind: common.CompositeKindStructure,
	Members:       &sema.StringMemberOrderedMap{},
}

var interfaceType2 = &sema.InterfaceType{
	Location:      location,
	Identifier:    "I2",
	CompositeKind: common.CompositeKindStructure,
	Members:       &sema.StringMemberOrderedMap{},
}

var interfaceType3 = &sema.InterfaceType{
	Location:      location,
	Identifier:    "I3",
	CompositeKind: common.CompositeKindStructure,
	Members:       &sema.StringMemberOrderedMap{},
}

var interfaceTypeWithConformance1 = &sema.InterfaceType{
	Location:                      location,
	Identifier:                    "I4",
	CompositeKind:                 common.CompositeKindStructure,
	Members:                       &sema.StringMemberOrderedMap{},
	ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceType1},
}

var entitlementType = sema.NewEntitlementType(
	nil,
	common.NewStringLocation(nil, "test"),
	"E",
)

var structType1 = &sema.CompositeType{
	Location:   location,
	Identifier: "S1",
	Kind:       common.CompositeKindStructure,
	Members:    &sema.StringMemberOrderedMap{},
}

var structType2 = &sema.CompositeType{
	Location:   location,
	Identifier: "S2",
	Kind:       common.CompositeKindStructure,
	Members:    &sema.StringMemberOrderedMap{},
}

var structTypeWithConformance1 = &sema.CompositeType{
	Location:                      location,
	Identifier:                    "S3",
	Kind:                          common.CompositeKindStructure,
	Members:                       &sema.StringMemberOrderedMap{},
	ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceType1},
}

var structTypeWithConformance2 = &sema.CompositeType{
	Location:                      location,
	Identifier:                    "S4",
	Kind:                          common.CompositeKindStructure,
	Members:                       &sema.StringMemberOrderedMap{},
	ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceType1, interfaceType2},
}

var resourceType1 = &sema.CompositeType{
	Location:   location,
	Identifier: "R1",
	Kind:       common.CompositeKindResource,
	Members:    &sema.StringMemberOrderedMap{},
}

var resourceTypeWithConformance1 = &sema.CompositeType{
	Location:                      location,
	Identifier:                    "R2",
	Kind:                          common.CompositeKindResource,
	Members:                       &sema.StringMemberOrderedMap{},
	ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceType1},
}

var resourceTypeWithConformance2 = &sema.CompositeType{
	Location:                      location,
	Identifier:                    "R3",
	Kind:                          common.CompositeKindResource,
	Members:                       &sema.StringMemberOrderedMap{},
	ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceType1, interfaceType2},
}

// checkSubTypeFunctions calls both checkSubTypeWithoutEquality and checkSubTypeWithoutEquality_gen
// and asserts they produce the same result
func checkSubTypeFunctions(t *testing.T, subType sema.Type, superType sema.Type) bool {
	result := sema.CheckSubTypeWithoutEquality(subType, superType)

	generatedSemaResult := sema.CheckSubTypeWithoutEquality_gen(subType, superType)
	assert.Equal(
		t,
		result,
		generatedSemaResult,
		"generated function in `sema` package produced different results for"+
			" subType=%s, superType=%s: manual=%s, generated=%s",
		subType,
		superType,
		result,
		generatedSemaResult,
	)

	elaboration := sema.NewElaboration(nil)

	elaboration.SetInterfaceType(interfaceType1.ID(), interfaceType1)
	elaboration.SetInterfaceType(interfaceType2.ID(), interfaceType2)
	elaboration.SetInterfaceType(interfaceType3.ID(), interfaceType3)
	elaboration.SetInterfaceType(interfaceTypeWithConformance1.ID(), interfaceTypeWithConformance1)

	elaboration.SetCompositeType(structType1.ID(), structType1)
	elaboration.SetCompositeType(structType2.ID(), structType2)
	elaboration.SetCompositeType(structTypeWithConformance1.ID(), structTypeWithConformance1)
	elaboration.SetCompositeType(structTypeWithConformance2.ID(), structTypeWithConformance2)

	elaboration.SetCompositeType(resourceType1.ID(), resourceType1)
	elaboration.SetCompositeType(resourceTypeWithConformance1.ID(), resourceTypeWithConformance1)
	elaboration.SetCompositeType(resourceTypeWithConformance2.ID(), resourceTypeWithConformance2)

	elaboration.SetEntitlementType(entitlementType.ID(), entitlementType)

	subStaticType := interpreter.ConvertSemaToStaticType(nil, subType)

	superStaticType := interpreter.ConvertSemaToStaticType(nil, superType)

	inter, err := interpreter.NewInterpreter(
		nil,
		common.AddressLocation{},
		&interpreter.Config{
			CompositeTypeHandler: func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
				return elaboration.CompositeType(typeID)
			},
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				return interpreter.VirtualImport{
					Elaboration: elaboration,
				}
			},
		},
	)
	require.NoError(t, err)

	generatedStaticTypeResult := interpreter.CheckSubTypeWithoutEquality_gen(
		inter,
		subStaticType,
		superStaticType,
	)
	assert.Equal(
		t,
		result,
		generatedStaticTypeResult,
		"generated function in `interpreter` package produced different results for"+
			" subType=%s, superType=%s: manual=%s, generated=%s",
		subType,
		superType,
		result,
		generatedStaticTypeResult,
	)

	return result
}

// TestCheckSubTypeWithoutEquality tests all paths of checkSubTypeWithoutEquality function
func TestCheckSubTypeWithoutEquality(t *testing.T) {
	t.Parallel()

	t.Run("sema.NeverType", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			superType sema.Type
		}{
			{"Never <: Any", sema.AnyType},
			{"Never <: AnyStruct", sema.AnyStructType},
			{"Never <: AnyResource", sema.AnyResourceType},
			{"Never <: Int", sema.IntType},
			{"Never <: String", sema.StringType},
			{"Never <: Bool", sema.BoolType},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.NeverType, tt.superType)
				assert.True(t, result, "sema.NeverType should be a subtype of %v", tt.superType)
			})
		}
	})

	t.Run("sema.AnyType", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			subType sema.Type
		}{
			{"Int <: Any", sema.IntType},
			{"String <: Any", sema.StringType},
			{"Bool <: Any", sema.BoolType},
			{"AnyStruct <: Any", sema.AnyStructType},
			{"AnyResource <: Any", sema.AnyResourceType},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := checkSubTypeFunctions(t, tt.subType, sema.AnyType)
				assert.True(t, result, "%v should be a subtype of sema.AnyType", tt.subType)
			})
		}
	})

	t.Run("sema.AnyStructType", func(t *testing.T) {
		t.Parallel()

		t.Run("struct types are subtypes of AnyStruct", func(t *testing.T) {
			tests := []struct {
				name    string
				subType sema.Type
			}{
				{"Int <: AnyStruct", sema.IntType},
				{"String <: AnyStruct", sema.StringType},
				{"Bool <: AnyStruct", sema.BoolType},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.AnyStructType)
					assert.True(t, result, "%v should be a subtype of sema.AnyStructType", tt.subType)
				})
			}
		})

		t.Run("resource types are NOT subtypes of AnyStruct", func(t *testing.T) {
			result := checkSubTypeFunctions(t, sema.AnyResourceType, sema.AnyStructType)
			assert.False(t, result, "AnyResource should NOT be a subtype of AnyStruct")
		})

		t.Run("sema.AnyType is NOT a subtype of AnyStruct", func(t *testing.T) {
			result := checkSubTypeFunctions(t, sema.AnyType, sema.AnyStructType)
			assert.False(t, result, "sema.AnyType should NOT be a subtype of AnyStruct")
		})
	})

	t.Run("sema.AnyResourceType", func(t *testing.T) {
		t.Parallel()

		t.Run("resource types are subtypes of AnyResource", func(t *testing.T) {
			result := checkSubTypeFunctions(t, sema.AnyResourceType, sema.AnyResourceType)
			assert.True(t, result, "AnyResource should be a subtype of AnyResource")
		})

		t.Run("struct types are NOT subtypes of AnyResource", func(t *testing.T) {
			tests := []sema.Type{
				sema.IntType,
				sema.StringType,
				sema.BoolType,
				sema.AnyStructType,
			}

			for _, subType := range tests {
				result := checkSubTypeFunctions(t, subType, sema.AnyResourceType)
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
				result := checkSubTypeFunctions(t, sema.IntType, sema.AnyResourceAttachmentType)
				assert.False(t, result)
			})

			t.Run("struct is not subtype", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.StringType, sema.AnyResourceAttachmentType)
				assert.False(t, result)
			})

			t.Run("AnyStruct is not subtype", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.AnyStructType, sema.AnyResourceAttachmentType)
				assert.False(t, result)
			})
		})

		t.Run("AnyStructAttachment", func(t *testing.T) {
			t.Run("resource is not subtype", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.AnyResourceType, sema.AnyStructAttachmentType)
				assert.False(t, result)
			})

			t.Run("non-attachment struct is not subtype", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.IntType, sema.AnyStructAttachmentType)
				assert.False(t, result)
			})

			t.Run("AnyResource is not subtype", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.AnyResourceType, sema.AnyStructAttachmentType)
				assert.False(t, result)
			})
		})
	})

	t.Run("sema.HashableStructType", func(t *testing.T) {
		t.Parallel()

		t.Run("hashable types are subtypes", func(t *testing.T) {
			tests := []sema.Type{
				sema.IntType,
				sema.StringType,
				sema.BoolType,
				sema.TheAddressType,
			}

			for _, subType := range tests {
				result := checkSubTypeFunctions(t, subType, sema.HashableStructType)
				assert.True(t, result, "%v should be a subtype of HashableStruct", subType)
			}
		})
	})

	t.Run("PathTypes", func(t *testing.T) {
		t.Parallel()

		t.Run("PathType", func(t *testing.T) {
			t.Run("StoragePath <: Path", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.StoragePathType, sema.PathType)
				assert.True(t, result)
			})

			t.Run("PrivatePath <: Path", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.PrivatePathType, sema.PathType)
				assert.True(t, result)
			})

			t.Run("PublicPath <: Path", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.PublicPathType, sema.PathType)
				assert.True(t, result)
			})

			t.Run("Int is NOT <: Path", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.IntType, sema.PathType)
				assert.False(t, result)
			})
		})

		t.Run("CapabilityPathType", func(t *testing.T) {
			t.Run("PrivatePath <: CapabilityPath", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.PrivatePathType, sema.CapabilityPathType)
				assert.True(t, result)
			})

			t.Run("PublicPath <: CapabilityPath", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.PublicPathType, sema.CapabilityPathType)
				assert.True(t, result)
			})

			t.Run("StoragePath is NOT <: CapabilityPath", func(t *testing.T) {
				result := checkSubTypeFunctions(t, sema.StoragePathType, sema.CapabilityPathType)
				assert.False(t, result)
			})
		})
	})

	t.Run("sema.StorableType", func(t *testing.T) {
		t.Parallel()

		t.Run("storable types are subtypes", func(t *testing.T) {
			tests := []sema.Type{
				sema.IntType,
				sema.StringType,
				sema.BoolType,
				sema.TheAddressType,
			}

			for _, subType := range tests {
				result := checkSubTypeFunctions(t, subType, sema.StorableType)
				assert.True(t, result, "%v should be a subtype of Storable", subType)
			}
		})
	})

	t.Run("NumberTypes", func(t *testing.T) {
		t.Parallel()

		t.Run("sema.NumberType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  sema.Type
				expected bool
			}{
				{"sema.NumberType <: Number", sema.NumberType, true},
				{"sema.SignedNumberType <: Number", sema.SignedNumberType, true},
				{"Int <: Number", sema.IntType, true},
				{"Int8 <: Number", sema.Int8Type, true},
				{"UInt <: Number", sema.UIntType, true},
				{"Fix64 <: Number", sema.Fix64Type, true},
				{"UFix64 <: Number", sema.UFix64Type, true},
				{"String is NOT <: Number", sema.StringType, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.NumberType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("sema.SignedNumberType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  sema.Type
				expected bool
			}{
				{"sema.SignedNumberType <: SignedNumber", sema.SignedNumberType, true},
				{"Int <: SignedNumber", sema.IntType, true},
				{"Int8 <: SignedNumber", sema.Int8Type, true},
				{"Fix64 <: SignedNumber", sema.Fix64Type, true},
				{"UInt is NOT <: SignedNumber", sema.UIntType, false},
				{"UFix64 is NOT <: SignedNumber", sema.UFix64Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.SignedNumberType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("sema.IntegerType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  sema.Type
				expected bool
			}{
				{"sema.IntegerType <: Integer", sema.IntegerType, true},
				{"sema.SignedIntegerType <: Integer", sema.SignedIntegerType, true},
				{"sema.FixedSizeUnsignedIntegerType <: Integer", sema.FixedSizeUnsignedIntegerType, true},
				{"sema.UIntType <: Integer", sema.UIntType, true},
				{"Int <: Integer", sema.IntType, true},
				{"UInt8 <: Integer", sema.UInt8Type, true},
				{"Fix64 is NOT <: Integer", sema.Fix64Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.IntegerType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("sema.SignedIntegerType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  sema.Type
				expected bool
			}{
				{"sema.SignedIntegerType <: SignedInteger", sema.SignedIntegerType, true},
				{"Int <: SignedInteger", sema.IntType, true},
				{"Int8 <: SignedInteger", sema.Int8Type, true},
				{"Int16 <: SignedInteger", sema.Int16Type, true},
				{"Int32 <: SignedInteger", sema.Int32Type, true},
				{"Int64 <: SignedInteger", sema.Int64Type, true},
				{"Int128 <: SignedInteger", sema.Int128Type, true},
				{"Int256 <: SignedInteger", sema.Int256Type, true},
				{"UInt is NOT <: SignedInteger", sema.UIntType, false},
				{"UInt8 is NOT <: SignedInteger", sema.UInt8Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.SignedIntegerType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("sema.FixedSizeUnsignedIntegerType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  sema.Type
				expected bool
			}{
				{"UInt8 <: FixedSizeUnsignedInteger", sema.UInt8Type, true},
				{"UInt16 <: FixedSizeUnsignedInteger", sema.UInt16Type, true},
				{"UInt32 <: FixedSizeUnsignedInteger", sema.UInt32Type, true},
				{"UInt64 <: FixedSizeUnsignedInteger", sema.UInt64Type, true},
				{"UInt128 <: FixedSizeUnsignedInteger", sema.UInt128Type, true},
				{"UInt256 <: FixedSizeUnsignedInteger", sema.UInt256Type, true},
				{"Word8 <: FixedSizeUnsignedInteger", sema.Word8Type, true},
				{"Word16 <: FixedSizeUnsignedInteger", sema.Word16Type, true},
				{"Word32 <: FixedSizeUnsignedInteger", sema.Word32Type, true},
				{"Word64 <: FixedSizeUnsignedInteger", sema.Word64Type, true},
				{"Word128 <: FixedSizeUnsignedInteger", sema.Word128Type, true},
				{"Word256 <: FixedSizeUnsignedInteger", sema.Word256Type, true},
				{"UInt is NOT <: FixedSizeUnsignedInteger", sema.UIntType, false},
				{"Int is NOT <: FixedSizeUnsignedInteger", sema.IntType, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.FixedSizeUnsignedIntegerType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("sema.FixedPointType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  sema.Type
				expected bool
			}{
				{"sema.FixedPointType <: FixedPoint", sema.FixedPointType, true},
				{"sema.SignedFixedPointType <: FixedPoint", sema.SignedFixedPointType, true},
				{"UFix64 <: FixedPoint", sema.UFix64Type, true},
				{"UFix128 <: FixedPoint", sema.UFix128Type, true},
				{"Fix64 <: FixedPoint", sema.Fix64Type, true},
				{"Fix128 <: FixedPoint", sema.Fix128Type, true},
				{"Int is NOT <: FixedPoint", sema.IntType, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.FixedPointType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})

		t.Run("sema.SignedFixedPointType", func(t *testing.T) {
			tests := []struct {
				name     string
				subType  sema.Type
				expected bool
			}{
				{"sema.SignedFixedPointType <: SignedFixedPoint", sema.SignedFixedPointType, true},
				{"Fix64 <: SignedFixedPoint", sema.Fix64Type, true},
				{"Fix128 <: SignedFixedPoint", sema.Fix128Type, true},
				{"UFix64 is NOT <: SignedFixedPoint", sema.UFix64Type, false},
				{"UFix128 is NOT <: SignedFixedPoint", sema.UFix128Type, false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := checkSubTypeFunctions(t, tt.subType, sema.SignedFixedPointType)
					assert.Equal(t, tt.expected, result)
				})
			}
		})
	})

	t.Run("sema.OptionalType", func(t *testing.T) {
		t.Parallel()

		t.Run("T <: T?", func(t *testing.T) {
			optionalInt := &sema.OptionalType{Type: sema.IntType}
			result := checkSubTypeFunctions(t, sema.IntType, optionalInt)
			assert.True(t, result, "Int should be a subtype of Int?")
		})

		t.Run("T? <: U? when T <: U", func(t *testing.T) {
			optionalNumber := &sema.OptionalType{Type: sema.NumberType}
			optionalInt := &sema.OptionalType{Type: sema.IntType}
			result := checkSubTypeFunctions(t, optionalInt, optionalNumber)
			assert.True(t, result, "Int? should be a subtype of Number?")
		})

		t.Run("T? is NOT <: U? when T is NOT <: U", func(t *testing.T) {
			optionalInt := &sema.OptionalType{Type: sema.IntType}
			optionalString := &sema.OptionalType{Type: sema.StringType}
			result := checkSubTypeFunctions(t, optionalInt, optionalString)
			assert.False(t, result, "Int? should NOT be a subtype of String?")
		})

		t.Run("nested optionals", func(t *testing.T) {
			// Int <: Int??
			doubleOptionalInt := &sema.OptionalType{
				Type: &sema.OptionalType{Type: sema.IntType},
			}
			result := checkSubTypeFunctions(t, sema.IntType, doubleOptionalInt)
			assert.True(t, result, "Int should be a subtype of Int??")
		})
	})

	t.Run("sema.DictionaryType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in key and value types", func(t *testing.T) {
			dict1 := &sema.DictionaryType{
				KeyType:   sema.IntType,
				ValueType: sema.IntType,
			}
			dict2 := &sema.DictionaryType{
				KeyType:   sema.NumberType,
				ValueType: sema.NumberType,
			}
			result := checkSubTypeFunctions(t, dict1, dict2)
			assert.True(t, result, "{Int: Int} should be a subtype of {Number: Number}")
		})

		t.Run("not subtype when key types don't match", func(t *testing.T) {
			dict1 := &sema.DictionaryType{
				KeyType:   sema.IntType,
				ValueType: sema.IntType,
			}
			dict2 := &sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.IntType,
			}
			result := checkSubTypeFunctions(t, dict1, dict2)
			assert.False(t, result, "{Int: Int} should NOT be a subtype of {String: Int}")
		})

		t.Run("not subtype when value types don't match", func(t *testing.T) {
			dict1 := &sema.DictionaryType{
				KeyType:   sema.IntType,
				ValueType: sema.IntType,
			}
			dict2 := &sema.DictionaryType{
				KeyType:   sema.IntType,
				ValueType: sema.StringType,
			}
			result := checkSubTypeFunctions(t, dict1, dict2)
			assert.False(t, result, "{Int: Int} should NOT be a subtype of {Int: String}")
		})

		t.Run("non-dictionary is not subtype", func(t *testing.T) {
			dict := &sema.DictionaryType{
				KeyType:   sema.IntType,
				ValueType: sema.StringType,
			}
			result := checkSubTypeFunctions(t, sema.IntType, dict)
			assert.False(t, result, "Int should NOT be a subtype of {Int: String}")
		})

		t.Run("dictionary with optional values", func(t *testing.T) {
			dict1 := &sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.IntType,
			}
			dict2 := &sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: &sema.OptionalType{Type: sema.IntType},
			}
			result := checkSubTypeFunctions(t, dict1, dict2)
			assert.True(t, result, "{String: Int} should be a subtype of {String: Int?}")
		})
	})

	t.Run("sema.VariableSizedType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in element type", func(t *testing.T) {
			arr1 := &sema.VariableSizedType{Type: sema.IntType}
			arr2 := &sema.VariableSizedType{Type: sema.NumberType}
			result := checkSubTypeFunctions(t, arr1, arr2)
			assert.True(t, result, "[Int] should be a subtype of [Number]")
		})

		t.Run("not subtype when element types don't match", func(t *testing.T) {
			arr1 := &sema.VariableSizedType{Type: sema.IntType}
			arr2 := &sema.VariableSizedType{Type: sema.StringType}
			result := checkSubTypeFunctions(t, arr1, arr2)
			assert.False(t, result, "[Int] should NOT be a subtype of [String]")
		})

		t.Run("non-array is not subtype", func(t *testing.T) {
			arr := &sema.VariableSizedType{Type: sema.IntType}
			result := checkSubTypeFunctions(t, sema.IntType, arr)
			assert.False(t, result, "Int should NOT be a subtype of [Int]")
		})

		t.Run("nested arrays", func(t *testing.T) {
			// [[Int]] <: [[Number]]
			arrInt := &sema.VariableSizedType{
				Type: &sema.VariableSizedType{Type: sema.IntType},
			}
			arrNumber := &sema.VariableSizedType{
				Type: &sema.VariableSizedType{Type: sema.NumberType},
			}
			result := checkSubTypeFunctions(t, arrInt, arrNumber)
			assert.True(t, result, "[[Int]] should be a subtype of [[Number]]")
		})
	})

	t.Run("sema.ConstantSizedType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in element type with same size", func(t *testing.T) {
			arr1 := &sema.ConstantSizedType{Type: sema.IntType, Size: 5}
			arr2 := &sema.ConstantSizedType{Type: sema.NumberType, Size: 5}
			result := checkSubTypeFunctions(t, arr1, arr2)
			assert.True(t, result, "[Int; 5] should be a subtype of [Number; 5]")
		})

		t.Run("not subtype when sizes differ", func(t *testing.T) {
			arr1 := &sema.ConstantSizedType{Type: sema.IntType, Size: 5}
			arr2 := &sema.ConstantSizedType{Type: sema.IntType, Size: 10}
			result := checkSubTypeFunctions(t, arr1, arr2)
			assert.False(t, result, "[Int; 5] should NOT be a subtype of [Int; 10]")
		})

		t.Run("not subtype when element types don't match", func(t *testing.T) {
			arr1 := &sema.ConstantSizedType{Type: sema.IntType, Size: 5}
			arr2 := &sema.ConstantSizedType{Type: sema.StringType, Size: 5}
			result := checkSubTypeFunctions(t, arr1, arr2)
			assert.False(t, result, "[Int; 5] should NOT be a subtype of [String; 5]")
		})

		t.Run("non-array is not subtype", func(t *testing.T) {
			arr := &sema.ConstantSizedType{Type: sema.IntType, Size: 5}
			result := checkSubTypeFunctions(t, sema.IntType, arr)
			assert.False(t, result, "Int should NOT be a subtype of [Int; 5]")
		})
	})

	t.Run("sema.ReferenceType", func(t *testing.T) {
		t.Parallel()

		t.Run("covariant in referenced type with compatible authorization", func(t *testing.T) {
			ref1 := &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			}
			ref2 := &sema.ReferenceType{
				Type:          sema.NumberType,
				Authorization: sema.UnauthorizedAccess,
			}
			result := checkSubTypeFunctions(t, ref1, ref2)
			assert.True(t, result, "&Int should be a subtype of &Number")
		})

		t.Run("not subtype when authorization doesn't permit", func(t *testing.T) {
			auth := sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{entitlementType},
				sema.Disjunction,
			)

			ref1 := &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			}
			ref2 := &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: auth,
			}
			result := checkSubTypeFunctions(t, ref1, ref2)
			assert.False(t, result, "unauthorized reference should NOT be a subtype of authorized reference")
		})

		t.Run("not subtype when referenced types don't match", func(t *testing.T) {
			ref1 := &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			}
			ref2 := &sema.ReferenceType{
				Type:          sema.StringType,
				Authorization: sema.UnauthorizedAccess,
			}
			result := checkSubTypeFunctions(t, ref1, ref2)
			assert.False(t, result, "&Int should NOT be a subtype of &String")
		})

		t.Run("non-reference is not subtype", func(t *testing.T) {
			ref := &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			}
			result := checkSubTypeFunctions(t, sema.IntType, ref)
			assert.False(t, result, "Int should NOT be a subtype of &Int")
		})

		t.Run("reference to resource type", func(t *testing.T) {
			// &AnyResource <: &AnyResource
			ref1 := &sema.ReferenceType{
				Type:          sema.AnyResourceType,
				Authorization: sema.UnauthorizedAccess,
			}
			ref2 := &sema.ReferenceType{
				Type:          sema.AnyResourceType,
				Authorization: sema.UnauthorizedAccess,
			}
			result := checkSubTypeFunctions(t, ref1, ref2)
			assert.True(t, result, "&AnyResource should be a subtype of &AnyResource")
		})

		t.Run("reference to optional type", func(t *testing.T) {
			// &Int <: &Int?
			refToInt := &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			}
			refToOptInt := &sema.ReferenceType{
				Type:          &sema.OptionalType{Type: sema.IntType},
				Authorization: sema.UnauthorizedAccess,
			}
			result := checkSubTypeFunctions(t, refToInt, refToOptInt)
			assert.True(t, result, "&Int should be a subtype of &Int?")
		})

		t.Run("reference with same authorization different types", func(t *testing.T) {
			// &String is NOT <: &Int even with same auth
			ref1 := &sema.ReferenceType{
				Type:          sema.StringType,
				Authorization: sema.UnauthorizedAccess,
			}
			ref2 := &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			}
			result := checkSubTypeFunctions(t, ref1, ref2)
			assert.False(t, result, "&String should NOT be a subtype of &Int")
		})
	})

	t.Run("sema.FunctionType", func(t *testing.T) {
		t.Parallel()

		t.Run("view function is subtype of impure function", func(t *testing.T) {
			viewFunc := &sema.FunctionType{
				Purity: sema.FunctionPurityView,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			impureFunc := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			result := checkSubTypeFunctions(t, viewFunc, impureFunc)
			assert.True(t, result, "view function should be a subtype of impure function")
		})

		t.Run("impure function is NOT subtype of view function", func(t *testing.T) {
			viewFunc := &sema.FunctionType{
				Purity: sema.FunctionPurityView,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			impureFunc := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			result := checkSubTypeFunctions(t, impureFunc, viewFunc)
			assert.False(t, result, "impure function should NOT be a subtype of view function")
		})

		t.Run("contravariant in parameter types", func(t *testing.T) {
			// fun(Number): Int  <:  fun(Int): Int
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.NumberType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.True(t, result, "fun(Number): Int should be a subtype of fun(Int): Int")
		})

		t.Run("covariant in return type", func(t *testing.T) {
			// fun(Int): Int  <:  fun(Int): Number
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.NumberType),
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.True(t, result, "fun(Int): Int should be a subtype of fun(Int): Number")
		})

		t.Run("not subtype when parameter contravariance fails", func(t *testing.T) {
			// fun(Int): Int is NOT <: fun(Number): Int
			// Because Number is NOT <: Int (contravariance requirement fails)
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.NumberType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.False(t, result, "fun(Int): Int should NOT be subtype of fun(Number): Int (contravariance fails)")
		})

		t.Run("not subtype when parameter arity differs", func(t *testing.T) {
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.False(t, result, "functions with different arities should NOT be subtypes")
		})

		t.Run("not subtype when constructor flags differ", func(t *testing.T) {
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				IsConstructor:        true,
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				IsConstructor:        false,
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.False(t, result, "constructor and non-constructor functions should NOT be subtypes")
		})

		t.Run("non-function is not subtype", func(t *testing.T) {
			fn := &sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			result := checkSubTypeFunctions(t, sema.IntType, fn)
			assert.False(t, result, "Int should NOT be a subtype of function")
		})

		t.Run("function with Void return type", func(t *testing.T) {
			// fun(Int): Void <: fun(Int): Void
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.True(t, result, "functions with Void return should be subtypes")
		})

		t.Run("function with different return types", func(t *testing.T) {
			// fun(Int): String is NOT <: fun(Int): Int
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.False(t, result, "function with String return should NOT be subtype of Int return")
		})

		t.Run("function with different arity", func(t *testing.T) {
			// fun() is NOT <: fun(Int)
			func1 := &sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				Arity:                &sema.Arity{Min: 0, Max: 0},
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(sema.IntType)},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				Arity:                &sema.Arity{Min: 1, Max: 1},
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.False(t, result, "functions with different arity should NOT be subtypes")
		})

		t.Run("function with type parameters not equal", func(t *testing.T) {
			// fun<T: Int>() is NOT <: fun<T: String>()
			typeParam1 := &sema.TypeParameter{
				Name:      "T",
				TypeBound: sema.IntType,
			}
			typeParam2 := &sema.TypeParameter{
				Name:      "T",
				TypeBound: sema.StringType,
			}
			func1 := &sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				TypeParameters:       []*sema.TypeParameter{typeParam1},
			}
			func2 := &sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				TypeParameters:       []*sema.TypeParameter{typeParam2},
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.False(t, result, "functions with different type parameter bounds should NOT be subtypes")
		})

		t.Run("function with different type parameter count", func(t *testing.T) {
			// fun<T>() is NOT <: fun<T, U>()
			typeParam := &sema.TypeParameter{
				Name:      "T",
				TypeBound: sema.IntType,
			}
			func1 := &sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				TypeParameters:       []*sema.TypeParameter{typeParam},
			}
			func2 := &sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				TypeParameters:       []*sema.TypeParameter{typeParam, typeParam},
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.False(t, result, "functions with different type parameter count should NOT be subtypes")
		})

		t.Run("function returning array", func(t *testing.T) {
			// fun(): [Int] <: fun(): [Number]
			func1 := &sema.FunctionType{
				Purity:     sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					&sema.VariableSizedType{Type: sema.IntType},
				),
			}
			func2 := &sema.FunctionType{
				Purity:     sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					&sema.VariableSizedType{Type: sema.NumberType},
				),
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.True(t, result, "fun(): [Int] should be subtype of fun(): [Number]")
		})

		t.Run("function with array parameter", func(t *testing.T) {
			// fun([Number]): Void <: fun([Int]): Void (contravariance)
			func1 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{Type: sema.NumberType})},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			}
			func2 := &sema.FunctionType{
				Purity: sema.FunctionPurityImpure,
				Parameters: []sema.Parameter{
					{TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{Type: sema.IntType})},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			}
			result := checkSubTypeFunctions(t, func1, func2)
			assert.True(t, result, "fun([Number]): Void should be subtype of fun([Int]): Void")
		})
	})

	t.Run("IntersectionType", func(t *testing.T) {
		t.Parallel()

		t.Run("AnyResource intersection with nil subtype", func(t *testing.T) {
			result := checkSubTypeFunctions(t,
				sema.AnyResourceType,
				&sema.IntersectionType{
					LegacyType: sema.AnyResourceType,
					Types:      []*sema.InterfaceType{interfaceType1},
				},
			)
			assert.False(t, result, "AnyResource should NOT be a subtype of AnyResource{I}")
		})

		t.Run("AnyStruct intersection with nil subtype", func(t *testing.T) {
			result := checkSubTypeFunctions(t,
				sema.AnyStructType,
				&sema.IntersectionType{
					LegacyType: sema.AnyStructType,
					Types:      []*sema.InterfaceType{interfaceType1},
				},
			)
			assert.False(t, result, "AnyStruct should NOT be a subtype of AnyStruct{I}")
		})

		t.Run("Any intersection with nil subtype", func(t *testing.T) {
			result := checkSubTypeFunctions(t,
				sema.AnyType,
				&sema.IntersectionType{
					LegacyType: sema.AnyType,
					Types:      []*sema.InterfaceType{interfaceType1},
				},
			)
			assert.False(t, result, "Any should NOT be a subtype of Any{I}")
		})

		// Tests for sema.IntersectionType subtype with nil sema.LegacyType
		t.Run("intersection with nil legacy type as subtype", func(t *testing.T) {
			// {I1, I2} <: {I1} when I1 is subset of {I1, I2}
			subType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType1, interfaceType2},
			}
			superType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.True(t, result, "{I1, I2} should be a subtype of {I1}")
		})

		t.Run("intersection with nil legacy type not subtype when not subset", func(t *testing.T) {
			// {I1} is NOT <: {I2}
			subType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "{I1} should NOT be a subtype of {I2}")
		})

		// Tests for sema.IntersectionType subtype with AnyResource sema.LegacyType
		t.Run("AnyResource intersection subtype with matching interfaces", func(t *testing.T) {
			// AnyResource{I1, I2} <: AnyResource{I1}
			subType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{interfaceType1, interfaceType2},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.True(t, result, "AnyResource{I1, I2} should be a subtype of AnyResource{I1}")
		})

		t.Run("AnyResource intersection not subtype of AnyStruct intersection", func(t *testing.T) {
			subType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "AnyResource{I} should NOT be a subtype of AnyStruct{I}")
		})

		// Tests for sema.IntersectionType subtype with sema.CompositeType sema.LegacyType
		t.Run("composite intersection subtype when interfaces match", func(t *testing.T) {

			// R{I1} <: AnyResource{} when R conforms to I1
			subType := &sema.IntersectionType{
				LegacyType: resourceTypeWithConformance1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.True(t, result, "R{I1} should be a subtype of AnyResource{}")
		})

		t.Run("composite intersection not subtype when composite doesn't conform", func(t *testing.T) {

			// R{I2} is NOT <: AnyResource{I1} when R doesn't conform to I1
			subType := &sema.IntersectionType{
				LegacyType: resourceType1,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "R{I2} should NOT be a subtype of AnyResource{I1} when R doesn't conform")
		})

		// Tests for sema.ConformingType (sema.CompositeType) as subtype of sema.IntersectionType
		t.Run("composite type subtype of intersection when conforming", func(t *testing.T) {

			// S <: AnyStruct{I1} when S conforms to I1
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, structTypeWithConformance1, superType)
			assert.True(t, result, "composite should be a subtype of intersection when conforming")
		})

		t.Run("composite type not subtype of intersection when not conforming", func(t *testing.T) {

			// S is NOT <: AnyStruct{I2} when S doesn't conform to I2
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, structType1, superType)
			assert.False(t, result, "composite should NOT be a subtype of intersection when not conforming")
		})

		t.Run("composite type not subtype when wrong base type", func(t *testing.T) {
			// Resource R is NOT <: AnyStruct{I1} even if conforming
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, resourceTypeWithConformance1, superType)
			assert.False(t, result, "resource should NOT be a subtype of struct intersection")
		})

		// Tests for sema.IntersectionType supertype with non-Any* legacy type
		t.Run("intersection with composite legacy type", func(t *testing.T) {
			// S1{I1} <: S1{I2} when S1 == S1 (owner may restrict/unrestrict)
			subType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.True(t, result, "S1{I1} should be a subtype of S1{I2} when same composite")
		})

		t.Run("intersection with different composite legacy types", func(t *testing.T) {
			// S1{I1} is NOT <: S2{I1} when S1 != S2
			subType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: structType2,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "S1{I1} should NOT be a subtype of S2{I1} when different composites")
		})

		t.Run("intersection with interface legacy not subtype of composite intersection", func(t *testing.T) {
			// I3{I1} is NOT <: S{I2} (interface legacy type vs composite legacy type)
			subType := &sema.IntersectionType{
				LegacyType: interfaceType3,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "I3{I1} should NOT be a subtype of S{I2}")
		})

		t.Run("composite type subtype of composite intersection", func(t *testing.T) {
			// S <: S{I1} (owner may freely restrict)
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, structType1, superType)
			assert.True(t, result, "composite should be a subtype of its own intersection type")
		})

		t.Run("intersection nil legacy type not subtype of composite intersection", func(t *testing.T) {
			// {I1} is NOT <: S{I1} statically
			subType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "{I1} should NOT be statically a subtype of S{I1}")
		})

		t.Run("intersection Any* legacy type not subtype of composite intersection", func(t *testing.T) {
			// AnyStruct{I1} is NOT <: S{I1} statically
			subType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "AnyStruct{I1} should NOT be statically a subtype of S{I1}")
		})

		t.Run("sema.AnyResourceType not subtype of composite intersection", func(t *testing.T) {
			// AnyResource is NOT <: R{I1} statically
			superType := &sema.IntersectionType{
				LegacyType: resourceType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, sema.AnyResourceType, superType)
			assert.False(t, result, "AnyResource should NOT be statically a subtype of R{I1}")
		})

		t.Run("sema.AnyStructType not subtype of composite intersection", func(t *testing.T) {
			// AnyStruct is NOT <: S{I1} statically
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, sema.AnyStructType, superType)
			assert.False(t, result, "AnyStruct should NOT be statically a subtype of S{I1}")
		})

		t.Run("sema.AnyType not subtype of composite intersection", func(t *testing.T) {
			// Any is NOT <: S{I1} statically
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, sema.AnyType, superType)
			assert.False(t, result, "Any should NOT be statically a subtype of S{I1}")
		})

		t.Run("optional type not subtype of composite intersection", func(t *testing.T) {
			// Int? is NOT <: S{I1}
			optionalType := &sema.OptionalType{Type: sema.IntType}
			superType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, optionalType, superType)
			assert.False(t, result, "Int? should NOT be subtype of S{I1}")
		})

		t.Run("struct composite intersection not subtype of AnyResource intersection", func(t *testing.T) {
			// S{I1} is NOT <: AnyResource{I2} because S is not a resource
			subType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "S{I1} should NOT be subtype of AnyResource{I2} (struct vs resource)")
		})

		t.Run("resource composite intersection is subtype of AnyResource intersection with conformance", func(t *testing.T) {
			// R{I1} <: AnyResource{I2} if R is a resource and R conforms to I2
			subType := &sema.IntersectionType{
				LegacyType: resourceTypeWithConformance2,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.True(t, result, "R{I1} should be subtype of AnyResource{I2} when R conforms to I2")
		})

		t.Run("resource composite intersection not subtype of AnyResource without conformance", func(t *testing.T) {
			// R{I1} is NOT <: AnyResource{I2} if R doesn't conform to I2
			subType := &sema.IntersectionType{
				LegacyType: resourceTypeWithConformance1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyResourceType,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "R{I1} should NOT be subtype of AnyResource{I2} when R doesn't conform to I2")
		})

		t.Run("struct composite intersection is subtype of AnyStruct intersection with conformance", func(t *testing.T) {
			// S{I1} <: AnyStruct{I2} if S is a struct and S conforms to I2
			subType := &sema.IntersectionType{
				LegacyType: structTypeWithConformance2,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.True(t, result, "S{I1} should be subtype of AnyStruct{I2} when S conforms to I2")
		})

		t.Run("resource composite intersection not subtype of AnyStruct intersection", func(t *testing.T) {
			// R{I1} is NOT <: AnyStruct{I2} because R is a resource, not a struct
			subType := &sema.IntersectionType{
				LegacyType: resourceType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.False(t, result, "R{I1} should NOT be subtype of AnyStruct{I2} (resource vs struct)")
		})

		t.Run("composite intersection is subtype of sema.AnyType intersection with conformance", func(t *testing.T) {
			// S{I1} <: Any{I2} if S conforms to I2
			subType := &sema.IntersectionType{
				LegacyType: structTypeWithConformance2,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyType,
				Types:      []*sema.InterfaceType{interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, superType)
			assert.True(t, result, "S{I1} should be subtype of Any{I2} when S conforms to I2")
		})

		// Test intersection subtype with sema.IntersectionType interface supertype
		t.Run("intersection type subtype of interface type", func(t *testing.T) {
			// {I1, I2} <: I1 when I1 is in the intersection set
			subType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType1, interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, interfaceType1)
			assert.True(t, result, "{I1, I2} should be a subtype of I1")
		})

		t.Run("intersection type not subtype of interface not in set", func(t *testing.T) {
			// {I1, I2} is NOT <: I3 when I3 is not in the intersection set
			subType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType1, interfaceType2},
			}
			result := checkSubTypeFunctions(t, subType, interfaceType3)
			assert.False(t, result, "{I1, I2} should NOT be a subtype of I3")
		})

		t.Run("parameterized type not subtype of intersection", func(t *testing.T) {
			// Capability<Int> is NOT <: {I1}
			capType := &sema.CapabilityType{
				BorrowType: sema.IntType,
			}
			superType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{},
			}
			result := checkSubTypeFunctions(t, capType, superType)
			assert.False(t, result, "Capability<Int> should NOT be subtype of {I1}")
		})
	})

	t.Run("CompositeType", func(t *testing.T) {
		t.Parallel()

		t.Run("different composite types are not subtypes", func(t *testing.T) {
			result := checkSubTypeFunctions(t, structType1, structType2)
			assert.False(t, result, "different composite types should NOT be subtypes")
		})

		t.Run("non-composite is not subtype of composite", func(t *testing.T) {
			result := checkSubTypeFunctions(t, sema.IntType, structType1)
			assert.False(t, result, "Int should NOT be a subtype of composite")
		})

		// Tests for sema.IntersectionType subtype with sema.CompositeType supertype
		t.Run("intersection with nil legacy type not subtype of composite", func(t *testing.T) {
			// {I} is NOT <: S statically
			subType := &sema.IntersectionType{
				Types: []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, structType1)
			assert.False(t, result, "{I} should NOT be statically a subtype of S")
		})

		t.Run("intersection with Any* legacy type not subtype of composite", func(t *testing.T) {
			// AnyStruct{I} is NOT <: S statically
			subType := &sema.IntersectionType{
				LegacyType: sema.AnyStructType,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, structType1)
			assert.False(t, result, "AnyStruct{I} should NOT be statically a subtype of S")
		})

		t.Run("intersection with matching composite legacy type", func(t *testing.T) {
			// S{I} <: S (owner may freely unrestrict)
			subType := &sema.IntersectionType{
				LegacyType: structType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, structType1)
			assert.True(t, result, "S{I} should be a subtype of S (unrestrict)")
		})

		t.Run("intersection with different composite legacy type", func(t *testing.T) {
			// S1{I} is NOT <: S2 when S1 != S2
			subType := &sema.IntersectionType{
				LegacyType: structType2,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, structType1)
			assert.False(t, result, "S1{I} should NOT be a subtype of S2")
		})

		t.Run("intersection with interface legacy type not subtype of composite", func(t *testing.T) {
			// I{I} is NOT <: S (interface legacy type in intersection)
			subType := &sema.IntersectionType{
				LegacyType: interfaceType1,
				Types:      []*sema.InterfaceType{interfaceType1},
			}
			result := checkSubTypeFunctions(t, subType, structType1)
			assert.False(t, result, "intersection with interface legacy type should NOT be subtype of composite")
		})
	})

	t.Run("InterfaceType", func(t *testing.T) {
		t.Parallel()

		t.Run("composite conforming to interface", func(t *testing.T) {
			result := checkSubTypeFunctions(t, structTypeWithConformance1, interfaceType1)
			assert.True(t, result, "composite conforming to interface should be a subtype")
		})

		t.Run("composite NOT conforming to interface", func(t *testing.T) {
			result := checkSubTypeFunctions(t, structType1, interfaceType1)
			assert.False(t, result, "composite NOT conforming to interface should NOT be a subtype")
		})

		t.Run("composite with wrong kind", func(t *testing.T) {
			result := checkSubTypeFunctions(t, resourceType1, interfaceType1)
			assert.False(t, result, "composite with different kind should NOT be a subtype")
		})

		t.Run("interface subtype of interface", func(t *testing.T) {
			result := checkSubTypeFunctions(t, interfaceTypeWithConformance1, interfaceType1)
			assert.True(t, result, "interface conforming to another interface should be a subtype")
		})

		t.Run("non-conforming type not subtype of interface", func(t *testing.T) {
			// Test fallback to sema.IsParameterizedSubType for types that don't match the specific cases
			// Int is NOT <: sema.InterfaceType
			result := checkSubTypeFunctions(t, sema.IntType, interfaceType1)
			assert.False(t, result, "Int should NOT be a subtype of interface")
		})
	})

	t.Run("ParameterizedType", func(t *testing.T) {
		t.Parallel()

		// This test uses sema.CapabilityType which is a sema.ParameterizedType

		t.Run("capability with matching borrow type", func(t *testing.T) {
			capability1 := &sema.CapabilityType{
				BorrowType: sema.IntType,
			}
			capability2 := &sema.CapabilityType{
				BorrowType: sema.NumberType,
			}

			result := checkSubTypeFunctions(t, capability1, capability2)
			assert.True(t, result, "Capability<Int> should be a subtype of Capability<Number>")
		})

		t.Run("capability with non-matching borrow type", func(t *testing.T) {
			capability1 := &sema.CapabilityType{
				BorrowType: sema.IntType,
			}
			capability2 := &sema.CapabilityType{
				BorrowType: sema.StringType,
			}

			result := checkSubTypeFunctions(t, capability1, capability2)
			assert.False(t, result, "Capability<Int> should NOT be a subtype of Capability<String>")
		})

		t.Run("non-capability is not subtype of capability", func(t *testing.T) {
			capability := &sema.CapabilityType{
				BorrowType: sema.IntType,
			}

			result := checkSubTypeFunctions(t, sema.IntType, capability)
			// This may pass or fail depending on sema.IsParameterizedSubType fallback
			// The function checks if sema.IntType's base type is a subtype of capability
			assert.False(t, result, "Int should NOT be a subtype of Capability<Int>")
		})

		t.Run("parameterized type with nil sema.BaseType", func(t *testing.T) {
			// Capability with nil sema.BorrowType has nil sema.BaseType
			nilCapability := &sema.CapabilityType{
				BorrowType: nil,
			}

			// Int is NOT <: Capability<nil>
			result := checkSubTypeFunctions(t, sema.IntType, nilCapability)
			assert.False(t, result, "Int should NOT be a subtype of Capability with nil sema.BorrowType")
		})

		t.Run("parameterized type with base types not matching", func(t *testing.T) {
			// This tests the case where base types don't match
			capability := &sema.CapabilityType{
				BorrowType: sema.IntType,
			}
			inclusiveRange := &sema.InclusiveRangeType{
				MemberType: sema.IntType,
			}

			// The base types need to be subtypes first
			result := checkSubTypeFunctions(t, capability, inclusiveRange)
			assert.False(t, result, "base types must match for parameterized subtypes")
		})
	})

	t.Run("unhandled type", func(t *testing.T) {
		// sema.TransactionType doesn't match any special case in the switch statement
		// so this falls back to the default-case.
		result := checkSubTypeFunctions(t, sema.IntType, &sema.TransactionType{
			Location: common.TransactionLocation{},
		})
		assert.False(t, result, "Int should NOT be a subtype of sema.TransactionType")
	})
}
