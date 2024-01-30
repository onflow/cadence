/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type LegacyContractUpdateValidator struct {
	TypeComparator

	underlyingUpdateValidator *ContractUpdateValidator
}

// NewContractUpdateValidator initializes and returns a validator, without performing any validation.
// Invoke the `Validate()` method of the validator returned, to start validating the contract.
func NewLegacyContractUpdateValidator(
	location common.Location,
	contractName string,
	oldProgram *ast.Program,
	newProgram *ast.Program,
) *LegacyContractUpdateValidator {

	underlyingValidator := NewContractUpdateValidator(location, contractName, oldProgram, newProgram)

	return &LegacyContractUpdateValidator{
		underlyingUpdateValidator: underlyingValidator,
	}
}

var _ UpdateValidator = &LegacyContractUpdateValidator{}

func (validator *LegacyContractUpdateValidator) getCurrentDeclaration() ast.Declaration {
	return validator.underlyingUpdateValidator.getCurrentDeclaration()
}

func (validator *LegacyContractUpdateValidator) setCurrentDeclaration(decl ast.Declaration) {
	validator.underlyingUpdateValidator.setCurrentDeclaration(decl)
}

// Validate validates the contract update, and returns an error if it is an invalid update.
func (validator *LegacyContractUpdateValidator) Validate() error {
	return validator.underlyingUpdateValidator.Validate()
}

func (validator *LegacyContractUpdateValidator) report(err error) {
	validator.underlyingUpdateValidator.report(err)
}

func (validator *LegacyContractUpdateValidator) checkField(oldField *ast.FieldDeclaration, newField *ast.FieldDeclaration) {
	err := oldField.TypeAnnotation.Type.CheckEqual(newField.TypeAnnotation.Type, validator)
	if err != nil {
		validator.report(&FieldMismatchError{
			DeclName:  validator.getCurrentDeclaration().DeclarationIdentifier().Identifier,
			FieldName: newField.Identifier.Identifier,
			Err:       err,
			Range:     ast.NewUnmeteredRangeFromPositioned(newField.TypeAnnotation),
		})
	}
}
