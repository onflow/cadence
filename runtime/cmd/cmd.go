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

package cmd

import (
	goerrors "errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func must(err error, location common.Location, codes map[common.Location][]byte) {
	if err == nil {
		return
	}
	printErr := pretty.NewErrorPrettyPrinter(os.Stderr, true).
		PrettyPrintError(err, location, codes)
	if printErr != nil {
		panic(printErr)
	}
	os.Exit(1)
}

func mustClosure(location common.Location, codes map[common.Location][]byte) func(error) {
	return func(e error) {
		must(e, location, codes)
	}
}

func PrepareProgramFromFile(location common.StringLocation, codes map[common.Location][]byte) (*ast.Program, func(error)) {
	code, err := os.ReadFile(string(location))

	program, must := PrepareProgram(code, location, codes)
	must(err)

	return program, must
}

func PrepareProgram(code []byte, location common.Location, codes map[common.Location][]byte) (*ast.Program, func(error)) {
	must := mustClosure(location, codes)

	program, err := parser.ParseProgram(nil, code, parser.Config{})
	codes[location] = code
	must(err)

	return program, must
}

var checkers = map[common.Location]*sema.Checker{}

func DefaultCheckerConfig(
	checkers map[common.Location]*sema.Checker,
	codes map[common.Location][]byte,
	standardLibraryValues []stdlib.StandardLibraryValue,
) *sema.Config {
	// NOTE: also declare values in the interpreter, e.g. for the `REPL` in `NewREPL`
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)

	for _, valueDeclaration := range standardLibraryValues {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	return &sema.Config{
		BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
			return baseValueActivation
		},
		AccessCheckMode: sema.AccessCheckModeStrict,
		ImportHandler: func(
			checker *sema.Checker,
			importedLocation common.Location,
			_ ast.Range,
		) (sema.Import, error) {
			if importedLocation == stdlib.CryptoCheckerLocation {
				cryptoChecker := stdlib.CryptoChecker()
				return sema.ElaborationImport{
					Elaboration: cryptoChecker.Elaboration,
				}, nil
			}

			stringLocation, ok := importedLocation.(common.StringLocation)
			if !ok {
				return nil, &sema.CheckerError{
					Location: checker.Location,
					Codes:    codes,
					Errors: []error{
						fmt.Errorf("cannot import `%s`. only files are supported", importedLocation),
					},
				}
			}

			importedChecker, ok := checkers[importedLocation]
			if !ok {
				importedProgram, _ := PrepareProgramFromFile(stringLocation, codes)
				importedChecker, _ = checker.SubChecker(importedProgram, importedLocation)
				checkers[importedLocation] = importedChecker
			}

			return sema.ElaborationImport{
				Elaboration: importedChecker.Elaboration,
			}, nil
		},
		AttachmentsEnabled: true,
	}
}

// PrepareChecker prepares and initializes a checker with a given code as a string,
// and a filename which is used for pretty-printing errors, if any
func PrepareChecker(
	program *ast.Program,
	location common.Location,
	codes map[common.Location][]byte,
	memberAccountAccess map[common.Location]map[common.Location]struct{},
	standardLibraryValues []stdlib.StandardLibraryValue,
	must func(error),
) (*sema.Checker, func(error)) {

	config := DefaultCheckerConfig(checkers, codes, standardLibraryValues)

	config.MemberAccountAccessHandler = func(checker *sema.Checker, memberLocation common.Location) bool {
		if memberAccountAccess == nil {
			return false
		}

		targets, ok := memberAccountAccess[checker.Location]
		if !ok {
			return false
		}

		_, ok = targets[memberLocation]
		return ok
	}

	checker, err := sema.NewChecker(
		program,
		location,
		nil,
		config,
	)
	must(err)

	return checker, must
}

