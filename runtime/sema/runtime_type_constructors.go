/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	Name        string
	Value       *FunctionType
	Description string
}

var OptionalTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var VariableSizedArrayTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var ConstantSizedArrayTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "type",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "size",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(MetaType),
}

var DictionaryTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "key",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "value",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

var CompositeTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "identifier",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

var InterfaceTypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "identifier",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&OptionalType{MetaType}),
}

func RuntimeTypeConstructors() []*RuntimeTypeConstructor {
	var functions []*RuntimeTypeConstructor

	functions = append(functions,
		&RuntimeTypeConstructor{
			"OptionalType",
			OptionalTypeFunctionType,
			"Creates a run-time type representing an optional version of the given run-time type",
		})

	functions = append(functions,
		&RuntimeTypeConstructor{
			"VariableSizedArrayType",
			VariableSizedArrayTypeFunctionType,
			"Creates a run-time type representing a variable-sized array type of the given run-time type",
		})

	functions = append(functions,
		&RuntimeTypeConstructor{
			"ConstantSizedArrayType",
			ConstantSizedArrayTypeFunctionType,
			"Creates a run-time type representing a constant-sized array type of the given run-time type with the specifized size",
		})

	functions = append(functions,
		&RuntimeTypeConstructor{
			"DictionaryType",
			DictionaryTypeFunctionType,
			"Creates a run-time type representing a dictionary type of the given run-time key and value types. Returns nil if the key type is not a valid dictionary key",
		})

	functions = append(functions,
		&RuntimeTypeConstructor{
			"CompositeType",
			CompositeTypeFunctionType,
			"Creates a run-time type representing the composite type associated with the given type identifier. Returns nil if the identifier does not correspond to any composite type",
		})

	functions = append(functions,
		&RuntimeTypeConstructor{
			"InterfaceType",
			InterfaceTypeFunctionType,
			"Creates a run-time type representing the interface type associated with the given type identifier. Returns nil if the identifier does not correspond to any interface type",
		})

	return functions
}
