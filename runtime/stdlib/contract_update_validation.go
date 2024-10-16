/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"sort"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
)

const typeRemovalPragmaName = "removedType"

type UpdateValidator interface {
	ast.TypeEqualityChecker

	Validate() error
	report(error)
	Location() common.Location

	getCurrentDeclaration() ast.Declaration
	setCurrentDeclaration(ast.Declaration)

	checkField(oldField *ast.FieldDeclaration, newField *ast.FieldDeclaration)
	checkNestedDeclarationRemoval(
		nestedDeclaration ast.Declaration,
		oldContainingDeclaration ast.Declaration,
		newContainingDeclaration ast.Declaration,
		removedTypes *orderedmap.OrderedMap[string, struct{}],
	)
	getAccountContractNames(address common.Address) ([]string, error)

	checkDeclarationKindChange(
		oldDeclaration ast.Declaration,
		newDeclaration ast.Declaration,
	) bool

	isTypeRemovalEnabled() bool
	WithTypeRemovalEnabled(enabled bool) UpdateValidator
}

type checkConformanceFunc func(
	oldDecl *ast.CompositeDeclaration,
	newDecl *ast.CompositeDeclaration,
)

type ContractUpdateValidator struct {
	*TypeComparator

	location                     common.Location
	contractName                 string
	oldProgram                   *ast.Program
	newProgram                   *ast.Program
	currentDecl                  ast.Declaration
	importLocations              map[ast.Identifier]common.Location
	accountContractNamesProvider AccountContractNamesProvider
	errors                       []error
	typeRemovalEnabled           bool
}

// ContractUpdateValidator should implement ast.TypeEqualityChecker
var _ ast.TypeEqualityChecker = &ContractUpdateValidator{}
var _ UpdateValidator = &ContractUpdateValidator{}

// NewContractUpdateValidator initializes and returns a validator, without performing any validation.
// Invoke the `Validate()` method of the validator returned, to start validating the contract.
func NewContractUpdateValidator(
	location common.Location,
	contractName string,
	accountContractNamesProvider AccountContractNamesProvider,
	oldProgram *ast.Program,
	newProgram *ast.Program,
) *ContractUpdateValidator {

	return &ContractUpdateValidator{
		location:                     location,
		oldProgram:                   oldProgram,
		newProgram:                   newProgram,
		contractName:                 contractName,
		accountContractNamesProvider: accountContractNamesProvider,
		importLocations:              map[ast.Identifier]common.Location{},
		TypeComparator:               &TypeComparator{},
	}
}

func (validator *ContractUpdateValidator) Location() common.Location {
	return validator.location
}

func (validator *ContractUpdateValidator) isTypeRemovalEnabled() bool {
	return validator.typeRemovalEnabled
}

func (validator *ContractUpdateValidator) WithTypeRemovalEnabled(enabled bool) UpdateValidator {
	validator.typeRemovalEnabled = enabled
	return validator
}

func (validator *ContractUpdateValidator) getCurrentDeclaration() ast.Declaration {
	return validator.currentDecl
}

func (validator *ContractUpdateValidator) setCurrentDeclaration(decl ast.Declaration) {
	validator.currentDecl = decl
}

func (validator *ContractUpdateValidator) getAccountContractNames(address common.Address) ([]string, error) {
	return validator.accountContractNamesProvider.GetAccountContractNames(address)
}

// Validate validates the contract update, and returns an error if it is an invalid update.
func (validator *ContractUpdateValidator) Validate() error {
	oldRootDecl := getRootDeclarationOfOldProgram(validator, validator.oldProgram, validator.newProgram)
	if validator.hasErrors() {
		return validator.getContractUpdateError()
	}

	newRootDecl := getRootDeclaration(validator, validator.newProgram)
	if validator.hasErrors() {
		return validator.getContractUpdateError()
	}

	validator.TypeComparator.RootDeclIdentifier = newRootDecl.DeclarationIdentifier()
	validator.TypeComparator.expectedIdentifierImportLocations = collectImports(validator, validator.oldProgram)
	validator.TypeComparator.foundIdentifierImportLocations = collectImports(validator, validator.newProgram)

	if validator.hasErrors() {
		return validator.getContractUpdateError()
	}

	checkDeclarationUpdatability(
		validator,
		oldRootDecl,
		newRootDecl,
		validator.checkConformance,
	)

	if validator.hasErrors() {
		return validator.getContractUpdateError()
	}

	return nil
}