func PrepareInterpreter(filename string, debugger *interpreter.Debugger) (*interpreter.Interpreter, *sema.Checker, func(error)) {

	codes := map[common.Location][]byte{}

	// do not need to meter this as it's a one-off overhead
	location := common.NewStringLocation(nil, filename)

	program, must := PrepareProgramFromFile(location, codes)

	standardLibraryValues := stdlib.DefaultScriptStandardLibraryValues(
		&StandardLibraryHandler{},
	)

	checker, must := PrepareChecker(
		program,
		location,
		codes,
		nil,
		standardLibraryValues,
		must,
	)

	must(checker.Check())

	var uuid uint64

	storage := interpreter.NewInMemoryStorage(nil)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	for _, value := range standardLibraryValues {
		interpreter.Declare(baseActivation, value)
	}

	config := &interpreter.Config{
		BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
			return baseActivation
		},
		Storage: storage,
		UUIDHandler: func() (uint64, error) {
			defer func() { uuid++ }()
			return uuid, nil
		},
		Debugger: debugger,
		ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			panic("Importing programs is not supported yet")
		},
	}

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		config,
	)
	must(err)

	must(inter.Interpret())

	return inter, checker, must
}

func ExitWithError(message string) {
	println(pretty.FormatErrorMessage(pretty.ErrorPrefix, message, true))
	os.Exit(1)
}

type StandardLibraryHandler struct {
	rand       *rand.Rand
	accountIDs map[common.Address]uint64
}

var _ stdlib.StandardLibraryHandler = &StandardLibraryHandler{}

func (*StandardLibraryHandler) ProgramLog(message string, locationRange interpreter.LocationRange) error {
	fmt.Printf("LOG @ %s: %s\n", formatLocationRange(locationRange), message)
	return nil
}

