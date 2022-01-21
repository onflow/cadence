package ipc

const (
	InitInterpreterRuntimeMethod = "initInterpreterRuntime"

	// 'Runtime' method names

	RuntimeMethodExecuteScript          = "executeScript"
	RuntimeMethodExecuteTransaction     = "executeTransaction"
	RuntimeMethodInvokeContractFunction = "invokeContractFunction"

	// 'Interface' method names

	InterfaceMethodGetCode         = "getCode"
	InterfaceMethodGetProgram      = "getProgram"
	InterfaceMethodResolveLocation = "resolveLocation"
)