func collectImports(validator UpdateValidator, program *ast.Program) map[string]common.Location {
	importLocations := map[string]common.Location{}

	imports := program.ImportDeclarations()

	for _, importDecl := range imports {
		importLocation := importDecl.Location

		addressLocation, ok := importLocation.(common.AddressLocation)
		if !ok {
			// e.g: Crypto
			continue
		}

		// if there are no identifiers given, the import covers all of them
		if len(importDecl.Identifiers) == 0 {
			allLocations, err := validator.getAccountContractNames(addressLocation.Address)
			if err != nil {
				validator.report(err)
			}
			for _, identifier := range allLocations {
				// associate the location of an identifier's import with the location it's being imported from
				// this assumes that two imports cannot have the same name, which should be prevented by the type checker
				importLocations[identifier] = common.AddressLocation{
					Name:    identifier,
					Address: addressLocation.Address,
				}
			}
		} else {
			for _, identifier := range importDecl.Identifiers {
				name := identifier.Identifier
				// associate the location of an identifier's import with the location it's being imported from.
				// This assumes that two imports cannot have the same name, which should be prevented by the type checker
				importLocations[name] = common.AddressLocation{
					Name:    name,
					Address: addressLocation.Address,
				}
			}
		}
	}

	return importLocations
}

func getRootDeclaration(validator UpdateValidator, program *ast.Program) ast.Declaration {
	decl, err := getRootDeclarationOfProgram(program)

	if err != nil {
		validator.report(&ContractNotFoundError{
			Range: ast.NewUnmeteredRangeFromPositioned(program),
		})
	}

	return decl
}

func getRootDeclarationOfOldProgram(validator UpdateValidator, program *ast.Program, position ast.HasPosition) ast.Declaration {
	decl, err := getRootDeclarationOfProgram(program)

	if err != nil {
		validator.report(&OldProgramError{
			Err: &ContractNotFoundError{
				Range: ast.NewUnmeteredRangeFromPositioned(position),
			},
			Location: validator.Location(),
		})
	}

	return decl
}

func getRootDeclarationOfProgram(program *ast.Program) (ast.Declaration, error) {
	compositeDecl := program.SoleContractDeclaration()
	if compositeDecl != nil {
		return compositeDecl, nil
	}

	interfaceDecl := program.SoleContractInterfaceDeclaration()
	if interfaceDecl != nil {
		return interfaceDecl, nil
	}

	return nil, &ContractNotFoundError{
		Range: ast.NewUnmeteredRangeFromPositioned(program),
	}
}

func (validator *ContractUpdateValidator) hasErrors() bool {
	return len(validator.errors) > 0
}

