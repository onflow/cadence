// Code generated from testdata/entitlement.cdc. DO NOT EDIT.
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

var FooType = &EntitlementType{
	Identifier: "Foo",
}

var BarType = &EntitlementType{
	Identifier: "Bar",
}

var BazType = &EntitlementMapType{
	Identifier:       "Baz",
	IncludesIdentity: false,
	Relations: []EntitlementRelation{
		EntitlementRelation{
			Input:  FooType,
			Output: BarType,
		},
	},
}

var QuxType = &EntitlementMapType{
	Identifier:       "Qux",
	IncludesIdentity: true,
	Relations: []EntitlementRelation{
		EntitlementRelation{
			Input:  FooType,
			Output: BarType,
		},
	},
}

func init() {
	BuiltinEntitlementMappings[BazType.Identifier] = BazType
	addToBaseActivation(BazType)
	BuiltinEntitlementMappings[QuxType.Identifier] = QuxType
	addToBaseActivation(QuxType)
	BuiltinEntitlements[FooType.Identifier] = FooType
	addToBaseActivation(FooType)
	BuiltinEntitlements[BarType.Identifier] = BarType
	addToBaseActivation(BarType)
}