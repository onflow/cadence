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

package analysis

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

var valueDeclarations = append(
	stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{}),
	stdlib.BuiltinFunctions...,
).ToSemaValueDeclarations()

var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

type ParsingCheckingError struct {
	error
	location common.Location
}

func (e ParsingCheckingError) Unwrap() error {
	return e.error
}

func (e ParsingCheckingError) Location() common.Location {
	return e.location
}

func (e ParsingCheckingError) ChildErrors() []error {
	return []error{e.error}
}

// LoadMode controls the amount of detail to return when loading.
// The bits below can be combined to specify what information is required.
//
type LoadMode int

const (
	// NeedSyntax provides the AST.
	NeedSyntax LoadMode = 0

	// NeedTypes provides the elaboration.
	NeedTypes LoadMode = 1 << iota

	// NeedPositionInfo provides position information (e.g. occurrences).
	NeedPositionInfo
)

type Programs map[common.LocationID]*Program

type Program struct {
	Location    common.Location
	Program     *ast.Program
	Elaboration *sema.Elaboration
}

// A Config specifies details about how programs should be loaded.
// The zero value is a valid configuration.
// Calls to Load do not modify this struct.
type Config struct {
	// Mode controls the level of information returned for each program.
	Mode LoadMode

	// ResolveAddressContractNames is called to resolve the contract names of an address location.
	ResolveAddressContractNames func(address common.Address) ([]string, error)

	// ResolveCode is called to resolve an import to its source code.
	ResolveCode func(
		location common.Location,
		importingLocation common.Location,
		importRange ast.Range,
	) (string, error)
}

func Load(config *Config, locations ...common.Location) (Programs, error) {
	programs := make(Programs, len(locations))
	for _, location := range locations {
		err := programs.Load(config, location)
		if err != nil {
			return nil, err
		}
	}

	return programs, nil
}

func (programs Programs) Load(config *Config, location common.Location) error {
	return programs.load(config, location, nil, ast.Range{})
}

func (programs Programs) load(
	config *Config,
	location common.Location,
	importingLocation common.Location,
	importRange ast.Range,
) error {

	if programs[location.ID()] != nil {
		return nil
	}

	wrapError := func(err error) ParsingCheckingError {
		return ParsingCheckingError{
			error:    err,
			location: location,
		}
	}

	code, err := config.ResolveCode(location, importingLocation, importRange)
	if err != nil {
		return err
	}

	program, err := parser2.ParseProgram(code)
	if err != nil {
		return wrapError(err)
	}

	var elaboration *sema.Elaboration
	if config.Mode&NeedTypes != 0 {
		elaboration, err = programs.check(config, program, location)
		if err != nil {
			return wrapError(err)
		}
	}

	programs[location.ID()] = &Program{
		Location:    location,
		Program:     program,
		Elaboration: elaboration,
	}

	return nil
}

func (programs Programs) check(
	config *Config,
	program *ast.Program,
	location common.Location,
) (
	*sema.Elaboration,
	error,
) {
	checker, err := sema.NewChecker(
		program,
		location,
		sema.WithPredeclaredValues(valueDeclarations),
		sema.WithPredeclaredTypes(typeDeclarations),
		sema.WithLocationHandler(
			sema.AddressLocationHandlerFunc(
				config.ResolveAddressContractNames,
			),
		),
		sema.WithPositionInfoEnabled(config.Mode&NeedPositionInfo != 0),
		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {

				var elaboration *sema.Elaboration
				switch importedLocation {
				case stdlib.CryptoChecker.Location:
					elaboration = stdlib.CryptoChecker.Elaboration

				default:
					err := programs.load(config, importedLocation, location, importRange)
					if err != nil {
						return nil, err
					}

					elaboration = programs[importedLocation.ID()].Elaboration
				}

				return sema.ElaborationImport{
					Elaboration: elaboration,
				}, nil
			},
		),
	)
	if err != nil {
		return nil, err
	}

	err = checker.Check()
	if err != nil {
		return nil, err
	}

	return checker.Elaboration, nil
}