func collectRemovedTypePragmas(
	validator UpdateValidator,
	pragmas []*ast.PragmaDeclaration,
	reportErrors bool,
) *orderedmap.OrderedMap[string, struct{}] {
	removedTypes := orderedmap.New[orderedmap.OrderedMap[string, struct{}]](len(pragmas))

	for _, pragma := range pragmas {
		invocationExpression, isInvocation := pragma.Expression.(*ast.InvocationExpression)
		if !isInvocation {
			continue
		}

		invokedIdentifier, isIdentifier := invocationExpression.InvokedExpression.(*ast.IdentifierExpression)
		if !isIdentifier || invokedIdentifier.Identifier.Identifier != typeRemovalPragmaName {
			continue
		}

		if len(invocationExpression.Arguments) != 1 {
			if reportErrors {
				validator.report(&InvalidTypeRemovalPragmaError{
					Expression: pragma.Expression,
					Range:      ast.NewUnmeteredRangeFromPositioned(pragma.Expression),
				})
			}
			continue
		}

		removedTypeName, isIdentifier := invocationExpression.Arguments[0].Expression.(*ast.IdentifierExpression)
		if !isIdentifier {
			if reportErrors {
				validator.report(&InvalidTypeRemovalPragmaError{
					Expression: pragma.Expression,
					Range:      ast.NewUnmeteredRangeFromPositioned(pragma.Expression),
				})
			}
			continue
		}

		removedTypes.Set(removedTypeName.Identifier.Identifier, struct{}{})
	}

	return removedTypes
}

func checkDeclarationUpdatability(
	validator UpdateValidator,
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
	checkConformance checkConformanceFunc,
) {

	if !validator.checkDeclarationKindChange(oldDeclaration, newDeclaration) {
		return
	}

	parentDecl := validator.getCurrentDeclaration()
	validator.setCurrentDeclaration(newDeclaration)
	defer func() {
		validator.setCurrentDeclaration(parentDecl)
	}()

	oldIdentifier := oldDeclaration.DeclarationIdentifier()
	newIdentifier := newDeclaration.DeclarationIdentifier()

	if oldIdentifier.Identifier != newIdentifier.Identifier {
		validator.report(&NameMismatchError{
			OldName: oldIdentifier.Identifier,
			NewName: newIdentifier.Identifier,
			Range:   ast.NewUnmeteredRangeFromPositioned(newIdentifier),
		})
	}

	checkFields(validator, oldDeclaration, newDeclaration)

	checkNestedDeclarations(validator, oldDeclaration, newDeclaration, checkConformance)

	if newDecl, ok := newDeclaration.(*ast.CompositeDeclaration); ok {
		if oldDecl, ok := oldDeclaration.(*ast.CompositeDeclaration); ok {
			checkConformance(oldDecl, newDecl)
		}
	}
}

func checkFields(
	validator UpdateValidator,
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) {

	oldFields := oldDeclaration.DeclarationMembers().FieldsByIdentifier()
	newFields := newDeclaration.DeclarationMembers().Fields()

	// Updated contract has to have at-most the same number of field as the old contract.
	// Any additional field may cause crashes/garbage-values when deserializing the already-stored data.
	// However, having fewer fields is fine for now. It will only leave some unused data,
	// and will not do any harm to the programs that are running.

	for _, newField := range newFields {
		oldField := oldFields[newField.Identifier.Identifier]
		if oldField == nil {
			validator.report(&ExtraneousFieldError{
				DeclName:  newDeclaration.DeclarationIdentifier().Identifier,
				FieldName: newField.Identifier.Identifier,
				Range:     ast.NewUnmeteredRangeFromPositioned(newField.Identifier),
			})

			continue
		}

		validator.checkField(oldField, newField)
	}
}

func (validator *ContractUpdateValidator) checkField(oldField *ast.FieldDeclaration, newField *ast.FieldDeclaration) {
	err := oldField.TypeAnnotation.Type.CheckEqual(newField.TypeAnnotation.Type, validator)
	if err != nil {
		validator.report(&FieldMismatchError{
			DeclName:  validator.currentDecl.DeclarationIdentifier().Identifier,
			FieldName: newField.Identifier.Identifier,
			Err:       err,
			Range:     ast.NewUnmeteredRangeFromPositioned(newField.TypeAnnotation),
		})
	}
}

