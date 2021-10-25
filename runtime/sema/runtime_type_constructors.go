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

	return functions
}
