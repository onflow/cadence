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

type Programs map[common.LocationID]*Program

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

	program, err := parser2.ParseProgram(code, nil)
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

	semaPredeclaredValues, _ :=
		stdlib.FlowDefaultPredeclaredValues(stdlib.FlowBuiltinImpls{})

	checker, err := sema.NewChecker(
		program,
		location,
		nil,
		sema.WithPredeclaredValues(semaPredeclaredValues),
		sema.WithPredeclaredTypes(stdlib.FlowDefaultPredeclaredTypes),
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
