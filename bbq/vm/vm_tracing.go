// duplicates some of the interpreter tracing

package vm

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// OnRecordTraceFunc is a function that records a trace.
type OnRecordTraceFunc func(
	vm *VM,
	operationName string,
	duration time.Duration,
	attrs []attribute.KeyValue,
)

const (
	// common
	tracingFunctionPrefix = "function."
	tracingImportPrefix   = "import."

	// type prefixes
	tracingArrayPrefix      = "array."
	tracingDictionaryPrefix = "dictionary."
	tracingCompositePrefix  = "composite."

	// Value operation postfixes
	tracingConstructPostfix            = "construct"
	tracingTransferPostfix             = "transfer"
	tracingConformsToStaticTypePostfix = "conformsToStaticType"
	tracingDeepRemovePostfix           = "deepRemove"
	tracingDestroyPostfix              = "destroy"

	// MemberAccessible operation prefixes
	tracingGetMemberPrefix    = "getMember."
	tracingSetMemberPrefix    = "setMember."
	tracingRemoveMemberPrefix = "removeMember."
)

func (vm *VM) reportTransferTrace(
	targetType string,
	valueType string,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingTransferPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("target type", targetType),
			attribute.String("value type", valueType),
		},
	)
}