func (validator *ContractUpdateValidator) checkDeclarationKindChange(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) bool {
	// Do not allow converting between different types of composite declarations:
	// e.g: - 'contracts' and 'contract-interfaces',
	//      - 'structs' and 'enums'
	if oldDeclaration.DeclarationKind() != newDeclaration.DeclarationKind() {
		validator.report(&InvalidDeclarationKindChangeError{
			Name:    oldDeclaration.DeclarationIdentifier().Identifier,
			OldKind: oldDeclaration.DeclarationKind(),
			NewKind: newDeclaration.DeclarationKind(),
			Range:   ast.NewUnmeteredRangeFromPositioned(newDeclaration.DeclarationIdentifier()),
		})

		return false
	}

	return true
}

func (validator *ContractUpdateValidator) checkNestedDeclarationRemoval(
	nestedDeclaration ast.Declaration,
	_ ast.Declaration,
	newContainingDeclaration ast.Declaration,
	removedTypes *orderedmap.OrderedMap[string, struct{}],
) {
	declarationKind := nestedDeclaration.DeclarationKind()

	// OK to remove events - they are not stored
	if declarationKind == common.DeclarationKindEvent {
		return
	}

	if validator.typeRemovalEnabled {
		// OK to remove a type if it is included in a #removedType pragma, and it is not an interface
		if removedTypes.Contains(nestedDeclaration.DeclarationIdentifier().Identifier) &&
			!declarationKind.IsInterfaceDeclaration() {
			return
		}
	}

	validator.report(&MissingDeclarationError{
		Name: nestedDeclaration.DeclarationIdentifier().Identifier,
		Kind: declarationKind,
		Range: ast.NewUnmeteredRangeFromPositioned(
			newContainingDeclaration.DeclarationIdentifier(),
		),
	})
}

func (validator *ContractUpdateValidator) oldTypeID(oldType *ast.NominalType) common.TypeID {
	oldImportLocation := validator.expectedIdentifierImportLocations[oldType.Identifier.Identifier]
	qualifiedIdentifier := oldType.String()
	if oldImportLocation == nil {
		return common.TypeID(qualifiedIdentifier)
	}
	return oldImportLocation.TypeID(nil, qualifiedIdentifier)
}

func checkTypeNotRemoved(
	validator UpdateValidator,
	newDeclaration ast.Declaration,
	removedTypes *orderedmap.OrderedMap[string, struct{}],
) {
	if !validator.isTypeRemovalEnabled() {
		return
	}

	if removedTypes.Contains(newDeclaration.DeclarationIdentifier().Identifier) {
		validator.report(&UseOfRemovedTypeError{
			Declaration: newDeclaration,
			Range:       ast.NewUnmeteredRangeFromPositioned(newDeclaration),
		})
	}
}

