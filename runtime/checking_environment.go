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
	"time"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/old_parser"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

// checkingEnvironmentReconfigured is the portion of checkingEnvironment
// that gets reconfigured by checkingEnvironment.Configure
type checkingEnvironmentReconfigured struct {
	runtimeInterface Interface
	codesAndPrograms CodesAndPrograms
}

type checkingEnvironment struct {
	checkingEnvironmentReconfigured

	config *sema.Config

	checkedImports importResolutionResults

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
	env := &checkingEnvironment{
		defaultBaseValueActivation: defaultBaseValueActivation,
		defaultBaseTypeActivation:  defaultBaseTypeActivation,
	}
	env.config = env.newConfig()
	return env
}

func (e *checkingEnvironment) newConfig() *sema.Config {
	return &sema.Config{
		AccessCheckMode:                  sema.AccessCheckModeStrict,
		BaseValueActivationHandler:       e.getBaseValueActivation,
		BaseTypeActivationHandler:        e.getBaseTypeActivation,
		ValidTopLevelDeclarationsHandler: validTopLevelDeclarations,
		LocationHandler:                  e.resolveLocation,
		ImportHandler:                    e.resolveImport,
		CheckHandler:                     newCheckHandler(&e.runtimeInterface),
	}
}

func (e *checkingEnvironment) configure(runtimeInterface Interface, codesAndPrograms CodesAndPrograms) {
	e.runtimeInterface = runtimeInterface
	e.codesAndPrograms = codesAndPrograms
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

func (e *checkingEnvironment) resolveLocation(
	identifiers []Identifier,
	location Location,
) (
	res []ResolvedLocation,
	err error,
) {
	return ResolveLocationWithInterface(
		e.runtimeInterface,
		identifiers,
		location,
	)
}

func (e *checkingEnvironment) resolveImport(
	_ *sema.Checker,
	importedLocation common.Location,
	importRange ast.Range,
) (sema.Import, error) {

	// Check for cyclic imports
	if e.checkedImports[importedLocation] {
		return nil, &sema.CyclicImportsError{
			Location: importedLocation,
			Range:    importRange,
		}
	} else {
		e.checkedImports[importedLocation] = true
		defer delete(e.checkedImports, importedLocation)
	}

	const getAndSetProgram = true
	program, err := e.GetProgram(
		importedLocation,
		getAndSetProgram,
		e.checkedImports,
	)
	if err != nil {
		return nil, err
	}

	return sema.ElaborationImport{
		Elaboration: program.Elaboration,
	}, nil
}

func (e *checkingEnvironment) check(
	location common.Location,
	program *ast.Program,
	checkedImports importResolutionResults,
) (
	elaboration *sema.Elaboration,
	err error,
) {
	e.checkedImports = checkedImports

	checker, err := sema.NewChecker(
		program,
		location,
		e,
		e.config,
	)
	if err != nil {
		return nil, err
	}

	elaboration = checker.Elaboration

	err = checker.Check()
	if err != nil {
		return nil, err
	}

	return elaboration, nil
}

func (e *checkingEnvironment) MeterMemory(usage common.MemoryUsage) error {
	return e.runtimeInterface.MeterMemory(usage)
}

func (e *checkingEnvironment) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	return e.getProgram(
		location,
		func() ([]byte, error) {
			return code, nil
		},
		getAndSetProgram,
		importResolutionResults{
			// Current program is already in check.
			// So mark it also as 'already seen'.
			location: true,
		},
	)
}

// parseAndCheckProgram parses and checks the given program.
func (e *checkingEnvironment) parseAndCheckProgram(
	code []byte,
	location common.Location,
	checkedImports importResolutionResults,
) (
	program *ast.Program,
	elaboration *sema.Elaboration,
	err error,
) {
	wrapParsingCheckingError := func(err error) error {
		switch err.(type) {
		// Wrap only parsing and checking errors.
		case *sema.CheckerError, parser.Error:
			return &ParsingCheckingError{
				Err:      err,
				Location: location,
			}
		default:
			return err
		}
	}

	// Parse

	reportMetric(
		func() {
			program, err = parser.ParseProgram(e, code, parser.Config{})
		},
		e.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramParsed(location, duration)
		},
	)
	if err != nil {
		return nil, nil, wrapParsingCheckingError(err)
	}

	// Check

	elaboration, err = e.check(location, program, checkedImports)
	if err != nil {
		return program, nil, wrapParsingCheckingError(err)
	}

	return program, elaboration, nil
}

