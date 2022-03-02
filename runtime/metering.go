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

	// legacy metrics  [40-55)
	OpTypeProgramParsed
	OpTypeProgramChecked
	OpTypeProgramInterpreted
	OpTypeValueDecoded
	OpTypeValueEncoded

	// protocol external methods [45-55)
	OpTypeGetCurrentBlockHeight
	OpTypeGetBlockAtHeight
	OpTypeUnsafeRandom
	_
	_
	_
	_
	_
	_
	_

	// transaction external methods [55-70)
	OpTypeGetSigningAccounts
	OpTypeDecodeArgument
	OpTypeEmitEvent
	OpTypeProgramLog
	OpTypeGenerateUUID

	OpTypeGetCode
	OpTypeResolveLocation
	OpTypeGetProgram
	OpTypeSetProgram
	_
	_
	_
	_
	_
	_

	// account-related external methods [70-110)
	OpTypeGetValue
	OpTypeSetValue
	OpTypeValueExists
	OpTypeAllocateStorageIndex

	OpTypeGetAccountContractNames
	OpTypeGetAccountContractCode
	OpTypeUpdateAccountContractCode
	OpTypeRemoveAccountContractCode

	OpTypeGetAccountKey
	OpTypeAddAccountKey
	OpTypeRevokeAccountKey
	OpTypeAddEncodedAccountKey
	OpTypeRevokeEncodedAccountKey

	OpTypeCreateAccount
	OpTypeGetStorageUsed
	OpTypeGetStorageCapacity
	OpTypeGetAccountBalance
	OpTypeGetAccountAvailableBalance
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

	// crypto external methods [110-125)
	OpTypeVerifySignature
	OpTypeHash
	OpTypeValidatePublicKey
	OpTypeBLSVerifyPOP
	OpTypeBLSAggregateSignatures
	OpTypeBLSAggregatePublicKeys
	_
	_
	_
	_
	_
	_
	_
	_
	_
)
