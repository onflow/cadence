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
	"github.com/onflow/cadence/runtime/sema"
)

type LegacyContractUpdateValidator struct {
	TypeComparator

	newElaboration *sema.Elaboration

	underlyingUpdateValidator *ContractUpdateValidator
}

// NewContractUpdateValidator initializes and returns a validator, without performing any validation.
// Invoke the `Validate()` method of the validator returned, to start validating the contract.
func NewLegacyContractUpdateValidator(
	location common.Location,
	contractName string,
	oldProgram *ast.Program,
	newProgram *ast.Program,
	newElaboration *sema.Elaboration,
) *LegacyContractUpdateValidator {

	underlyingValidator := NewContractUpdateValidator(location, contractName, oldProgram, newProgram)

	return &LegacyContractUpdateValidator{
		underlyingUpdateValidator: underlyingValidator,
		newElaboration:            newElaboration,
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
	underlyingValidator := validator.underlyingUpdateValidator

	oldRootDecl := getRootDeclaration(validator, underlyingValidator.oldProgram)
	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	newRootDecl := getRootDeclaration(validator, underlyingValidator.newProgram)
	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	validator.TypeComparator.RootDeclIdentifier = newRootDecl.DeclarationIdentifier()

	checkDeclarationUpdatability(validator, oldRootDecl, newRootDecl)

	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	return nil
}

func (validator *LegacyContractUpdateValidator) report(err error) {
	validator.underlyingUpdateValidator.report(err)
}

func (validator *LegacyContractUpdateValidator) checkEntitlementsUpgrade(oldType *ast.ReferenceType, newType *ast.ReferenceType) error {
	newAuthorization := newType.Authorization
}

func (validator *LegacyContractUpdateValidator) checkTypeUpgradability(oldType ast.Type, newType ast.Type) error {

	switch oldType := oldType.(type) {
	case *ast.OptionalType:
		if newOptional, isOptional := newType.(*ast.OptionalType); isOptional {
			return validator.checkTypeUpgradability(oldType.Type, newOptional.Type)
		}
	case *ast.ReferenceType:
		if newReference, isReference := newType.(*ast.ReferenceType); isReference {
			if newReference.Authorization != nil {
				err := validator.checkEntitlementsUpgrade(oldType, newReference)
				if err != nil {
					return err
				}
			}
			return validator.checkTypeUpgradability(oldType.Type, newReference.Type)
		}
	case *ast.IntersectionType:
		// intersection types cannot be upgraded unless they have a legacy restricted type,
		// in which case they must be upgraded according to the migration rules: i.e. R{I} -> R
		if oldType.LegacyRestrictedType == nil {
			break
		}
		return validator.checkTypeUpgradability(oldType.LegacyRestrictedType, newType)
	case *ast.VariableSizedType:
		if newVariableSizedType, isVariableSizedType := newType.(*ast.VariableSizedType); isVariableSizedType {
			return validator.checkTypeUpgradability(oldType.Type, newVariableSizedType.Type)
		}
	case *ast.ConstantSizedType:
		if newConstantSizedType, isConstantSizedType := newType.(*ast.ConstantSizedType); isConstantSizedType {
			if oldType.Size.Value.Cmp(newConstantSizedType.Size.Value) != 0 ||
				oldType.Size.Base != newConstantSizedType.Size.Base {
				return newTypeMismatchError(oldType, newConstantSizedType)
			}
			return validator.checkTypeUpgradability(oldType.Type, newConstantSizedType.Type)
		}
	case *ast.DictionaryType:
		if newDictionaryType, isDictionaryType := newType.(*ast.DictionaryType); isDictionaryType {
			err := validator.checkTypeUpgradability(oldType.KeyType, newDictionaryType.KeyType)
			if err != nil {
				return err
			}
			return validator.checkTypeUpgradability(oldType.ValueType, newDictionaryType.ValueType)
		}
	case *ast.InstantiationType:
		// if the type is a Capability, allow the borrow type to change according to the normal upgrade rules
		if oldNominalType, isNominal := oldType.Type.(*ast.NominalType); isNominal &&
			oldNominalType.Identifier.Identifier == "Capability" {

			if instantiationType, isInstantiation := newType.(*ast.InstantiationType); isInstantiation {
				if newNominalType, isNominal := oldType.Type.(*ast.NominalType); isNominal &&
					newNominalType.Identifier.Identifier == "Capability" {

					// Capability insantiation types must have exactly 1 type argument
					if len(oldType.TypeArguments) != 1 || len(instantiationType.TypeArguments) != 1 {
						break
					}

					oldTypeArg := oldType.TypeArguments[0]
					newTypeArg := instantiationType.TypeArguments[0]

					return validator.checkTypeUpgradability(oldTypeArg.Type, newTypeArg.Type)
				}
			}
		}
	}

	return oldType.CheckEqual(newType, validator)

}

func (validator *LegacyContractUpdateValidator) checkField(oldField *ast.FieldDeclaration, newField *ast.FieldDeclaration) {
	oldType := oldField.TypeAnnotation.Type
	newType := newField.TypeAnnotation.Type

	err := validator.checkTypeUpgradability(oldType, newType)
	if err == nil {
		return
	}

	validator.report(&FieldMismatchError{
		DeclName:  validator.getCurrentDeclaration().DeclarationIdentifier().Identifier,
		FieldName: newField.Identifier.Identifier,
		Err:       err,
		Range:     ast.NewUnmeteredRangeFromPositioned(newField.TypeAnnotation),
	})

}
