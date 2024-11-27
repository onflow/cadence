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
	tracingInvokePrefix   = "invoke."

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
			attribute.String("Target type", targetType),
			attribute.String("Value type", valueType),
		},
	)
}

func (vm *VM) reportInvokeTrace(
	funcName string,
	argCount int,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingInvokePrefix,
		duration,
		[]attribute.KeyValue{
			attribute.String("Function name", funcName),
			attribute.Int("Arg count", argCount),
		},
	)
}

func (vm *VM) reportArrayConstructTrace(
	elementType string,
	size int,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingArrayPrefix+tracingConstructPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("Element type", elementType),
			attribute.Int("Size", size),
		},
	)
}

func (vm *VM) reportCompositeConstructTrace(
	compositeType string,
	compositeKind string,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingCompositePrefix+tracingConstructPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("Composite Type", compositeType),
			attribute.String("Composite Kind", compositeKind),
		},
	)
}

func (vm *VM) reportSetMemberTrace(
	fieldName string,
	fieldValue string,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingSetMemberPrefix,
		duration,
		[]attribute.KeyValue{
			attribute.String("Field name", fieldName),
			attribute.String("Field value", fieldValue),
		},
	)
}

func (vm *VM) reportGetMemberTrace(
	fieldName string,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingGetMemberPrefix,
		duration,
		[]attribute.KeyValue{
			attribute.String("Field name", fieldName),
		},
	)
}

func (vm *VM) reportCompositeValueDestroyTrace(
	typeID string,
	kind string,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingCompositePrefix+tracingDestroyPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("TypeID", typeID),
			attribute.String("Kind", kind),
		},
	)
}
