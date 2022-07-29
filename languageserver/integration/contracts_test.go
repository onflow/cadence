package integration

import (
	"testing"

	"github.com/onflow/cadence/languageserver/protocol"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"
)

func Test_ContractUpdate(t *testing.T) {
	const code = `
	  pub contract HelloWorld {
			pub let greeting: String

			pub fun hello(): String {
				return self.greeting
			}

			init(a: String) {
				self.greeting = a
			}
     }
        `
	program, err := parser.ParseProgram(code, nil)
	require.NoError(t, err)

	checker, err := sema.NewChecker(program, common.StringLocation("foo"), nil, false)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	client := &mockFlowClient{}

	t.Run("update contract information", func(t *testing.T) {
		contract := &contractInfo{}
		contract.update("Hello", 1, checker)

		assert.Equal(t, protocol.DocumentURI("Hello"), contract.uri)
		assert.Equal(t, "HelloWorld", contract.name)
		assert.Equal(t, contractTypeDeclaration, contract.kind)

		assert.Len(t, contract.parameters, 1)
		assert.Equal(t, "a", contract.parameters[0].Identifier)
		assert.Equal(t, "", contract.parameters[0].Label)
	})

	t.Run("get codeleneses", func(t *testing.T) {
		contract := &contractInfo{}
		contract.update("Hello", 1, checker)

		client.
			On("GetActiveClientAccount").
			Return(&clientAccount{
				Account: nil,
				Name:    "",
				Active:  false,
			})

		contract.codelens(client)
	})
}