func (h *StandardLibraryHandler) ReadRandom(p []byte) error {
	if h.rand == nil {
		h.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	h.rand.Read(p)
	return nil
}

func (*StandardLibraryHandler) GetBlockAtHeight(_ uint64) (block stdlib.Block, exists bool, err error) {
	return stdlib.Block{}, false, goerrors.New("blocks are not supported in this environment")
}

func (*StandardLibraryHandler) GetCurrentBlockHeight() (uint64, error) {
	return 0, goerrors.New("blocks are not supported in this environment")
}

func (*StandardLibraryHandler) GetAccountBalance(_ common.Address) (uint64, error) {
	return 0, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) GetAccountAvailableBalance(_ common.Address) (uint64, error) {
	return 0, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) CommitStorageTemporarily(_ *interpreter.Interpreter) error {
	// NO-OP
	return nil
}

func (*StandardLibraryHandler) GetStorageUsed(_ common.Address) (uint64, error) {
	return 0, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) GetStorageCapacity(_ common.Address) (uint64, error) {
	return 0, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) ValidatePublicKey(_ *stdlib.PublicKey) error {
	return goerrors.New("crypto functionality is not available in this environment")
}

func (*StandardLibraryHandler) VerifySignature(
	_ []byte,
	_ string,
	_ []byte,
	_ []byte,
	_ sema.SignatureAlgorithm,
	_ sema.HashAlgorithm,
) (
	bool,
	error,
) {
	return false, goerrors.New("crypto functionality is not available in this environment")
}

func (*StandardLibraryHandler) BLSVerifyPOP(_ *stdlib.PublicKey, _ []byte) (bool, error) {
	return false, goerrors.New("crypto functionality is not available in this environment")
}

func (*StandardLibraryHandler) Hash(_ []byte, _ string, _ sema.HashAlgorithm) ([]byte, error) {
	return nil, goerrors.New("crypto functionality is not available in this environment")
}

func (*StandardLibraryHandler) GetAccountKey(_ common.Address, _ int) (*stdlib.AccountKey, error) {
	return nil, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) AccountKeysCount(_ common.Address) (uint64, error) {
	return 0, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) GetAccountContractNames(_ common.Address) ([]string, error) {
	return nil, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) GetAccountContractCode(_ common.AddressLocation) ([]byte, error) {
	return nil, goerrors.New("accounts are not supported in this environment")
}

func (*StandardLibraryHandler) EmitEvent(
	_ *interpreter.Interpreter,
	_ *sema.CompositeType,
	_ []interpreter.Value,
	_ interpreter.LocationRange,
) {
	// NO-OP, only called for built-in events,
	// which never occurs, as all related functionality producing events is unavailable
}

func (h *StandardLibraryHandler) GenerateAccountID(address common.Address) (uint64, error) {
	if h.accountIDs == nil {
		h.accountIDs = map[common.Address]uint64{}
	}
	h.accountIDs[address]++
	return h.accountIDs[address], nil
}

func (*StandardLibraryHandler) AddAccountKey(
	_ common.Address,
	_ *stdlib.PublicKey,
	_ sema.HashAlgorithm,
	_ int,
) (
	*stdlib.AccountKey,
	error,
) {
	return nil, goerrors.New("accounts are not available in this environment")
}

func (*StandardLibraryHandler) RevokeAccountKey(_ common.Address, _ int) (*stdlib.AccountKey, error) {
	return nil, goerrors.New("accounts are not available in this environment")
}

func (*StandardLibraryHandler) ParseAndCheckProgram(_ []byte, _ common.Location, _ bool) (*interpreter.Program, error) {
	return nil, goerrors.New("nested parsing and checking is not supported in this environment")
}

func (*StandardLibraryHandler) UpdateAccountContractCode(_ common.AddressLocation, _ []byte) error {
	return goerrors.New("accounts are not available in this environment")
}

func (*StandardLibraryHandler) RecordContractUpdate(_ common.AddressLocation, _ *interpreter.CompositeValue) {
	// NO-OP
}

func (h *StandardLibraryHandler) ContractUpdateRecorded(_ common.AddressLocation) bool {
	// NO-OP
	return false
}

func (*StandardLibraryHandler) InterpretContract(
	_ common.AddressLocation,
	_ *interpreter.Program,
	_ string,
	_ stdlib.DeployedContractConstructorInvocation,
) (
	*interpreter.CompositeValue,
	error,
) {
	return nil, goerrors.New("nested interpreting is not available in this environment")
}

func (*StandardLibraryHandler) TemporarilyRecordCode(_ common.AddressLocation, _ []byte) {
	// NO-OP
}

func (*StandardLibraryHandler) RemoveAccountContractCode(_ common.AddressLocation) error {
	return goerrors.New("accounts are not available in this environment")
}

func (*StandardLibraryHandler) RecordContractRemoval(_ common.AddressLocation) {
	// NO-OP
}

func (*StandardLibraryHandler) CreateAccount(_ common.Address) (address common.Address, err error) {
	return common.ZeroAddress, goerrors.New("accounts are not available in this environment")
}

func (*StandardLibraryHandler) BLSAggregatePublicKeys(_ []*stdlib.PublicKey) (*stdlib.PublicKey, error) {
	return nil, goerrors.New("crypto functionality is not available in this environment")
}

func (*StandardLibraryHandler) BLSAggregateSignatures(_ [][]byte) ([]byte, error) {
	return nil, goerrors.New("crypto functionality is not available in this environment")
}

func (h *StandardLibraryHandler) NewOnEventEmittedHandler() interpreter.OnEventEmittedFunc {
	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		event *interpreter.CompositeValue,
		_ *sema.CompositeType,
	) error {
		fmt.Printf(
			"EVENT @ %s: %s\n",
			formatLocationRange(locationRange),
			event.String(),
		)
		return nil
	}
}

func formatLocationRange(locationRange interpreter.LocationRange) string {
	var builder strings.Builder
	if locationRange.Location != nil {
		_, _ = fmt.Fprintf(
			&builder,
			"%s:",
			locationRange.Location,
		)
	}
	startPosition := locationRange.StartPosition()
	_, _ = fmt.Fprintf(
		&builder,
		"%d:%d",
		startPosition.Line,
		startPosition.Column,
	)
	return builder.String()
}
