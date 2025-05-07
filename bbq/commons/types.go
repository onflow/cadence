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

package commons

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

var BuiltinTypes = common.Concat[sema.Type](
	sema.AllBuiltinTypes,
	[]sema.Type{
		&sema.ConstantSizedType{},
		&sema.VariableSizedType{},
		&sema.DictionaryType{},
		&sema.FunctionType{},
		&sema.OptionalType{},

		// TODO: add other types.
	},
)

func TypeQualifiedName(typ sema.Type, functionName string) string {
	if typ == nil {
		return functionName
	}

	typeQualifier := TypeQualifier(typ)
	return typeQualifier + "." + functionName
}

func QualifiedName(typeName, functionName string) string {
	if typeName == "" {
		return functionName
	}

	return typeName + "." + functionName
}

// TypeQualifier returns the prefix to be appended to an identifier
// (e.g: to a function name), to make it type-qualified.
// For primitive types, the type-qualifier is the typeID itself.
// For derived types (e.g: arrays, dictionaries, capabilities, etc.) the type-qualifier
// is a predefined identifier.
// TODO: Add other types
// TODO: Maybe make this a method on the type
func TypeQualifier(typ sema.Type) string {
	switch typ := typ.(type) {
	case *sema.ConstantSizedType:
		return TypeQualifierArrayConstantSized
	case *sema.VariableSizedType:
		return TypeQualifierArrayVariableSized
	case *sema.DictionaryType:
		return TypeQualifierDictionary
	case *sema.FunctionType:
		return TypeQualifierFunction
	case *sema.OptionalType:
		return TypeQualifierOptional
	case *sema.ReferenceType:
		return TypeQualifier(typ.Type)
	case *sema.IntersectionType:
		// TODO: Revisit. Probably this is not needed here?
		return TypeQualifier(typ.Types[0])
	case *sema.CapabilityType:
		return interpreter.PrimitiveStaticTypeCapability.String()
	default:
		return typ.QualifiedString()
	}
}
