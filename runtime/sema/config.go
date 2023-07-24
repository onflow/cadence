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

type Config struct {
	// ContractValueHandler is used to construct the contract variable
	ContractValueHandler ContractValueHandlerFunc
	// MemberAccountAccessHandler is used to determine if the access of a member with account access modifier is valid
	MemberAccountAccessHandler MemberAccountAccessHandlerFunc
	// ValidTopLevelDeclarationsHandler is used to determine the kinds of declarations
	// which are valid at the top-level for a given location
	ValidTopLevelDeclarationsHandler ValidTopLevelDeclarationsHandlerFunc
	BaseTypeActivation               *VariableActivation
	// ImportHandler is used to resolve unresolved imports
	ImportHandler       ImportHandlerFunc
	BaseValueActivation *VariableActivation
	// CheckHandler is the function which is used for the checking of a program
	CheckHandler CheckHandlerFunc
	// LocationHandler is used to resolve locations
	LocationHandler LocationHandlerFunc
	// AccessCheckMode is the mode for access control checks.
	// It determines how access modifiers how existing and missing acess modifiers are treated
	AccessCheckMode AccessCheckMode
	// ExtendedElaborationEnabled determines if extended elaboration information is generated
	ExtendedElaborationEnabled bool
	// SuggestionsEnabled determines if additional, potentially-computationally intensive,
	// suggested fixes are calculated when producing type errors
	SuggestionsEnabled bool
	// ErrorShortCircuitingEnabled determines if error short-circuiting is enabled.
	// When enabled, the checker will stop running once it encounters an error.
	// When disabled (the default), the checker reports the error then continues checking
	ErrorShortCircuitingEnabled bool
	// PositionInfoEnabled determines if position information is generated.
	// Position info includes origins, occurrences, member accesses, ranges, and function invocations
	PositionInfoEnabled bool
	// AllowNativeDeclarations determines if declarations may be native
	AllowNativeDeclarations bool
	// AllowStaticDeclarations determines if declarations may be static
	AllowStaticDeclarations bool
	// AccountLinkingEnabled determines if account linking is enabled
	AccountLinkingEnabled bool
	// AttachmentsEnabled determines if attachments are enabled
	AttachmentsEnabled bool
	// CapabilityControllersEnabled determines if capability controllers are enabled
	CapabilityControllersEnabled bool
}
