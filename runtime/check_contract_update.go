package runtime

import (
	"fmt"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
)

type ContractUpdateValidator struct {
	oldContract             *ast.CompositeDeclaration
	newContract             *ast.CompositeDeclaration
	oldNestedCompositeDecls map[string]*ast.CompositeDeclaration
	newNestedCompositeDecls map[string]*ast.CompositeDeclaration
	entryPoint              string
	currentDeclName         string
}

func NewContractUpdateValidator(
	r *interpreterRuntime,
	context Context,
	existingCode []byte,
	newProgram *ast.Program) (*ContractUpdateValidator, error) {

	oldProgram, err := r.parse(existingCode, context)
	if err != nil {
		return nil, err
	}

	oldCompDecls := oldProgram.CompositeDeclarations()
	if oldCompDecls == nil || len(oldCompDecls) != 1 {
		panic(errors.NewUnreachableError())
	}

	newCompDecls := newProgram.CompositeDeclarations()
	if newCompDecls == nil || len(newCompDecls) != 1 {
		panic(errors.NewUnreachableError())
	}

	return &ContractUpdateValidator{oldContract: oldCompDecls[0], newContract: newCompDecls[0]}, nil
}

func (validator *ContractUpdateValidator) Validate() error {
	// TODO: check for interfaces?

	validator.entryPoint = validator.newContract.Identifier.Identifier
	return validator.checkCompositeDeclUpdatability(validator.oldContract, validator.newContract)
}

