package stdlib

import (
	"fmt"

	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/interpreter"
	"github.com/dapperlabs/cadence/runtime/sema"
)

// StandardLibraryFunction

type StandardLibraryFunction struct {
	Name           string
	Type           sema.InvokableType
	Function       interpreter.HostFunctionValue
	ArgumentLabels []string
}

func (f StandardLibraryFunction) ValueDeclarationType() sema.Type {
	return f.Type
}

func (StandardLibraryFunction) ValueDeclarationKind() common.DeclarationKind {
	return common.DeclarationKindFunction
}

func (StandardLibraryFunction) ValueDeclarationPosition() ast.Position {
	return ast.Position{}
}

func (StandardLibraryFunction) ValueDeclarationIsConstant() bool {
	return true
}

func (f StandardLibraryFunction) ValueDeclarationArgumentLabels() []string {
	return f.ArgumentLabels
}

func NewStandardLibraryFunction(
	name string,
	functionType sema.InvokableType,
	function interpreter.HostFunction,
	argumentLabels []string,
) StandardLibraryFunction {
	functionValue := interpreter.NewHostFunctionValue(function)
	return StandardLibraryFunction{
		Name:           name,
		Type:           functionType,
		Function:       functionValue,
		ArgumentLabels: argumentLabels,
	}
}

// StandardLibraryFunctions

type StandardLibraryFunctions []StandardLibraryFunction

func (functions StandardLibraryFunctions) ToValueDeclarations() map[string]sema.ValueDeclaration {
	valueDeclarations := make(map[string]sema.ValueDeclaration, len(functions))
	for _, function := range functions {
		valueDeclarations[function.Name] = function
	}
	return valueDeclarations
}

func (functions StandardLibraryFunctions) ToValues() map[string]interpreter.Value {
	values := make(map[string]interpreter.Value, len(functions))
	for _, function := range functions {
		values[function.Name] = function.Function
	}
	return values
}

// AssertionError

type AssertionError struct {
	Message string
	interpreter.LocationRange
}

func (e AssertionError) Error() string {
	const message = "assertion failed"
	if e.Message == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", message, e.Message)
}
