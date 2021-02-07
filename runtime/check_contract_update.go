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
	currentFieldName        string
	visitedDecls            map[string]bool
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

	return &ContractUpdateValidator{
			oldContract:  oldCompDecls[0],
			newContract:  newCompDecls[0],
			visitedDecls: map[string]bool{}},
		nil
}

func (validator *ContractUpdateValidator) Validate() error {
	validator.entryPoint = validator.newContract.Identifier.Identifier
	return validator.checkCompositeDeclUpdatability(validator.oldContract, validator.newContract)
}

func (validator *ContractUpdateValidator) checkCompositeDeclUpdatability(
	oldCompositeDecl *ast.CompositeDeclaration,
	newCompositeDecl *ast.CompositeDeclaration) error {

	parentDeclName := validator.currentDeclName
	validator.currentDeclName = newCompositeDecl.Identifier.Identifier
	defer func() {
		validator.currentDeclName = parentDeclName
	}()

	// If the same same decl is already visited, then do no check again.
	// This also is used to avoid getting stuck on circular dependencies between composite decls.
	if validator.visitedDecls[validator.currentDeclName] {
		return nil
	}

	validator.visitedDecls[validator.currentDeclName] = true

	oldFields := oldCompositeDecl.Members.Fields()
	newFields := newCompositeDecl.Members.Fields()

	// Updated contract has to have at-most the same number of field as the old contract.
	// Any additional field may cause crashes/garbage-values when deserializing the
	// already-stored data. However, having less number of fields is fine for now. It will
	// only leave some unused-data, and will not do any harm to the programs that are running.

	// This is a fail-fast check.
	if len(oldFields) < len(newFields) {
		if validator.isNestedDecl() {
			return fmt.Errorf("too many fields in `%s`. expected %d, found %d",
				validator.currentDeclName, len(oldFields), len(newFields))
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
		err := validator.visitField(newField, oldFiledTypes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) visitField(newField *ast.FieldDeclaration, oldFiledTypes map[string]ast.Type) error {
	prevFieldName := validator.currentFieldName
	validator.currentFieldName = newField.Identifier.Identifier
	defer func() {
		validator.currentFieldName = prevFieldName
	}()

	oldType := oldFiledTypes[validator.currentFieldName]
	if oldType == nil {
		if validator.isNestedDecl() {
			return fmt.Errorf("found new field `%s` in `%s", validator.currentDeclName, validator.currentFieldName)
		}

		return fmt.Errorf("found new field `%s`", validator.currentFieldName)
	}

	return validator.checkType(oldType, newField.TypeAnnotation.Type)
}

func (validator *ContractUpdateValidator) checkType(oldType ast.Type, newType ast.Type) error {
	switch expectedType := oldType.(type) {
	case *ast.NominalType:
		return validator.checkNominalTypeEquality(expectedType, newType)
	case *ast.OptionalType:
		return validator.checkOptionalTypeEquality(expectedType, newType)
	case *ast.VariableSizedType:
		return validator.checkVarSizedTypeEquality(expectedType, newType)
	case *ast.ConstantSizedType:
		return validator.checkConstantSizedTypeEquality(expectedType, newType)
	case *ast.DictionaryType:
		return validator.checkDictionaryTypeEquality(expectedType, newType)
	case *ast.RestrictedType:
		return validator.checkRestrictedTypeEquality(expectedType, newType)
	case *ast.FunctionType:
	case *ast.ReferenceType:
		// non storable type
		panic(errors.NewUnreachableError())
	default:
		if !newType.Equal(oldType) {
			return validator.getTypeMismatchError(oldType, newType)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) checkNominalTypeEquality(expected *ast.NominalType, found ast.Type) error {
	foundNominalType, ok := found.(*ast.NominalType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	ok = validator.checkNameEquality(expected, foundNominalType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	var compositeDeclName string
	if isQualifiedName(foundNominalType) {
		compositeDeclName = foundNominalType.NestedIdentifiers[0].Identifier
	} else {
		compositeDeclName = foundNominalType.Identifier.Identifier
	}

	// If the two types are nominal, then the fields of the two composite declarations
	// referred by this nominal type, also needs to be compatible.

	validator.loadCompositeDecls()

	oldCompositeDecl := validator.oldNestedCompositeDecls[compositeDeclName]
	if oldCompositeDecl == nil {
		// If the declaration is not available, that means this is an imported contract.
		// Thus, no need to validate anymore.
		return nil
	}

	newCompositeDecl := validator.newNestedCompositeDecls[compositeDeclName]
	if newCompositeDecl == nil {
		// If the declaration is not available, that means this is an imported contract.
		// Thus, no need to validate anymore. Ideally shouldn't reach here.
		return nil
	}

	return validator.checkCompositeDeclUpdatability(oldCompositeDecl, newCompositeDecl)
}

func (validator *ContractUpdateValidator) checkOptionalTypeEquality(expected *ast.OptionalType, found ast.Type) error {
	foundOptionalType, ok := found.(*ast.OptionalType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	return validator.checkType(expected.Type, foundOptionalType.Type)
}

func (validator *ContractUpdateValidator) checkVarSizedTypeEquality(expected *ast.VariableSizedType, found ast.Type) error {
	foundVarSizedType, ok := found.(*ast.VariableSizedType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	return validator.checkType(expected.Type, foundVarSizedType.Type)
}

func (validator *ContractUpdateValidator) checkConstantSizedTypeEquality(expected *ast.ConstantSizedType, found ast.Type) error {
	foundVarSizedType, ok := found.(*ast.ConstantSizedType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	// Check size
	if foundVarSizedType.Size.Value.Cmp(expected.Size.Value) != 0 ||
		foundVarSizedType.Size.Base != expected.Size.Base {
		return validator.getTypeMismatchError(expected, found)
	}

	// Check type
	return validator.checkType(expected.Type, foundVarSizedType.Type)
}

func (validator *ContractUpdateValidator) checkDictionaryTypeEquality(expected *ast.DictionaryType, found ast.Type) error {
	foundDictionaryType, ok := found.(*ast.DictionaryType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	err := validator.checkType(expected.KeyType, foundDictionaryType.KeyType)
	if err != nil {
		return err
	}

	return validator.checkType(expected.ValueType, foundDictionaryType.ValueType)
}

func (validator *ContractUpdateValidator) checkRestrictedTypeEquality(expected *ast.RestrictedType, found ast.Type) error {
	foundRestrictedType, ok := found.(*ast.RestrictedType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	err := validator.checkType(expected.Type, foundRestrictedType.Type)
	if err != nil || len(expected.Restrictions) != len(foundRestrictedType.Restrictions) {
		return validator.getTypeMismatchError(expected, found)
	}

	for index, expectedRestriction := range expected.Restrictions {
		foundRestriction := foundRestrictedType.Restrictions[index]
		err := validator.checkType(expectedRestriction, foundRestriction)
		if err != nil {
			return validator.getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) checkInstantiationTypeEquality(expected *ast.InstantiationType, found ast.Type) error {
	foundInstType, ok := found.(*ast.InstantiationType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	err := validator.checkType(expected.Type, foundInstType.Type)
	if err != nil || len(expected.TypeArguments) != len(foundInstType.TypeArguments) {
		return validator.getTypeMismatchError(expected, found)
	}

	for index, typeArgs := range expected.TypeArguments {
		otherTypeArgs := foundInstType.TypeArguments[index]
		if !typeArgs.Type.Equal(otherTypeArgs.Type) {
			return validator.getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) loadCompositeDecls() {
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
}

func (validator *ContractUpdateValidator) checkNameEquality(oldNominalType *ast.NominalType, newNominalType *ast.NominalType) bool {
	oldIsQualifiedName := isQualifiedName(oldNominalType)
	newIsQualifiedName := isQualifiedName(newNominalType)

	// A field with a composite type can be defined in two ways:
	// 	- Using type name (var x @ResourceName)
	//	- Using qualified type name (var x @ContractName.ResourceName)

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

func (validator *ContractUpdateValidator) getTypeMismatchError(
	oldType ast.Type,
	newType ast.Type) error {

	if validator.isNestedDecl() {
		return fmt.Errorf("type annotations does not match for field `%s` in `%s`. expected `%s`, found `%s`",
			validator.currentFieldName, validator.currentDeclName, oldType, newType)
	}

	return fmt.Errorf("type annotations does not match for field `%s`. expected `%s`, found `%s`",
		validator.currentFieldName, oldType, newType)
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