func checkNestedDeclarations(
	validator UpdateValidator,
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
	checkConformance checkConformanceFunc,
) {

	var removedTypes *orderedmap.OrderedMap[string, struct{}]
	if validator.isTypeRemovalEnabled() {
		// process pragmas first, as they determine whether types can later be removed
		oldRemovedTypes := collectRemovedTypePragmas(
			validator,
			oldDeclaration.DeclarationMembers().Pragmas(),
			// Do not report errors for pragmas in the old code.
			// We are only interested in collecting the pragmas in old code.
			// This also avoid reporting mixed errors from both old and new codes.
			false,
		)

		removedTypes = collectRemovedTypePragmas(
			validator,
			newDeclaration.DeclarationMembers().Pragmas(),
			true,
		)

		// #typeRemoval pragmas cannot be removed, so any that appear in the old program must appear in the new program
		// they can however, be added, so use the new program's type removals for the purposes of checking the upgrade
		oldRemovedTypes.Foreach(func(oldRemovedType string, _ struct{}) {
			if !removedTypes.Contains(oldRemovedType) {
				validator.report(&TypeRemovalPragmaRemovalError{
					RemovedType: oldRemovedType,
				})
			}
		})
	}

	oldNominalTypeDecls := getNestedNominalTypeDecls(oldDeclaration)

	// Check nested structs, enums, etc.
	newNestedCompositeDecls := newDeclaration.DeclarationMembers().Composites()
	for _, newNestedDecl := range newNestedCompositeDecls {
		checkTypeNotRemoved(validator, newNestedDecl, removedTypes)
		oldNestedDecl, found := oldNominalTypeDecls[newNestedDecl.Identifier.Identifier]
		if !found {
			// Then it's a new declaration
			continue
		}

		checkDeclarationUpdatability(validator, oldNestedDecl, newNestedDecl, checkConformance)

		// If there's a matching new decl, then remove the old one from the map.
		delete(oldNominalTypeDecls, newNestedDecl.Identifier.Identifier)
	}

	// Check nested attachments, etc.
	newNestedAttachmentDecls := newDeclaration.DeclarationMembers().Attachments()
	for _, newNestedDecl := range newNestedAttachmentDecls {
		checkTypeNotRemoved(validator, newNestedDecl, removedTypes)
		oldNestedDecl, found := oldNominalTypeDecls[newNestedDecl.Identifier.Identifier]
		if !found {
			// Then it's a new declaration
			continue
		}

		checkDeclarationUpdatability(validator, oldNestedDecl, newNestedDecl, checkConformance)

		// If there's a matching new decl, then remove the old one from the map.
		delete(oldNominalTypeDecls, newNestedDecl.Identifier.Identifier)
	}

	// Check nested interfaces.
	newNestedInterfaces := newDeclaration.DeclarationMembers().Interfaces()
	for _, newNestedDecl := range newNestedInterfaces {
		checkTypeNotRemoved(validator, newNestedDecl, removedTypes)
		oldNestedDecl, found := oldNominalTypeDecls[newNestedDecl.Identifier.Identifier]
		if !found {
			// Then this is a new declaration.
			continue
		}

		checkDeclarationUpdatability(validator, oldNestedDecl, newNestedDecl, checkConformance)

		// If there's a matching new decl, then remove the old one from the map.
		delete(oldNominalTypeDecls, newNestedDecl.Identifier.Identifier)
	}

	// The remaining old declarations don't have a corresponding new declaration,
	// i.e., an existing declaration was removed.
	// Hence, report an error.

	missingDeclarations := make([]ast.Declaration, 0, len(oldNominalTypeDecls))

	for _, declaration := range oldNominalTypeDecls { //nolint:maprange
		missingDeclarations = append(missingDeclarations, declaration)
	}

	sort.Slice(missingDeclarations, func(i, j int) bool {
		return missingDeclarations[i].DeclarationIdentifier().Identifier <
			missingDeclarations[j].DeclarationIdentifier().Identifier
	})

	for _, declaration := range missingDeclarations {
		validator.checkNestedDeclarationRemoval(declaration, oldDeclaration, newDeclaration, removedTypes)
	}

	// Check enum-cases, if there are any.
	checkEnumCases(validator, oldDeclaration, newDeclaration)
}

func getNestedNominalTypeDecls(declaration ast.Declaration) map[string]ast.Declaration {
	compositeAndInterfaceDecls := map[string]ast.Declaration{}

	nestedCompositeDecls := declaration.DeclarationMembers().CompositesByIdentifier()
	for identifier, nestedDecl := range nestedCompositeDecls { //nolint:maprange
		compositeAndInterfaceDecls[identifier] = nestedDecl
	}

	nestedAttachmentDecls := declaration.DeclarationMembers().AttachmentsByIdentifier()
	for identifier, nestedDecl := range nestedAttachmentDecls { //nolint:maprange
		compositeAndInterfaceDecls[identifier] = nestedDecl
	}

	nestedInterfaceDecls := declaration.DeclarationMembers().InterfacesByIdentifier()
	for identifier, nestedDecl := range nestedInterfaceDecls { //nolint:maprange
		compositeAndInterfaceDecls[identifier] = nestedDecl
	}

	return compositeAndInterfaceDecls
}

