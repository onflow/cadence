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

package common

import (
	"encoding/json"

	"github.com/onflow/cadence/runtime/errors"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=CompositeKind

type CompositeKind uint

const (
	CompositeKindUnknown CompositeKind = iota
	CompositeKindStructure
	CompositeKindResource
	CompositeKindContract
	CompositeKindEvent
	CompositeKindEnum
	CompositeKindAttachment
)

func CompositeKindCount() int {
	return len(_CompositeKind_index) - 1
}

var AllCompositeKinds = []CompositeKind{
	CompositeKindStructure,
	CompositeKindResource,
	CompositeKindContract,
	CompositeKindEvent,
	CompositeKindEnum,
	CompositeKindAttachment,
}

var CompositeKindsWithFieldsAndFunctions = []CompositeKind{
	CompositeKindStructure,
	CompositeKindResource,
	CompositeKindContract,
	CompositeKindAttachment,
}

var InstantiableCompositeKindsWithFieldsAndFunctions = []CompositeKind{
	CompositeKindStructure,
	CompositeKindResource,
	CompositeKindContract,
}

func (k CompositeKind) Name() string {
	switch k {
	case CompositeKindStructure:
		return "structure"
	case CompositeKindResource:
		return "resource"
	case CompositeKindContract:
		return "contract"
	case CompositeKindEvent:
		return "event"
	case CompositeKindEnum:
		return "enum"
	case CompositeKindAttachment:
		return "attachment"
	}

	panic(errors.NewUnreachableError())
}

func (k CompositeKind) Keyword() string {
	switch k {
	case CompositeKindStructure:
		return "struct"
	case CompositeKindResource:
		return "resource"
	case CompositeKindContract:
		return "contract"
	case CompositeKindEvent:
		return "event"
	case CompositeKindEnum:
		return "enum"
	case CompositeKindAttachment:
		return "attachment"
	}

	panic(errors.NewUnreachableError())
}

func (k CompositeKind) DeclarationKind(isInterface bool) DeclarationKind {
	switch k {
	case CompositeKindStructure:
		if isInterface {
			return DeclarationKindStructureInterface
		}
		return DeclarationKindStructure

	case CompositeKindResource:
		if isInterface {
			return DeclarationKindResourceInterface
		}
		return DeclarationKindResource

	case CompositeKindContract:
		if isInterface {
			return DeclarationKindContractInterface
		}
		return DeclarationKindContract

	case CompositeKindEvent:
		if isInterface {
			return DeclarationKindUnknown
		}
		return DeclarationKindEvent

	case CompositeKindEnum:
		if isInterface {
			return DeclarationKindUnknown
		}
		return DeclarationKindEnum
	case CompositeKindAttachment:
		if isInterface {
			return DeclarationKindUnknown
		}
		return DeclarationKindAttachment
	}

	panic(errors.NewUnreachableError())
}

func (k CompositeKind) Annotation() string {
	if k != CompositeKindResource {
		return ""
	}
	return "@"
}

func (k CompositeKind) TransferOperator() string {
	if k != CompositeKindResource {
		return "="
	}
	return "<-"
}

func (k CompositeKind) MoveOperator() string {
	if k != CompositeKindResource {
		return ""
	}
	return "<-"
}

func (k CompositeKind) ConstructionKeyword() string {
	if k != CompositeKindResource {
		return ""
	}
	return "create"
}

func (k CompositeKind) DestructionKeyword() any {
	if k != CompositeKindResource {
		return ""
	}
	return "destroy"
}

func (k CompositeKind) SupportsInterfaces() bool {
	switch k {
	case CompositeKindStructure,
		CompositeKindResource,
		CompositeKindContract:

		return true

	case CompositeKindEvent,
		CompositeKindEnum,
		CompositeKindAttachment:

		return false
	}

	panic(errors.NewUnreachableError())
}

func (k CompositeKind) SupportsAttachments() bool {
	switch k {
	case CompositeKindStructure,
		CompositeKindResource:

		return true

	case CompositeKindEvent,
		CompositeKindEnum,
		CompositeKindContract,
		CompositeKindAttachment:

		return false
	}

	panic(errors.NewUnreachableError())
}

func (k CompositeKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}
