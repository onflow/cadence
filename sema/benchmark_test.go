package sema_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/contracts"
)

func BenchmarkFlowTokenContract(b *testing.B) {

	burnerLocation := common.NewStringLocation(nil, "Burner")
	viewResolverLocation := common.NewStringLocation(nil, "ViewResolver")
	fungibleTokenLocation := common.NewStringLocation(nil, "FungibleToken")
	metadataViewsLocation := common.NewStringLocation(nil, "MetadataViews")
	fungibleTokenMetadataViewsLocation := common.NewStringLocation(nil, "FungibleTokenMetadataViews")
	nonFungibleTokenLocation := common.NewStringLocation(nil, "NonFungibleToken")
	flowTokenLocation := common.NewStringLocation(nil, "FlowToken")

	codes := map[common.Location][]byte{
		burnerLocation:                     []byte(contracts.RealBurnerContract),
		viewResolverLocation:               []byte(contracts.RealViewResolverContract),
		fungibleTokenLocation:              []byte(contracts.RealFungibleTokenContract),
		metadataViewsLocation:              []byte(contracts.RealMetadataViewsContract),
		fungibleTokenMetadataViewsLocation: []byte(contracts.RealFungibleTokenMetadataViewsContract),
		nonFungibleTokenLocation:           []byte(contracts.RealNonFungibleTokenContract),
		flowTokenLocation:                  []byte(contracts.RealFlowContract),
	}

	programs := make(map[common.Location]*ast.Program)

	for location, code := range codes {
		program, err := parser.ParseProgram(nil, code, parser.Config{})
		require.NoError(b, err)

		programs[location] = program
	}

	elaborations := make(map[common.Location]*sema.Elaboration)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)

	for _, valueDeclaration := range stdlib.InterpreterDefaultStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	config := &sema.Config{
		AccessCheckMode: sema.AccessCheckModeStrict,
		BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
			return baseValueActivation
		},
	}
	config.ImportHandler = func(
		_ *sema.Checker,
		importedLocation common.Location,
		importRange ast.Range,
	) (sema.Import, error) {

		elaboration, ok := elaborations[importedLocation]
		if ok {
			return sema.ElaborationImport{
				Elaboration: elaboration,
			}, nil
		}

		program, ok := programs[importedLocation]
		if !ok {
			return nil, fmt.Errorf("imported program not found: %s", importedLocation)
		}

		importedChecker, err := sema.NewChecker(program, importedLocation, nil, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create checker for imported program: %w", err)
		}

		err = importedChecker.Check()
		if err != nil {
			return nil, fmt.Errorf("failed to check imported program: %w", err)
		}

		elaboration = importedChecker.Elaboration

		elaborations[importedLocation] = elaboration

		return sema.ElaborationImport{
			Elaboration: elaboration,
		}, nil
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		checker, err := sema.NewChecker(programs[flowTokenLocation], flowTokenLocation, nil, config)
		require.NoError(b, err)

		require.NoError(b, checker.Check())
	}
}