// checkEnumCases validates updating enum cases. Updated enum must:
//   - Have at-least the same number of enum-cases as the old enum (Adding is allowed, but no removals).
//   - Preserve the order of the old enum-cases (Adding to top/middle is not allowed, swapping is not allowed).
func checkEnumCases(
	validator UpdateValidator,
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) {
	newEnumCases := newDeclaration.DeclarationMembers().EnumCases()
	oldEnumCases := oldDeclaration.DeclarationMembers().EnumCases()

	newEnumCaseCount := len(newEnumCases)
	oldEnumCaseCount := len(oldEnumCases)

	if newEnumCaseCount < oldEnumCaseCount {
		validator.report(&MissingEnumCasesError{
			DeclName: newDeclaration.DeclarationIdentifier().Identifier,
			Expected: oldEnumCaseCount,
			Found:    newEnumCaseCount,
			Range:    ast.NewUnmeteredRangeFromPositioned(newDeclaration.DeclarationIdentifier()),
		})

		// If some enum cases are removed, trying to match each enum case
		// may result in too many regression errors.
		// Hence, return.
		return
	}

	// Check whether the enum cases matches the old enum cases.
	for index, newEnumCase := range newEnumCases {
		// If there are no more old enum-cases, then these are newly added enum-cases,
		// which should be fine.
		if index >= oldEnumCaseCount {
			continue
		}

		oldEnumCase := oldEnumCases[index]
		if oldEnumCase.Identifier.Identifier != newEnumCase.Identifier.Identifier {
			validator.report(&EnumCaseMismatchError{
				ExpectedName: oldEnumCase.Identifier.Identifier,
				FoundName:    newEnumCase.Identifier.Identifier,
				Range:        ast.NewUnmeteredRangeFromPositioned(newEnumCase),
			})
		}
	}
}

func (validator *ContractUpdateValidator) checkConformance(
	oldDecl *ast.CompositeDeclaration,
	newDecl *ast.CompositeDeclaration,
) {

	// Here it is assumed enums will always have one and only one conformance.
	// This is enforced by the checker.
	// Therefore, below check for multiple conformances is only applicable
	// for non-enum type composite declarations. i.e: structs, resources, etc.

	oldConformances := oldDecl.Conformances
	newConformances := newDecl.Conformances

	// All the existing conformances must have a match. Order is not important.
	// Having extra new conformance is OK. See: https://github.com/onflow/cadence/issues/1394

	// Note: Removing a conformance is NOT OK. That could lead to type-safety issues.
	// e.g:
	//  - Someone stores an array of type `[{I}]` with `T:I` objects inside.
	//  - Later T’s conformance to `I` is removed.
	//  - Now `[{I}]` contains objects if `T` that does not conform to `I`.

	for _, oldConformance := range oldConformances {
		found := false
		for index, newConformance := range newConformances {
			err := oldConformance.CheckEqual(newConformance, validator)
			if err == nil {
				found = true

				// Remove the matched conformance, so we don't have to check it again.
				// i.e: optimization
				newConformances = append(newConformances[:index], newConformances[index+1:]...)
				break
			}
		}

		if !found {
			oldConformanceID := validator.oldTypeID(oldConformance)

			validator.report(&ConformanceMismatchError{
				DeclName:           newDecl.Identifier.Identifier,
				MissingConformance: string(oldConformanceID),
				Range:              ast.NewUnmeteredRangeFromPositioned(newDecl.Identifier),
			})

			return
		}
	}
}

func (validator *ContractUpdateValidator) report(err error) {
	if err == nil {
		return
	}
	validator.errors = append(validator.errors, err)
}

func (validator *ContractUpdateValidator) getContractUpdateError() error {
	return &ContractUpdateError{
		ContractName: validator.contractName,
		Errors:       validator.errors,
		Location:     validator.location,
	}
}

func containsEnumsInProgram(program *ast.Program) bool {
	declaration, err := getRootDeclarationOfProgram(program)

	if err != nil {
		return false
	}

	return containsEnums(declaration)
}

