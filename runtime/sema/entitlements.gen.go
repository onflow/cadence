// Code generated from entitlements.cdc. DO NOT EDIT.
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

package sema

var MutateType = &EntitlementType{
	Identifier: "Mutate",
}

var InsertType = &EntitlementType{
	Identifier: "Insert",
}

var RemoveType = &EntitlementType{
	Identifier: "Remove",
}

func init() {
	BuiltinEntitlements[MutateType.Identifier] = MutateType
	BuiltinEntitlements[InsertType.Identifier] = InsertType
	BuiltinEntitlements[RemoveType.Identifier] = RemoveType
}
