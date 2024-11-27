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

func prepareArrayAndMapValueTraceAttrs(typeInfo string, count int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int("count", count),
		attribute.String("type", typeInfo),
	}
}

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

func (vm *VM) reportFunctionTrace(
	funcName string,
	argCount int,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingFunctionPrefix+funcName,
		duration,
		[]attribute.KeyValue{
			attribute.Int("count", argCount),
		},
	)
}

func (vm *VM) reportArrayValueConstructTrace(
	typeInfo string,
	count int,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingArrayPrefix+tracingConstructPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (vm *VM) reportCompositeValueConstructTrace(
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
			attribute.String("type", compositeType),
			attribute.String("kind", compositeKind),
		},
	)
}

func (vm *VM) reportCompositeValueSetMemberTrace(
	typeID string,
	kind string,
	name string,
	value string,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingCompositePrefix+tracingSetMemberPrefix+name,
		duration,
		[]attribute.KeyValue{
			attribute.String("typeID", typeID),
			attribute.String("kind", kind),
			attribute.String("name", name),
			attribute.String("value", value),
		},
	)
}

func (vm *VM) reportCompositeValueGetMemberTrace(
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingGetMemberPrefix,
		duration,
		[]attribute.KeyValue{
			attribute.String("typeID", typeID),
			attribute.String("kind", kind),
			attribute.String("name", name),
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
			attribute.String("typeID", typeID),
			attribute.String("kind", kind),
		},
	)
}

func (vm *VM) reportArrayValueTransferTrace(
	typeInfo string,
	count int,
	duration time.Duration,
) {
	config := vm.config
	config.OnRecordTrace(
		vm,
		tracingArrayPrefix+tracingTransferPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}
