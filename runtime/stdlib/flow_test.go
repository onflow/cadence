package stdlib_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestFlowEventTypeIDs(t *testing.T) {
	for _, ty := range []sema.Type{
		stdlib.AccountCreatedEventType,
		stdlib.AccountKeyAddedEventType,
		stdlib.AccountKeyRemovedEventType,
		stdlib.AccountCodeUpdatedEventType,
	} {
		assert.True(t, strings.HasPrefix(string(ty.ID()), "flow"))
	}
}
