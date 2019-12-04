package stdlib

import (
	"github.com/dapperlabs/flow-go/language/runtime/interpreter"
	"github.com/dapperlabs/flow-go/language/runtime/sema"
)

// This file defines functions built in to the Flow runtime.

// TODO: improve types
var createAccountFunctionType = &sema.FunctionType{
	ParameterTypeAnnotations: sema.NewTypeAnnotations(
		// publicKeys
		&sema.VariableSizedType{
			Type: &sema.VariableSizedType{
				Type: &sema.IntType{},
			},
		},
		// code
		&sema.OptionalType{
			Type: &sema.VariableSizedType{
				Type: &sema.IntType{},
			},
		},
	),
	// value
	// TODO: add proper type
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.IntType{},
	),
}

var addAccountKeyFunctionType = &sema.FunctionType{
	ParameterTypeAnnotations: sema.NewTypeAnnotations(
		// address
		&sema.AddressType{},
		// key
		&sema.VariableSizedType{
			Type: &sema.IntType{},
		},
	),
	// nothing
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VoidType{},
	),
}

var removeAccountKeyFunctionType = &sema.FunctionType{
	ParameterTypeAnnotations: sema.NewTypeAnnotations(
		// address
		&sema.AddressType{},
		// index
		&sema.IntType{},
	),
	// nothing
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VoidType{},
	),
}

var updateAccountCodeFunctionType = &sema.FunctionType{
	ParameterTypeAnnotations: sema.NewTypeAnnotations(
		// address
		&sema.AddressType{},
		// code
		&sema.VariableSizedType{
			Type: &sema.IntType{},
		},
	),
	// nothing
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VoidType{},
	),
}

var updateAccountContractFunctionType = &sema.FunctionType{
	ParameterTypeAnnotations: sema.NewTypeAnnotations(
		// address
		&sema.AddressType{},
		// code
		&sema.VariableSizedType{
			Type: &sema.IntType{},
		},
	),
	// nothing
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VoidType{},
	),
	RequiredArgumentCount: (func() *int {
		var count = 2
		return &count
	})(),
}

var getAccountFunctionType = &sema.FunctionType{
	ParameterTypeAnnotations: sema.NewTypeAnnotations(
		// address
		&sema.AddressType{},
	),
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.PublicAccountType{},
	),
}

var logFunctionType = &sema.FunctionType{
	ParameterTypeAnnotations: sema.NewTypeAnnotations(
		&sema.AnyType{},
	),
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VoidType{},
	),
}

// FlowBuiltinImpls defines the set of functions needed to implement the Flow
// built-in functions.
type FlowBuiltinImpls struct {
	CreateAccount         interpreter.HostFunction
	AddAccountKey         interpreter.HostFunction
	RemoveAccountKey      interpreter.HostFunction
	UpdateAccountCode     interpreter.HostFunction
	UpdateAccountContract interpreter.HostFunction
	GetAccount            interpreter.HostFunction
	Log                   interpreter.HostFunction
}

// FlowBuiltInFunctions returns a list of standard library functions, bound to
// the provided implementation.
func FlowBuiltInFunctions(impls FlowBuiltinImpls) StandardLibraryFunctions {
	return StandardLibraryFunctions{
		NewStandardLibraryFunction(
			"createAccount",
			createAccountFunctionType,
			impls.CreateAccount,
			nil,
		),
		NewStandardLibraryFunction(
			"addAccountKey",
			addAccountKeyFunctionType,
			impls.AddAccountKey,
			nil,
		),
		NewStandardLibraryFunction(
			"removeAccountKey",
			removeAccountKeyFunctionType,
			impls.RemoveAccountKey,
			nil,
		),
		NewStandardLibraryFunction(
			"updateAccountCode",
			updateAccountCodeFunctionType,
			impls.UpdateAccountCode,
			nil,
		),
		NewStandardLibraryFunction(
			"updateAccountCode",
			updateAccountContractFunctionType,
			impls.UpdateAccountContract,
			nil,
		),
		NewStandardLibraryFunction(
			"getAccount",
			getAccountFunctionType,
			impls.GetAccount,
			nil,
		),
		NewStandardLibraryFunction(
			"log",
			logFunctionType,
			impls.Log,
			nil,
		),
	}
}
