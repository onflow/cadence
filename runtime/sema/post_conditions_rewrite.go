package sema

import (
	"github.com/dapperlabs/cadence/runtime/ast"
)

type PostConditionsRewrite struct {
	BeforeStatements        []ast.Statement
	RewrittenPostConditions ast.Conditions
}
