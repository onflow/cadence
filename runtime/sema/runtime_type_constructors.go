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

package sema

type RuntimeTypeConstructor struct {
	Name      string
	Value     *FunctionType
	DocString string
}

var MetaTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	nil,
	MetaTypeAnnotation,
)

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

var InterfaceTypeFunctionType = NewSimpleFunctionType(
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

var RestrictedTypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier: "identifier",
			TypeAnnotation: NewTypeAnnotation(
				&OptionalType{
					Type: StringType,
				},
			),
		},
		{
			Identifier: "restrictions",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: StringType,
				},
			),
		},
	},
	OptionalMetaTypeAnnotation,
)

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

var runtimeTypeConstructors = []*RuntimeTypeConstructor{
	{
		Name:      "OptionalType",
		Value:     OptionalTypeFunctionType,
		DocString: "Creates a run-time type representing an optional version of the given run-time type.",
	},

	{
		Name:      "VariableSizedArrayType",
		Value:     VariableSizedArrayTypeFunctionType,
		DocString: "Creates a run-time type representing a variable-sized array type of the given run-time type.",
	},

	{
		Name:      "ConstantSizedArrayType",
		Value:     ConstantSizedArrayTypeFunctionType,
		DocString: "Creates a run-time type representing a constant-sized array type of the given run-time type with the specifized size.",
	},

	{
		Name:  "DictionaryType",
		Value: DictionaryTypeFunctionType,
		DocString: `Creates a run-time type representing a dictionary type of the given run-time key and value types.
		Returns nil if the key type is not a valid dictionary key.`,
	},

	{
		Name:  "CompositeType",
		Value: CompositeTypeFunctionType,
		DocString: `Creates a run-time type representing the composite type associated with the given type identifier.
		Returns nil if the identifier does not correspond to any composite type.`,
	},

	{
		Name:  "InterfaceType",
		Value: InterfaceTypeFunctionType,
		DocString: `Creates a run-time type representing the interface type associated with the given type identifier.
		Returns nil if the identifier does not correspond to any interface type.`,
	},

	{
		Name:      "FunctionType",
		Value:     FunctionTypeFunctionType,
		DocString: "Creates a run-time type representing a function type associated with the given parameters and return type.",
	},

	{
		Name:  "ReferenceType",
		Value: ReferenceTypeFunctionType,
		DocString: `Creates a run-time type representing a reference type of the given type. The first argument specifies the set of entitlements to which
		this reference is entitled. Providing an empty array will result in an unauthorized return value. Providing invalid entitlements in the input array
		will result in a nil return value`,
	},

	{
		Name:  "RestrictedType",
		Value: RestrictedTypeFunctionType,
		DocString: `Creates a run-time type representing a restricted type of the first argument, restricted by the interface identifiers in the second argument.
		Returns nil if the restriction is not valid.`,
	},

	{
		Name:      "CapabilityType",
		Value:     CapabilityTypeFunctionType,
		DocString: "Creates a run-time type representing a capability type of the given reference type. Returns nil if the type is not a reference.",
	},
}