func (validator *ContractUpdateValidator) checkCompositeDeclUpdatability(
	oldCompositeDecl *ast.CompositeDeclaration,
	newCompositeDecl *ast.CompositeDeclaration) error {

	validator.currentDeclName = newCompositeDecl.Identifier.Identifier

	oldFields := oldCompositeDecl.Members.Fields()
	newFields := newCompositeDecl.Members.Fields()

	// Updated contract has to have at-most the same number of field as the old contract.
	// Any additional field may cause crashes/garbage-values when deserializing the
	// already-stored data. However, having less number of fields is fine for now. It will
	// only leave some unused-data, and will not do any harm to the programs that are running.

	// This is a fail-fast check.
	if len(oldFields) < len(newFields) {
		if validator.isNestedDecl() {
			return fmt.Errorf("too many fields in %q. expected %d, found %d",
				newCompositeDecl.Identifier, len(oldFields), len(newFields))
		}

		return fmt.Errorf("too many fields. expected %d, found %d",
			len(oldFields), len(newFields))
	}

	// Put the old field-types against their field-names to a map, for faster lookup.
	oldFiledTypes := map[string]ast.Type{}
	for _, field := range oldFields {
		oldFiledTypes[field.Identifier.Identifier] = field.TypeAnnotation.Type
	}

	for _, newField := range newFields {
		fieldName := newField.Identifier.Identifier
		oldType := oldFiledTypes[fieldName]
		if oldType == nil {
			if validator.isNestedDecl() {
				return fmt.Errorf("found new field %q in %q", newCompositeDecl.Identifier, fieldName)
			}

			return fmt.Errorf("found new field %q", fieldName)
		}

		newType := newField.TypeAnnotation.Type

		switch newNominalType := newType.(type) {
		case *ast.NominalType:
			oldNominalType, ok := oldType.(*ast.NominalType)
			if !ok {
				return validator.getTypeMismatchError(fieldName, oldType, newType)
			}

			// If the two types are nominal, then the fields of the two composite declarations
			// referred by this nominal type, also needs to be compatible.
			err := validator.checkNominalType(fieldName, oldNominalType, newNominalType)
			if err != nil {
				return err
			}
		default:
			if !newType.Equal(oldType) {
				return validator.getTypeMismatchError(fieldName, oldType, newType)
			}
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) getTypeMismatchError(
	fieldName string,
	oldType ast.Type,
	newType ast.Type) error {

	if validator.isNestedDecl() {
		return fmt.Errorf("type annotations does not match for field %q in %q. expected %q, found %q",
			fieldName, validator.currentDeclName, oldType, newType)
	}

	return fmt.Errorf("type annotations does not match for field %q. expected %q, found %q",
		fieldName, oldType, newType)
}

func (validator *ContractUpdateValidator) checkNominalType(
	fieldName string,
	oldNominalType *ast.NominalType,
	newNominalType *ast.NominalType) error {

	// A field with a composite type can be defined in two ways:
	// 	- Using type name (var x @ResourceName)
	//	- Using qualified type name (var x @ContractName.ResourceName)

	ok := validator.checkNameEquality(oldNominalType, newNominalType)
	if !ok {
		return validator.getTypeMismatchError(fieldName, oldNominalType, newNominalType)
	}

	var compositeDeclName string
	if isQualifiedName(newNominalType) {
		compositeDeclName = newNominalType.NestedIdentifiers[0].Identifier
	} else {
		compositeDeclName = newNominalType.Identifier.Identifier
	}

	if validator.oldNestedCompositeDecls == nil {
		validator.oldNestedCompositeDecls = map[string]*ast.CompositeDeclaration{}
		for _, nestedComposite := range validator.oldContract.Members.Composites() {
			validator.oldNestedCompositeDecls[nestedComposite.Identifier.Identifier] = nestedComposite
		}
	}

	if validator.newNestedCompositeDecls == nil {
		validator.newNestedCompositeDecls = map[string]*ast.CompositeDeclaration{}
		for _, nestedComposite := range validator.newContract.Members.Composites() {
			validator.newNestedCompositeDecls[nestedComposite.Identifier.Identifier] = nestedComposite
		}
	}

	oldCompositeDecl := validator.oldNestedCompositeDecls[compositeDeclName]
	if oldCompositeDecl == nil {
		// If the declaration not available, that means this is an imported contract.
		// Thus, no need to validate anymore.
		return nil
	}

	newCompositeDecl := validator.newNestedCompositeDecls[compositeDeclName]
	if newCompositeDecl == nil {
		// If the declaration not available, that means this is an imported contract.
		// Thus, no need to validate anymore. Ideally shouldn't reach here.
		return nil
	}

	return validator.checkCompositeDeclUpdatability(oldCompositeDecl, newCompositeDecl)
}

func (validator *ContractUpdateValidator) checkNameEquality(oldNominalType *ast.NominalType, newNominalType *ast.NominalType) bool {
	oldIsQualifiedName := isQualifiedName(oldNominalType)
	newIsQualifiedName := isQualifiedName(newNominalType)

	if oldIsQualifiedName && !newIsQualifiedName {
		return validator.checkIdentifierEquality(oldNominalType, newNominalType)
	}

	if newIsQualifiedName && !oldIsQualifiedName {
		return validator.checkIdentifierEquality(newNominalType, oldNominalType)
	}

	return newNominalType.Equal(oldNominalType)
}

func (validator *ContractUpdateValidator) checkIdentifierEquality(
	qualifiedNominalType *ast.NominalType,
	simpleNominalType *ast.NominalType) bool {

	// Situation:
	// qualifiedNominalType -> identifier: A, nestedIdentifiers: [foo, bar, ...]
	// simpleNominalType -> identifier: foo,  nestedIdentifiers: [bar, ...]

	// If the qualified identifier refers to a composite decl that is not the enclosing contract,
	// then it must be referring to an imported contract. That means the two types are no longer the same.
	if qualifiedNominalType.Identifier.Identifier != validator.oldContract.Identifier.Identifier {
		return false
	}

	if qualifiedNominalType.NestedIdentifiers[0].Identifier != simpleNominalType.Identifier.Identifier {
		return false
	}

	return isEqual(simpleNominalType.NestedIdentifiers, qualifiedNominalType.NestedIdentifiers[1:])
}

func (validator *ContractUpdateValidator) isNestedDecl() bool {
	return validator.entryPoint != validator.currentDeclName
}

func isEqual(array1 []ast.Identifier, array2 []ast.Identifier) bool {
	if len(array1) != len(array2) {
		return false
	}

	for index, element := range array2 {
		if array1[index] != element {
			return false
		}
	}
	return true
}

func isQualifiedName(nominalType *ast.NominalType) bool {
	return len(nominalType.NestedIdentifiers) > 0
}
