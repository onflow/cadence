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

package common

import (
	"encoding/json"
	"math"

	"github.com/onflow/cadence/errors"
)

//go:generate stringer -type=DeclarationKind

type DeclarationKind uint

const (
	DeclarationKindUnknown DeclarationKind = iota
	DeclarationKindValue
	DeclarationKindFunction
	DeclarationKindVariable
	DeclarationKindConstant
	DeclarationKindType
	DeclarationKindParameter
	DeclarationKindArgumentLabel
	DeclarationKindStructure
	DeclarationKindResource
	DeclarationKindContract
	DeclarationKindEvent
	DeclarationKindField
	DeclarationKindInitializer
	DeclarationKindDestructorLegacy
	DeclarationKindStructureInterface
	DeclarationKindResourceInterface
	DeclarationKindContractInterface
	DeclarationKindEntitlement
	DeclarationKindEntitlementMapping
	DeclarationKindImport
	DeclarationKindSelf
	DeclarationKindBase
	DeclarationKindTransaction
	DeclarationKindPrepare
	DeclarationKindExecute
	DeclarationKindTypeParameter
	DeclarationKindPragma
	DeclarationKindEnum
	DeclarationKindEnumCase
	DeclarationKindAttachment
)

func DeclarationKindCount() int {
	return len(_DeclarationKind_index) - 1
}

func (k DeclarationKind) IsTypeDeclaration() bool {
	switch k {
	case DeclarationKindStructure,
		DeclarationKindResource,
		DeclarationKindContract,
		DeclarationKindEntitlement,
		DeclarationKindEntitlementMapping,
		DeclarationKindEvent,
		DeclarationKindStructureInterface,
		DeclarationKindResourceInterface,
		DeclarationKindContractInterface,
		DeclarationKindTypeParameter,
		DeclarationKindEnum,
		DeclarationKindAttachment:

		return true

	default:
		return false
	}
}

func (k DeclarationKind) Name() string {
	switch k {
	case DeclarationKindValue:
		return "value"
	case DeclarationKindFunction:
		return "function"
	case DeclarationKindVariable:
		return "variable"
	case DeclarationKindConstant:
		return "constant"
	case DeclarationKindType:
		return "type"
	case DeclarationKindParameter:
		return "parameter"
	case DeclarationKindArgumentLabel:
		return "argument label"
	case DeclarationKindStructure:
		return "structure"
	case DeclarationKindResource:
		return "resource"
	case DeclarationKindContract:
		return "contract"
	case DeclarationKindEvent:
		return "event"
	case DeclarationKindField:
		return "field"
	case DeclarationKindInitializer:
		return "initializer"
	case DeclarationKindDestructorLegacy:
		return "legacy destructor"
	case DeclarationKindAttachment:
		return "attachment"
	case DeclarationKindStructureInterface:
		return "structure interface"
	case DeclarationKindResourceInterface:
		return "resource interface"
	case DeclarationKindContractInterface:
		return "contract interface"
	case DeclarationKindEntitlement:
		return "entitlement"
	case DeclarationKindEntitlementMapping:
		return "entitlement mapping"
	case DeclarationKindImport:
		return "import"
	case DeclarationKindSelf:
		return "self"
	case DeclarationKindBase:
		return "base"
	case DeclarationKindTransaction:
		return "transaction"
	case DeclarationKindPrepare:
		return "prepare"
	case DeclarationKindExecute:
		return "execute"
	case DeclarationKindTypeParameter:
		return "type parameter"
	case DeclarationKindPragma:
		return "pragma"
	case DeclarationKindEnum:
		return "enum"
	case DeclarationKindEnumCase:
		return "enum case"
	case DeclarationKindUnknown:
		return "unknown"
	}

	panic(errors.NewUnreachableError())
}

func (k DeclarationKind) Keywords() string {
	switch k {
	case DeclarationKindFunction:
		return "fun"
	case DeclarationKindVariable:
		return "var"
	case DeclarationKindConstant:
		return "let"
	case DeclarationKindStructure:
		return "struct"
	case DeclarationKindResource:
		return "resource"
	case DeclarationKindContract:
		return "contract"
	case DeclarationKindEvent:
		return "event"
	case DeclarationKindInitializer:
		return "init"
	case DeclarationKindDestructorLegacy: // Deprecated
		return "destroy"
	case DeclarationKindAttachment:
		return "attachment"
	case DeclarationKindStructureInterface:
		return "struct interface"
	case DeclarationKindResourceInterface:
		return "resource interface"
	case DeclarationKindContractInterface:
		return "contract interface"
	case DeclarationKindEntitlement:
		return "entitlement"
	case DeclarationKindEntitlementMapping:
		return "entitlement mapping"
	case DeclarationKindImport:
		return "import"
	case DeclarationKindSelf:
		return "self"
	case DeclarationKindBase:
		return "base"
	case DeclarationKindTransaction:
		return "transaction"
	case DeclarationKindPrepare:
		return "prepare"
	case DeclarationKindExecute:
		return "execute"
	case DeclarationKindEnum:
		return "enum"
	case DeclarationKindEnumCase:
		return "case"
	default:
		return ""
	}
}

func (k DeclarationKind) IsInterfaceDeclaration() bool {
	switch k {
	case DeclarationKindContractInterface,
		DeclarationKindStructureInterface,
		DeclarationKindResourceInterface:
		return true
	}
	return false
}

func (k DeclarationKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

type DeclarationKindSet uint64

const (
	EmptyDeclarationKindSet DeclarationKindSet = 0
	AllDeclarationKindsSet  DeclarationKindSet = math.MaxUint64
)

func NewDeclarationKindSet(declarationKinds ...DeclarationKind) DeclarationKindSet {
	var set DeclarationKindSet
	for _, declarationKind := range declarationKinds {
		set = set.With(declarationKind)
	}
	return set
}

func (s DeclarationKindSet) With(kind DeclarationKind) DeclarationKindSet {
	return s | DeclarationKindSet(1<<kind)
}

func (s DeclarationKindSet) Has(kind DeclarationKind) bool {
	return s&(1<<kind) != 0
}
