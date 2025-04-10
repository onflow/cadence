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

type RuntimeTypeConstructor struct {
	Name      string
	Value     *FunctionType
	DocString string
}

var MetaTypeFunctionType = func() *FunctionType {
	return &FunctionType{
		Purity: FunctionPurityView,
		TypeParameters: []*TypeParameter{
			{Name: "T"},
		},
		ReturnTypeAnnotation: MetaTypeAnnotation,
	}
}()

const OptionalTypeFunctionName = "OptionalType"

var OptionalTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	MetaTypeAnnotation,
)

const VariableSizedArrayTypeFunctionName = "VariableSizedArrayType"

var VariableSizedArrayTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	MetaTypeAnnotation,
)

const ConstantSizedArrayTypeFunctionName = "ConstantSizedArrayType"

var ConstantSizedArrayTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier:     "type",
			TypeAnnotation: MetaTypeAnnotation,
		},
		{
			Identifier:     "size",
			TypeAnnotation: IntTypeAnnotation,
		},
	},
	MetaTypeAnnotation,
)

var OptionalMetaTypeAnnotation = NewTypeAnnotation(&OptionalType{
	Type: MetaType,
})

const DictionaryTypeFunctionName = "DictionaryType"

var DictionaryTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier:     "key",
			TypeAnnotation: MetaTypeAnnotation,
		},
		{
			Identifier:     "value",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	OptionalMetaTypeAnnotation,
)

const CompositeTypeFunctionName = "CompositeType"

var CompositeTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "identifier",
			TypeAnnotation: StringTypeAnnotation,
		},
	},
	OptionalMetaTypeAnnotation,
)

const FunctionTypeFunctionName = "FunctionType"

var FunctionTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier: "parameters",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: MetaType,
				},
			),
		},
		{
			Identifier:     "return",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	MetaTypeAnnotation,
)

const IntersectionTypeFunctionName = "IntersectionType"

var IntersectionTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier: "types",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: StringType,
				},
			),
		},
	},
	OptionalMetaTypeAnnotation,
)

const ReferenceTypeFunctionName = "ReferenceType"

var ReferenceTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier: "entitlements",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: StringType,
				},
			),
		},
		{
			Identifier:     "type",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	OptionalMetaTypeAnnotation,
)

const CapabilityTypeFunctionName = "CapabilityType"

var CapabilityTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	OptionalMetaTypeAnnotation,
)

const InclusiveRangeTypeFunctionName = "InclusiveRangeType"

var InclusiveRangeTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	OptionalMetaTypeAnnotation,
)

var RuntimeTypeConstructors = []*RuntimeTypeConstructor{
	{
		Name:      MetaTypeName,
		Value:     MetaTypeFunctionType,
		DocString: "Creates a run-time type representing the given static type as a value",
	},
	{
		Name:      OptionalTypeFunctionName,
		Value:     OptionalTypeFunctionType,
		DocString: "Creates a run-time type representing an optional version of the given run-time type.",
	},

	{
		Name:      VariableSizedArrayTypeFunctionName,
		Value:     VariableSizedArrayTypeFunctionType,
		DocString: "Creates a run-time type representing a variable-sized array type of the given run-time type.",
	},

	{
		Name:      ConstantSizedArrayTypeFunctionName,
		Value:     ConstantSizedArrayTypeFunctionType,
		DocString: "Creates a run-time type representing a constant-sized array type of the given run-time type with the specified size.",
	},

	{
		Name:  DictionaryTypeFunctionName,
		Value: DictionaryTypeFunctionType,
		DocString: `Creates a run-time type representing a dictionary type of the given run-time key and value types.
		Returns nil if the key type is not a valid dictionary key.`,
	},

	{
		Name:  CompositeTypeFunctionName,
		Value: CompositeTypeFunctionType,
		DocString: `Creates a run-time type representing the composite type associated with the given type identifier.
		Returns nil if the identifier does not correspond to any composite type.`,
	},

	{
		Name:      FunctionTypeFunctionName,
		Value:     FunctionTypeFunctionType,
		DocString: "Creates a run-time type representing a function type associated with the given parameters and return type.",
	},

	{
		Name:  ReferenceTypeFunctionName,
		Value: ReferenceTypeFunctionType,
		DocString: `
		Creates a run-time type representing a reference type of the given type.

		The first argument specifies the set of entitlements to which this reference is entitled.

		Providing an empty array will result in an unauthorized return value.

		Providing invalid entitlements in the input array will result in a nil return value`,
	},

	{
		Name:  IntersectionTypeFunctionName,
		Value: IntersectionTypeFunctionType,
		DocString: `Creates a run-time type representing an intersection of the interface identifiers in the argument.
		Returns nil if the intersection is not valid.`,
	},

	{
		Name:      CapabilityTypeFunctionName,
		Value:     CapabilityTypeFunctionType,
		DocString: "Creates a run-time type representing a capability type of the given reference type. Returns nil if the type is not a reference.",
	},

	{
		Name:  InclusiveRangeTypeFunctionName,
		Value: InclusiveRangeTypeFunctionType,
		DocString: `Creates a run-time type representing an inclusive range type of the given run-time member type. 
		Returns nil if the member type is not a valid inclusive range member type.`,
	},
}
