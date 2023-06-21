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

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=AccessCheckMode

type AccessCheckMode uint

const (
	// AccessCheckModeDefault indicates the default access check mode should be used.
	AccessCheckModeDefault AccessCheckMode = iota
	// AccessCheckModeStrict indicates that access modifiers are required
	// and access checks are always enforced.
	AccessCheckModeStrict
	// AccessCheckModeNotSpecifiedRestricted indicates modifiers are optional.
	// Access is assumed private if not specified
	AccessCheckModeNotSpecifiedRestricted
	// AccessCheckModeNotSpecifiedUnrestricted indicates access modifiers are optional.
	// Access is assumed public if not specified
	AccessCheckModeNotSpecifiedUnrestricted
	// AccessCheckModeNone indicates access modifiers are optional and ignored.
	AccessCheckModeNone
)

var AccessCheckModes = []AccessCheckMode{
	AccessCheckModeStrict,
	AccessCheckModeNotSpecifiedRestricted,
	AccessCheckModeNotSpecifiedUnrestricted,
	AccessCheckModeNone,
}

func (mode AccessCheckMode) IsReadableAccess(access Access) bool {
	switch mode {
	case AccessCheckModeStrict,
		AccessCheckModeNotSpecifiedRestricted:

		return access.PermitsAccess(UnauthorizedAccess)

	case AccessCheckModeNotSpecifiedUnrestricted:

		return access == PrimitiveAccess(ast.AccessNotSpecified) ||
			access.PermitsAccess(UnauthorizedAccess)

	case AccessCheckModeNone:
		return true

	default:
		panic(errors.NewUnreachableError())
	}
}

func (mode AccessCheckMode) IsWriteableAccess(access Access) bool {
	switch mode {
	case AccessCheckModeStrict,
		AccessCheckModeNotSpecifiedRestricted:

		return access.PermitsAccess(PrimitiveAccess(ast.AccessPublicSettable))

	case AccessCheckModeNotSpecifiedUnrestricted:

		return access == PrimitiveAccess(ast.AccessNotSpecified) ||
			access.PermitsAccess(PrimitiveAccess(ast.AccessPublicSettable))

	case AccessCheckModeNone:
		return true

	default:
		panic(errors.NewUnreachableError())
	}
}
