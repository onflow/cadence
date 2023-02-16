/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

// .get<T>
const AccountCapabilitiesTypeGetFunctionName = `get`
const AccountCapabilitiesTypeGetFunctionDocString = `
get returns the capability at the public path, if one was stored there.
`

// .borrow<T>
const AccountCapabilitiesTypeBorrowFunctionName = `borrow`
const AccountCapabilitiesTypeBorrowFunctionDocString = `
borrow gets the capability at the given path, and borrows the capability if it exists.

Returns ` + "nil" + ` if the capability does not exist or cannot be borrowed using the given type.

The function is equivalent to ` + "get(path)?.borrow()`."

// .forEach
const AccountCapabilitiesTypeForEachFunctionName = "forEach"
const AccountCapabilitiesTypeForEachFunctionDocString = `
For each iterates through all the public capabilities of the public account.

Returning false from the function stops the iteration.
`

// .getController
const AuthAccountCapabilitiesTypeGetControllerFunctionName = "getController"
const AuthAccountCapabilitiesTypeGetControllerFunctionDocString = `
Get capability controller for capability with the specified id.

If the id does not reference an existing capability or the capability does not target a storage path on this address, return ` + "`nil`"

// .getControllers
const AuthAccountCapabilitiesTypeGetControllersFunctionName = "getControllers"
const AuthAccountCapabilitiesTypeGetControllersFunctionDocString = `
Get all capability controllers for capabilities that target this storage path
`

// .forEachController
const AuthAccountCapabilitiesTypeForEachControllerFunctionName = "forEachController"
const AuthAccountCapabilitiesTypeForEachControllerFunctionDocString = `
Iterate through all capability controllers for capabilities that target this storage path.

Returning false from the function stops the iteration.
`

// .issue
const AuthAccountCapabilitiesTypeIssueFunctionName = "issue"
const AuthAccountCapabilitiesTypeIssueFunctionDocString = `
Issue/create a new capability.
`

var typeParamT = &TypeParameter{
	Name: "T",
}

var genericTypeT = &GenericType{
	TypeParameter: typeParamT,
}

var AccountCapabilitiesTypeGetFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		typeParamT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(PublicPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &CapabilityType{
				BorrowType: genericTypeT,
			},
		},
	),
}

var AccountCapabilitiesTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		capabilityTypeParameter, // T: &Any
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(PublicPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &GenericType{
				TypeParameter: capabilityTypeParameter,
			},
		},
	),
}

// given args, instantiate a function type (*args) -> Bool
func higherOrderPredicate(params []Parameter) TypeAnnotation {
	return NewTypeAnnotation(
		NewSimpleFunctionType(FunctionPurityImpure, params, NewTypeAnnotation(BoolType)),
	)
}

var AccountCapabilitiesTypeForEachFunctionType = &FunctionType{
	// function: fun(PublicPath, Type): Bool
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: higherOrderPredicate([]Parameter{
				{
					Identifier:     "path",
					TypeAnnotation: NewTypeAnnotation(PublicPathType),
				},
				{
					Identifier:     "capabilityType",
					TypeAnnotation: NewTypeAnnotation(MetaType),
				},
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
}

var AuthAccountCapabilitiesTypeGetControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "byCapabilityID",
			TypeAnnotation: NewTypeAnnotation(UInt64Type),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &ReferenceType{
				Type: CapabilityControllerType,
			},
		},
	),
}

var AuthAccountCapabilitiesTypeGetControllersFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "forPath",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: &ReferenceType{
				Type: CapabilityControllerType,
			},
		},
	),
}

var AuthAccountCapabilitiesTypeForEachControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "forPath",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
		{
			// TODO the predicate argument should not require a label, imo. verify this with Janez and Bastian
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: higherOrderPredicate([]Parameter{
				{
					Identifier: "controller",
					TypeAnnotation: NewTypeAnnotation(&ReferenceType{
						Type: CapabilityControllerType,
					}),
				},
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
}

var AuthAccountCapabilitiesTypeIssueFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{typeParamT},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&CapabilityType{
			BorrowType: genericTypeT,
		},
	),
}
