package common

//go:generate go run golang.org/x/tools/cmd/stringer -type=ComputationKind -trimprefix=ComputationKind

// ComputationKind captures kind of computation that would be used for metring computation
type ComputationKind uint

// [1000,2000) is reserved for Cadence interpreter and runtime
const computationKindRangeStart = 1000

const (
	ComputationKindUnknown ComputationKind = 0
	// interpreter - base
	ComputationKindStatement ComputationKind = computationKindRangeStart + iota
	ComputationKindLoop
	ComputationKindFunctionInvocation
	_
	_
	_
	_
	_
	_
	_
	// interpreter value operations
	ComputationKindCreateCompositeValue
	ComputationKindCopyCompositeValue
	ComputationKindTransferCompositeValue
	ComputationKindDestroyCompositeValue
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
	ComputationKindCreateArrayValue
	ComputationKindCopyArrayValue
	ComputationKindTransferArrayValue
	ComputationKindDestroyArrayValue
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
	ComputationKindCreateDictionaryValue
	ComputationKindCopyDictionaryValue
	ComputationKindTransferDictionaryValue
	ComputationKindDestroyDictionaryValue
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
	_
	_
	_
	// stdlibs computation kinds
	//
	ComputationKindSTDLIBPanic
	ComputationKindSTDLIBAssert
	ComputationKindSTDLIBunsafeRandom
	_
	_
	_
	_
	_
	// RLP
	ComputationKindSTDLIBRLPDecodeString
	ComputationKindSTDLIBRLPDecodeList
	// Crypto
)
