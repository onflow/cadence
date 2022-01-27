package ipc

const (

	// 'Runtime' method names

	RuntimeMethodExecuteScript          = "executeScript"
	RuntimeMethodExecuteTransaction     = "executeTransaction"
	RuntimeMethodInvokeContractFunction = "invokeContractFunction"

	// 'Interface' method names

	InterfaceMethodGetCode                   = "getCode"
	InterfaceMethodGetProgram                = "getProgram"
	InterfaceMethodResolveLocation           = "resolveLocation"
	InterfaceMethodProgramLog                = "programLog"
	InterfaceMethodGetAccountContractCode    = "getAccountContractCode"
	InterfaceMethodUpdateAccountContractCode = "updateAccountContractCode"
	InterfaceMethodGetValue                  = "getValue"
	InterfaceMethodSetValue                  = "setValue"
	InterfaceMethodAllocateStorageIndex      = "allocateStorageIndex"
)