func containsEnums(declaration ast.Declaration) bool {
	if declaration.DeclarationKind() == common.DeclarationKindEnum {
		return true
	}

	nestedCompositeDecls := declaration.DeclarationMembers().Composites()
	for _, nestedDecl := range nestedCompositeDecls {
		if containsEnums(nestedDecl) {
			return true
		}
	}

	return false
}

// Contract update related errors

// ContractUpdateError is reported upon any invalid update to a contract or contract interface.
// It contains all the errors reported during the update validation.
type ContractUpdateError struct {
	Location     common.Location
	ContractName string
	Errors       []error
}

var _ errors.UserError = &ContractUpdateError{}
var _ errors.ParentError = &ContractUpdateError{}

func (*ContractUpdateError) IsUserError() {}

func (e *ContractUpdateError) Error() string {
	return fmt.Sprintf("cannot update contract `%s`", e.ContractName)
}

func (e *ContractUpdateError) ChildErrors() []error {
	return e.Errors
}

func (e *ContractUpdateError) Unwrap() []error {
	return e.Errors
}

func (e *ContractUpdateError) ImportLocation() common.Location {
	return e.Location
}

// FieldMismatchError is reported during a contract update, when a type of a field
// does not match the existing type of the same field.
type FieldMismatchError struct {
	Err       error
	DeclName  string
	FieldName string
	ast.Range
}

var _ errors.UserError = &FieldMismatchError{}
var _ errors.SecondaryError = &FieldMismatchError{}

func (*FieldMismatchError) IsUserError() {}

func (e *FieldMismatchError) Error() string {
	return fmt.Sprintf("mismatching field `%s` in `%s`",
		e.FieldName,
		e.DeclName,
	)
}

func (e *FieldMismatchError) SecondaryError() string {
	return e.Err.Error()
}

// TypeMismatchError is reported during a contract update, when a type of the new program
// does not match the existing type.
type TypeMismatchError struct {
	ExpectedType ast.Type
	FoundType    ast.Type
	ast.Range
}

var _ errors.UserError = &TypeMismatchError{}

