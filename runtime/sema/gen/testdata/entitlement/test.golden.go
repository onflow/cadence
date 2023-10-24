// Code generated from testdata/entitlement/test.cdc. DO NOT EDIT.
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

package entitlement

import "github.com/onflow/cadence/runtime/sema"

var FooType = &sema.EntitlementType{
	Identifier: "Foo",
}

var BarType = &sema.EntitlementType{
	Identifier: "Bar",
}

var BazType = &sema.EntitlementMapType{
	Identifier:       "Baz",
	IncludesIdentity: false,
	Relations: []sema.EntitlementRelation{
		sema.EntitlementRelation{
			Input:  FooType,
			Output: BarType,
		},
	},
}

var QuxType = &sema.EntitlementMapType{
	Identifier:       "Qux",
	IncludesIdentity: true,
	Relations: []sema.EntitlementRelation{
		sema.EntitlementRelation{
			Input:  FooType,
			Output: BarType,
		},
	},
}

func init() {
	sema.BuiltinEntitlementMappings[BazType.Identifier] = BazType
	sema.BuiltinEntitlementMappings[QuxType.Identifier] = QuxType
	sema.BuiltinEntitlements[FooType.Identifier] = FooType
	sema.BuiltinEntitlements[BarType.Identifier] = BarType
}
