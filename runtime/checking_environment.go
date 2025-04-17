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

package runtime

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

type checkingEnvironment struct {
	// defaultBaseTypeActivation is the base type activation that applies to all locations by default.
	defaultBaseTypeActivation *sema.VariableActivation
	// The base type activations for individual locations.
	// location == nil is the base type activation that applies to all locations,
	// unless there is a base type activation for the given location.
	//
	// Base type activations are lazily / implicitly created
	// by DeclareType / semaBaseActivationFor
	baseTypeActivationsByLocation map[common.Location]*sema.VariableActivation

	// defaultBaseValueActivation is the base value activation that applies to all locations by default.
	defaultBaseValueActivation *sema.VariableActivation
	// The base value activations for individual locations.
	// location == nil is the base value activation that applies to all locations,
	// unless there is a base value activation for the given location.
	//
	// Base value activations are lazily / implicitly created
	// by DeclareValue / semaBaseActivationFor
	baseValueActivationsByLocation map[common.Location]*sema.VariableActivation
}

func newCheckingEnvironment() *checkingEnvironment {
	defaultBaseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	defaultBaseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
	return &checkingEnvironment{
		defaultBaseValueActivation: defaultBaseValueActivation,
		defaultBaseTypeActivation:  defaultBaseTypeActivation,
	}
}

// getBaseValueActivation returns the base activation for the given location.
// If a value was declared for the location (using DeclareValue),
// then the specific base value activation for this location is returned.
// Otherwise, the default base activation that applies for all locations is returned.
func (e *checkingEnvironment) getBaseValueActivation(
	location common.Location,
) (
	baseValueActivation *sema.VariableActivation,
) {
	baseValueActivationsByLocation := e.baseValueActivationsByLocation
	// Use the base value activation for the location, if any
	// (previously implicitly created using DeclareValue)
	baseValueActivation = baseValueActivationsByLocation[location]
	if baseValueActivation == nil {
		// If no base value activation for the location exists
		// (no value was previously, specifically declared for the location using DeclareValue),
		// return the base value activation that applies to all locations by default
		baseValueActivation = e.defaultBaseValueActivation
	}
	return

}

// getBaseTypeActivation returns the base activation for the given location.
// If a type was declared for the location (using DeclareType),
// then the specific base type activation for this location is returned.
// Otherwise, the default base activation that applies for all locations is returned.
func (e *checkingEnvironment) getBaseTypeActivation(
	location common.Location,
) (
	baseTypeActivation *sema.VariableActivation,
) {
	// Use the base type activation for the location, if any
	// (previously implicitly created using DeclareType)
	baseTypeActivationsByLocation := e.baseTypeActivationsByLocation
	baseTypeActivation = baseTypeActivationsByLocation[location]
	if baseTypeActivation == nil {
		// If no base type activation for the location exists
		// (no type was previously, specifically declared for the location using DeclareType),
		// return the base type activation that applies to all locations by default
		baseTypeActivation = e.defaultBaseTypeActivation
	}
	return
}

func (e *checkingEnvironment) semaBaseActivationFor(
	location common.Location,
	baseActivationsByLocation *map[Location]*sema.VariableActivation,
	defaultBaseActivation *sema.VariableActivation,
) (baseActivation *sema.VariableActivation) {
	if location == nil {
		return defaultBaseActivation
	}

	if *baseActivationsByLocation == nil {
		*baseActivationsByLocation = map[Location]*sema.VariableActivation{}
	} else {
		baseActivation = (*baseActivationsByLocation)[location]
	}
	if baseActivation == nil {
		baseActivation = sema.NewVariableActivation(defaultBaseActivation)
		(*baseActivationsByLocation)[location] = baseActivation
	}
	return baseActivation
}

func (e *checkingEnvironment) declareValue(valueDeclaration stdlib.StandardLibraryValue, location common.Location) {
	e.semaBaseActivationFor(
		location,
		&e.baseValueActivationsByLocation,
		e.defaultBaseValueActivation,
	).DeclareValue(valueDeclaration)
}

func (e *checkingEnvironment) declareType(typeDeclaration stdlib.StandardLibraryType, location common.Location) {
	e.semaBaseActivationFor(
		location,
		&e.baseTypeActivationsByLocation,
		e.defaultBaseTypeActivation,
	).DeclareType(typeDeclaration)
}
