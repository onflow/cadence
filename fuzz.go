package cadence

import (
	"unicode/utf8"

	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/parser"
	"github.com/dapperlabs/cadence/runtime/sema"
)

func Fuzz(data []byte) int {

	if !utf8.Valid(data) {
		return 0
	}

	program, _, err := parser.ParseProgram(string(data))

	if err != nil {
		return 0
	}

	checker, err := sema.NewChecker(
		program,
		ast.StringLocation("test"),
		sema.WithAccessCheckMode(sema.AccessCheckModeNotSpecifiedUnrestricted),
	)
	if err != nil {
		return 0
	}

	err = checker.Check()
	if err != nil {
		return 0
	}

	return 1
}
