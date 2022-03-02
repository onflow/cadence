package runtime

type MetringOperationType uint

const (
	// base [0-10)
	OpTypeStatement MetringOperationType = iota
	OpTypeLoop
	OpTypeFunctionInvocation
	_
	_
	_
	_
	_
	_
	_

	// value operations [10-40)
	OpTypeCompositeValueCreate
	OpTypeCompositeValueCopy
	OpTypeCompositeValueTransfer
	OpTypeCompositeValueDestroy
	OpTypeArrayValueCreate
	OpTypeArrayValueCopy
	OpTypeArrayValueTransfer
	OpTypeArrayValueDestroy
	OpTypeDictionaryValueCreate
	OpTypeDictionaryValueCopy
	OpTypeDictionaryValueTransfer
	OpTypeDictionaryValueDestroy
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// legacy metrics  [40-45)
	OpTypeProgramParsed
	OpTypeProgramChecked
	OpTypeProgramInterpreted
	OpTypeValueDecoded
	OpTypeValueEncoded

	// stdlib metrics
	// TODO figure out how to pass those here
)