func (*TypeMismatchError) IsUserError() {}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`",
		e.ExpectedType,
		e.FoundType,
	)
}

// ExtraneousFieldError is reported during a contract update, when an updated composite
// declaration has more fields than the existing declaration.
type ExtraneousFieldError struct {
	DeclName  string
	FieldName string
	ast.Range
}

var _ errors.UserError = &ExtraneousFieldError{}

func (*ExtraneousFieldError) IsUserError() {}

func (e *ExtraneousFieldError) Error() string {
	return fmt.Sprintf("found new field `%s` in `%s`",
		e.FieldName,
		e.DeclName,
	)
}

// ContractNotFoundError is reported during a contract update, if no contract can be
// found in the program.
type ContractNotFoundError struct {
	ast.Range
}

var _ errors.UserError = &ContractNotFoundError{}

func (*ContractNotFoundError) IsUserError() {}

func (e *ContractNotFoundError) Error() string {
	return "cannot find any contract or contract interface"
}

// InvalidDeclarationKindChangeError is reported during a contract update, when an attempt is made
// to convert an existing contract to a contract interface, or vise versa.
type InvalidDeclarationKindChangeError struct {
	Name    string
	OldKind common.DeclarationKind
	NewKind common.DeclarationKind
	ast.Range
}

var _ errors.UserError = &InvalidDeclarationKindChangeError{}

func (*InvalidDeclarationKindChangeError) IsUserError() {}

func (e *InvalidDeclarationKindChangeError) Error() string {
	return fmt.Sprintf("trying to convert %s `%s` to a %s", e.OldKind.Name(), e.Name, e.NewKind.Name())
}

// ConformanceMismatchError is reported during a contract update, when the enum conformance of the new program
// does not match the existing one.
type ConformanceMismatchError struct {
	DeclName           string
	MissingConformance string
	ast.Range
}

var _ errors.UserError = &ConformanceMismatchError{}

func (*ConformanceMismatchError) IsUserError() {}

func (e *ConformanceMismatchError) Error() string {
	return fmt.Sprintf(
		"conformances do not match in `%s`: missing `%s`",
		e.DeclName,
		e.MissingConformance,
	)
}

// EnumCaseMismatchError is reported during an enum update, when an updated enum case
// does not match the existing enum case.
type EnumCaseMismatchError struct {
	ExpectedName string
	FoundName    string
	ast.Range
}

var _ errors.UserError = &EnumCaseMismatchError{}

func (*EnumCaseMismatchError) IsUserError() {}

func (e *EnumCaseMismatchError) Error() string {
	return fmt.Sprintf("mismatching enum case: expected `%s`, found `%s`",
		e.ExpectedName,
		e.FoundName,
	)
}

// MissingEnumCasesError is reported during an enum update, if any enum cases are removed
// from an existing enum.
type MissingEnumCasesError struct {
	DeclName string
	Expected int
	Found    int
	ast.Range
}

var _ errors.UserError = &MissingEnumCasesError{}

func (*MissingEnumCasesError) IsUserError() {}

func (e *MissingEnumCasesError) Error() string {
	return fmt.Sprintf(
		"missing cases in enum `%s`: expected %d or more, found %d",
		e.DeclName,
		e.Expected,
		e.Found,
	)
}

// MissingDeclarationError is reported during a contract update,
// if an existing declaration is removed.
type MissingDeclarationError struct {
	Name string
	Kind common.DeclarationKind
	ast.Range
}

var _ errors.UserError = &MissingDeclarationError{}

func (*MissingDeclarationError) IsUserError() {}

func (e *MissingDeclarationError) Error() string {
	return fmt.Sprintf(
		"missing %s declaration `%s`",
		e.Kind.Name(),
		e.Name,
	)
}

// InvalidTypeRemovalPragmaError is reported during a contract update
// if a malformed #removedType pragma is encountered
type InvalidTypeRemovalPragmaError struct {
	Expression ast.Expression
	ast.Range
}

var _ errors.UserError = &InvalidTypeRemovalPragmaError{}

func (*InvalidTypeRemovalPragmaError) IsUserError() {}

func (e *InvalidTypeRemovalPragmaError) Error() string {
	return fmt.Sprintf(
		"invalid #removedType pragma: %s",
		e.Expression.String(),
	)
}

// UseOfRemovedTypeError is reported during a contract update
// if a type is encountered that is also in a #removedType pragma
type UseOfRemovedTypeError struct {
	Declaration ast.Declaration
	ast.Range
}

var _ errors.UserError = &UseOfRemovedTypeError{}

func (*UseOfRemovedTypeError) IsUserError() {}

func (e *UseOfRemovedTypeError) Error() string {
	return fmt.Sprintf(
		"cannot declare %s, type has been removed with a #removedType pragma",
		e.Declaration.DeclarationIdentifier(),
	)
}

// TypeRemovalPragmaRemovalError is reported during a contract update
// if a #removedType pragma is removed
type TypeRemovalPragmaRemovalError struct {
	RemovedType string
}

var _ errors.UserError = &TypeRemovalPragmaRemovalError{}

func (*TypeRemovalPragmaRemovalError) IsUserError() {}

func (e *TypeRemovalPragmaRemovalError) Error() string {
	return fmt.Sprintf(
		"missing #removedType pragma for %s",
		e.RemovedType,
	)
}

// NameMismatchError is reported during a contract update, when an a composite
// declaration has a different name than the existing declaration.
type NameMismatchError struct {
	OldName string
	NewName string
	ast.Range
}

var _ errors.UserError = &NameMismatchError{}

func (*NameMismatchError) IsUserError() {}

func (e *NameMismatchError) Error() string {
	return fmt.Sprintf("name mismatch: got `%s`, expected `%s`",
		e.NewName,
		e.OldName,
	)
}
