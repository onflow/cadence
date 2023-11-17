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

package entitlements

import (
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type EntitlementsMigration struct {
	Interpreter *interpreter.Interpreter
}

var _ migrations.Migration = EntitlementsMigration{}

func NewEntitlementsMigration() EntitlementsMigration {
	return EntitlementsMigration{}
}

func (EntitlementsMigration) Name() string {
	return "AccountTypeMigration"
}

// Converts its input to an entitled type according to the following rules:
// * `ConvertToEntitledType(&T) ---> auth(Entitlements(T)) &T`
// * `ConvertToEntitledType(Capability<T>) ---> Capability<ConvertToEntitledType(T)>`
// * `ConvertToEntitledType(T?) ---> ConvertToEntitledType(T)?
// * `ConvertToEntitledType(T) ---> T`
// where Entitlements(I) is defined as the result of T.SupportedEntitlements()
func ConvertToEntitledType(t sema.Type) sema.Type {
	switch t := t.(type) {
	case *sema.ReferenceType:
		switch t.Authorization {
		case sema.UnauthorizedAccess:
			innerType := ConvertToEntitledType(t.Type)
			auth := sema.UnauthorizedAccess
			if entitlementSupportingType, ok := innerType.(sema.EntitlementSupportingType); ok {
				supportedEntitlements := entitlementSupportingType.SupportedEntitlements()
				if supportedEntitlements.Len() > 0 {
					auth = sema.EntitlementSetAccess{
						SetKind:      sema.Conjunction,
						Entitlements: supportedEntitlements,
					}
				}
			}
			return sema.NewReferenceType(
				nil,
				auth,
				innerType,
			)
		// type is already entitled
		default:
			return t
		}
	case *sema.OptionalType:
		return sema.NewOptionalType(nil, ConvertToEntitledType(t.Type))
	case *sema.CapabilityType:
		return sema.NewCapabilityType(nil, ConvertToEntitledType(t.BorrowType))
	default:
		return t
	}
}

func (EntitlementsMigration) Migrate(value interpreter.Value) (newValue interpreter.Value) {

	return nil
}
