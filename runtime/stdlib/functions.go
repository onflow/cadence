/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package stdlib

import (
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// StandardLibraryFunction

type StandardLibraryFunction struct {
	Name           string
	Type           *sema.FunctionType
	DocString      string
	Function       *interpreter.HostFunctionValue
	ArgumentLabels []string
	Available      func(common.Location) bool
}

func (f StandardLibraryFunction) ValueDeclarationName() string {
	return f.Name
}

func (f StandardLibraryFunction) ValueDeclarationValue(_ *interpreter.Interpreter) interpreter.Value {
	return f.Function
}

func (f StandardLibraryFunction) ValueDeclarationType() sema.Type {
	return f.Type
}

func (f StandardLibraryFunction) ValueDeclarationDocString() string {
	return f.DocString
}

func (StandardLibraryFunction) ValueDeclarationKind() common.DeclarationKind {
	return common.DeclarationKindFunction
}

func (StandardLibraryFunction) ValueDeclarationPosition() ast.Position {
	return ast.EmptyPosition
}

func (StandardLibraryFunction) ValueDeclarationIsConstant() bool {
	return true
}

func (f StandardLibraryFunction) ValueDeclarationAvailable(location common.Location) bool {
	if f.Available == nil {
		return true
	}
	return f.Available(location)
}

func (f StandardLibraryFunction) ValueDeclarationArgumentLabels() []string {
	return f.ArgumentLabels
}

func NewStandardLibraryFunction(
	name string,
	functionType *sema.FunctionType,
	docString string,
	function interpreter.HostFunction,
) StandardLibraryFunction {

	parameters := functionType.Parameters

	argumentLabels := make([]string, len(parameters))

	for i, parameter := range parameters {
		argumentLabels[i] = parameter.EffectiveArgumentLabel()
	}

	functionValue := interpreter.NewUnmeteredHostFunctionValue(function, functionType)

	return StandardLibraryFunction{
		Name:           name,
		Type:           functionType,
		DocString:      docString,
		Function:       functionValue,
		ArgumentLabels: argumentLabels,
	}
}

// StandardLibraryFunctions

type StandardLibraryFunctions []StandardLibraryFunction

func (functions StandardLibraryFunctions) ToSemaValueDeclarations() []sema.ValueDeclaration {
	valueDeclarations := make([]sema.ValueDeclaration, len(functions))
	for i, function := range functions {
		valueDeclarations[i] = function
	}
	return valueDeclarations
}

func (functions StandardLibraryFunctions) ToInterpreterValueDeclarations() []interpreter.ValueDeclaration {
	valueDeclarations := make([]interpreter.ValueDeclaration, len(functions))
	for i, function := range functions {
		valueDeclarations[i] = function
	}
	return valueDeclarations
}

// AssertionError

type AssertionError struct {
	Message string
	interpreter.LocationRange
}

var _ errors.UserError = AssertionError{}

func (AssertionError) IsUserError() {}

func (e AssertionError) Error() string {
	const message = "assertion failed"
	if e.Message == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", message, e.Message)
}
