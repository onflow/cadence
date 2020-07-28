package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestExportRecursiveType(t *testing.T) {

	t.Parallel()

	ty := &sema.CompositeType{
		Location:   utils.TestLocation,
		Identifier: "Foo",
		Kind:       common.CompositeKindResource,
		Members:    map[string]*sema.Member{},
		Fields:     []string{"foo"},
	}

	ty.Members["foo"] = &sema.Member{
		ContainerType: ty,
		Access:        ast.AccessNotSpecified,
		Identifier:    ast.Identifier{Identifier: "foo"},
		// NOTE: recursive type
		TypeAnnotation:  sema.NewTypeAnnotation(ty),
		DeclarationKind: common.DeclarationKindField,
		VariableKind:    ast.VariableKindVariable,
	}

	expected := &cadence.ResourceType{
		TypeID:     "S.test.Foo",
		Identifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "foo",
			},
		},
	}

	// NOTE: recursion should be kept
	expected.Fields[0].Type = expected

	assert.Equal(t,
		expected,
		exportType(ty, map[sema.TypeID]cadence.Type{}),
	)
}