func (e *checkingEnvironment) GetProgram(
	location Location,
	storeProgram bool,
	checkedImports importResolutionResults,
) (
	*interpreter.Program,
	error,
) {
	return e.getProgram(
		location,
		func() ([]byte, error) {
			return getLocationCodeFromInterface(e.runtimeInterface, location)
		},
		storeProgram,
		checkedImports,
	)
}

// getProgram returns the existing program at the given location, if available.
// If it is not available, it loads the code, and then parses and checks it.
func (e *checkingEnvironment) getProgram(
	location Location,
	getCode func() ([]byte, error),
	getAndSetProgram bool,
	checkedImports importResolutionResults,
) (
	program *interpreter.Program,
	err error,
) {
	load := func() (*interpreter.Program, error) {
		code, err := getCode()
		if err != nil {
			return nil, err
		}

		e.codesAndPrograms.setCode(location, code)

		parsedProgram, elaboration, err := e.parseAndCheckProgramWithRecovery(
			code,
			location,
			checkedImports,
		)
		if parsedProgram != nil {
			e.codesAndPrograms.setProgram(location, parsedProgram)
		}
		if err != nil {
			return nil, err
		}

		return &interpreter.Program{
			Program:     parsedProgram,
			Elaboration: elaboration,
		}, nil
	}

	if !getAndSetProgram {
		return load()
	}

	errors.WrapPanic(func() {
		program, err = e.runtimeInterface.GetOrLoadProgram(location, func() (program *interpreter.Program, err error) {
			// Loading is done by Cadence.
			// If it panics with a user error, e.g. when parsing fails due to a memory metering error,
			// then do not treat it as an external error (the load callback is called by the embedder)
			panicErr := UserPanicToError(func() {
				program, err = load()
			})
			if panicErr != nil {
				return nil, panicErr
			}

			if err != nil {
				err = interpreter.WrappedExternalError(err)
			}

			return
		})
	})

	return
}

// parseAndCheckProgramWithRecovery parses and checks the given program.
// It first attempts to parse and checks the program as usual.
// If parsing or checking fails, recovery is attempted.
//
// Recovery attempts to parse the contract with the old parser,
// and if it succeeds, uses the program recovery handler
// to produce an elaboration for the old program.
func (e *checkingEnvironment) parseAndCheckProgramWithRecovery(
	code []byte,
	location common.Location,
	checkedImports importResolutionResults,
) (
	program *ast.Program,
	elaboration *sema.Elaboration,
	err error,
) {
	// Attempt to parse and check the program as usual
	program, elaboration, err = e.parseAndCheckProgram(
		code,
		location,
		checkedImports,
	)
	if err == nil {
		return program, elaboration, nil
	}

	// If parsing or checking fails, attempt to recover

	recoveredProgram, recoveredElaboration := e.recoverProgram(
		code,
		location,
		checkedImports,
	)

	// If recovery failed, return the original error
	if recoveredProgram == nil || recoveredElaboration == nil {
		return program, elaboration, err
	}

	recoveredElaboration.IsRecovered = true

	// If recovery succeeded, return the recovered program and elaboration
	return recoveredProgram, recoveredElaboration, nil
}

// recoverProgram parses and checks the given program with the old parser,
// and recovers the elaboration from the old program.
func (e *checkingEnvironment) recoverProgram(
	oldCode []byte,
	location common.Location,
	checkedImports importResolutionResults,
) (
	program *ast.Program,
	elaboration *sema.Elaboration,
) {
	// Parse

	var err error
	reportMetric(
		func() {
			program, err = old_parser.ParseProgram(e, oldCode, old_parser.Config{})
		},
		e.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramParsed(location, duration)
		},
	)
	if err != nil {
		return nil, nil
	}

	// Recover elaboration from the old program

	var newCode []byte
	errors.WrapPanic(func() {
		newCode, err = e.runtimeInterface.RecoverProgram(program, location)
	})
	if err != nil || newCode == nil {
		return nil, nil
	}

	// Parse and check the recovered program

	program, err = parser.ParseProgram(e, newCode, parser.Config{})
	if err != nil {
		return nil, nil
	}

	elaboration, err = e.check(location, program, checkedImports)
	if err != nil || elaboration == nil {
		return nil, nil
	}

	e.codesAndPrograms.setCode(location, newCode)

	return program, elaboration
}

func (e *checkingEnvironment) temporarilyRecordCode(location common.AddressLocation, code []byte) {
	e.codesAndPrograms.setCode(location, code)
}
