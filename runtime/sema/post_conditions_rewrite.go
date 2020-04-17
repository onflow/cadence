package sema

import (
	"github.com/onflow/cadence/runtime/ast"
)

type PostConditionsRewrite struct {
	BeforeStatements        []ast.Statement
	RewrittenPostConditions ast.Conditions
}
