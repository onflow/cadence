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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

// VisitInterfaceDeclaration checks the given interface declaration.
//
// NOTE: This function assumes that the interface type was previously declared using
// `declareInterfaceType` and exists in `checker.Elaboration.InterfaceDeclarationTypes`,
// and that the members and nested declarations for the interface type were declared
// through `declareInterfaceMembers`.
func (checker *Checker) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) (_ struct{}) {

	const kind = ContainerKindInterface

	interfaceType := checker.Elaboration.InterfaceDeclarationType(declaration)
	if interfaceType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[interfaceType] = true
	defer func() {
		checker.containerTypes[interfaceType] = false
	}()

	checker.checkDeclarationAccessModifier(
		declaration.Access,
		declaration.DeclarationKind(),
		nil,
		declaration.StartPos,
		true,
	)

	// NOTE: functions are checked separately
	checker.checkFieldsAccessModifier(declaration.Members.Fields(), &declaration.CompositeKind)

	checker.checkNestedIdentifiers(declaration.Members)

	// Activate new scope for nested types

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	// Declare nested types

	checker.declareInterfaceNestedTypes(declaration)

	checker.checkInitializers(
		declaration.Members.Initializers(),
		declaration.Members.Fields(),
		interfaceType,
		declaration.DeclarationDocString(),
		interfaceType.InitializerPurity,
		interfaceType.InitializerParameters,
		kind,
		nil,
	)

	checker.checkUnknownSpecialFunctions(declaration.Members.SpecialFunctions())
	checker.checkSpecialFunctionDefaultImplementation(
		declaration,
		declaration.DeclarationKind().Name(),
	)

	checker.checkInterfaceFunctions(
		declaration.Members.Functions(),
		interfaceType,
		declaration.DeclarationKind(),
		&declaration.CompositeKind,
		declaration.DeclarationDocString(),
	)

	fieldPositionGetter := func(name string) ast.Position {
		return interfaceType.FieldPosition(name, declaration)
	}

	checker.checkResourceFieldNesting(
		interfaceType.Members,
		interfaceType.CompositeKind,
		fieldPositionGetter,
	)

	checker.checkDestructors(
		declaration.Members.Destructors(),
		declaration.Members.FieldsByIdentifier(),
		interfaceType.Members,
		interfaceType,
		declaration.DeclarationKind(),
		declaration.DeclarationDocString(),
		kind,
	)

	// check that members conform to their entitlement declarations, where applicable

	interfaceType.Members.Foreach(func(name string, member *Member) {
		checker.checkMemberEntitlementConformance(interfaceType, member)
	})

	// NOTE: visit entitlements, then interfaces, then composites
	// DON'T use `nestedDeclarations`, because of non-deterministic order

	for _, nestedEntitlement := range declaration.Members.Entitlements() {
		ast.AcceptDeclaration[struct{}](nestedEntitlement, checker)
	}

	for _, nestedInterface := range declaration.Members.Interfaces() {
		ast.AcceptDeclaration[struct{}](nestedInterface, checker)
	}

	for _, nestedComposite := range declaration.Members.Composites() {
		// Composite declarations nested in interface declarations are type requirements,
		// i.e. they should be checked like interfaces

		checker.visitCompositeDeclaration(nestedComposite, kind)
	}

	return
}

// declareInterfaceNestedTypes declares the types nested in an interface.
// It is used when declaring the interface's members (`declareInterfaceMembers`)
// and checking the interface declaration (`VisitInterfaceDeclaration`).
//
// It assumes the types were previously added to the elaboration in `InterfaceNestedDeclarations`,
// and the type for the declaration was added to the elaboration in `InterfaceDeclarationTypes`.
func (checker *Checker) declareInterfaceNestedTypes(
	declaration *ast.InterfaceDeclaration,
) {

	interfaceType := checker.Elaboration.InterfaceDeclarationType(declaration)
	nestedDeclarations := checker.Elaboration.InterfaceNestedDeclarations(declaration)

	interfaceType.NestedTypes.Foreach(func(name string, nestedType Type) {
		nestedDeclaration := nestedDeclarations[name]

		identifier := nestedDeclaration.DeclarationIdentifier()
		if identifier == nil {
			// It should be impossible to have a nested declaration
			// that does not have an identifier

			panic(errors.NewUnreachableError())
		}

		_, err := checker.typeActivations.declareType(typeDeclaration{
			identifier:               *identifier,
			ty:                       nestedType,
			declarationKind:          nestedDeclaration.DeclarationKind(),
			access:                   checker.accessFromAstAccess(nestedDeclaration.DeclarationAccess()),
			docString:                nestedDeclaration.DeclarationDocString(),
			allowOuterScopeShadowing: false,
		})
		checker.report(err)
	})
}

func (checker *Checker) checkInterfaceFunctions(
	functions []*ast.FunctionDeclaration,
	selfType NominalType,
	declarationKind common.DeclarationKind,
	compositeKind *common.CompositeKind,
	selfDocString string,
) {
	for _, function := range functions {
		// NOTE: new activation, as function declarations
		// shouldn't be visible in other function declarations,
		// and `self` is only visible inside function

		func() {
			checker.enterValueScope()
			defer checker.leaveValueScope(function.EndPosition, false)

			checker.declareSelfValue(selfType, selfDocString)

			mustExit := false
			checkResourceLoss := false

			if function.FunctionBlock != nil {
				if function.FunctionBlock.HasStatements() {
					mustExit = true
					checkResourceLoss = true
				} else if function.FunctionBlock.PreConditions.IsEmpty() &&
					function.FunctionBlock.PostConditions.IsEmpty() {

					checker.report(
						&InvalidImplementationError{
							Pos:             function.FunctionBlock.StartPosition(),
							ContainerKind:   declarationKind,
							ImplementedKind: common.DeclarationKindFunction,
						},
					)
				}
			}

			checker.visitFunctionDeclaration(
				function,
				functionDeclarationOptions{
					mustExit:          mustExit,
					declareFunction:   false,
					checkResourceLoss: checkResourceLoss,
				},
				compositeKind,
			)
		}()
	}
}

// declareInterfaceType declares the type for the given interface declaration
// and records it in the elaboration. It also recursively declares all types
// for all nested declarations.
//
// NOTE: The function does *not* declare any members
//
// See `declareInterfaceMembers` for the declaration of the interface type members.
// See `VisitInterfaceDeclaration` for the checking of the interface declaration.
func (checker *Checker) declareInterfaceType(declaration *ast.InterfaceDeclaration) *InterfaceType {

	identifier := declaration.Identifier

	interfaceType := &InterfaceType{
		Location:      checker.Location,
		Identifier:    identifier.Identifier,
		CompositeKind: declaration.CompositeKind,
		NestedTypes:   &StringTypeOrderedMap{},
		Members:       &StringMemberOrderedMap{},
	}

	variable, err := checker.typeActivations.declareType(typeDeclaration{
		identifier:               identifier,
		ty:                       interfaceType,
		declarationKind:          declaration.DeclarationKind(),
		access:                   checker.accessFromAstAccess(declaration.Access),
		docString:                declaration.DocString,
		allowOuterScopeShadowing: false,
	})
	checker.report(err)
	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(
			identifier.Identifier,
			variable,
		)
	}

	checker.Elaboration.SetInterfaceDeclarationType(declaration, interfaceType)
	checker.Elaboration.SetInterfaceTypeDeclaration(interfaceType, declaration)

	if !declaration.CompositeKind.SupportsInterfaces() {
		checker.report(
			&InvalidInterfaceDeclarationError{
				CompositeKind: declaration.CompositeKind,
				Range:         ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Identifier),
			},
		)
	}

	// Activate new scope for nested declarations

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, false)

	// Check and declare nested types

	nestedDeclarations, nestedInterfaceTypes, nestedCompositeTypes, nestedEntitlementTypes :=
		checker.declareNestedDeclarations(
			declaration.CompositeKind,
			declaration.DeclarationKind(),
			declaration.Members.Composites(),
			declaration.Members.Interfaces(),
			declaration.Members.Entitlements(),
		)

	checker.Elaboration.SetInterfaceNestedDeclarations(declaration, nestedDeclarations)

	for _, nestedInterfaceType := range nestedInterfaceTypes {
		interfaceType.NestedTypes.Set(nestedInterfaceType.Identifier, nestedInterfaceType)
		nestedInterfaceType.SetContainerType(interfaceType)
	}

	for _, nestedCompositeType := range nestedCompositeTypes {
		interfaceType.NestedTypes.Set(nestedCompositeType.Identifier, nestedCompositeType)
		nestedCompositeType.SetContainerType(interfaceType)
	}

	for _, nestedEntitlementType := range nestedEntitlementTypes {
		interfaceType.NestedTypes.Set(nestedEntitlementType.Identifier, nestedEntitlementType)
		nestedEntitlementType.SetContainerType(interfaceType)
	}

	return interfaceType
}

// declareInterfaceMembers declares the members for the given interface declaration,
// and recursively for all nested declarations.
//
// NOTE: This function assumes that the interface type and the nested declarations' types
// were previously declared using `declareInterfaceType` and exists
// in the elaboration's `InterfaceDeclarationTypes` and `InterfaceNestedDeclarations` fields.
func (checker *Checker) declareInterfaceMembers(declaration *ast.InterfaceDeclaration) {

	interfaceType := checker.Elaboration.InterfaceDeclarationType(declaration)
	if interfaceType == nil {
		panic(errors.NewUnreachableError())
	}

	// Activate new scope for nested declarations

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, false)

	// Declare nested types

	checker.declareInterfaceNestedTypes(declaration)

	// Declare members

	members, fields, origins := checker.defaultMembersAndOrigins(
		declaration.Members,
		interfaceType,
		ContainerKindInterface,
		declaration.DeclarationKind(),
	)

	if interfaceType.CompositeKind == common.CompositeKindContract {
		checker.checkMemberStorability(members)
	}

	interfaceType.Members = members
	interfaceType.Fields = fields
	if checker.PositionInfo != nil {
		checker.PositionInfo.recordMemberOrigins(interfaceType, origins)
	}

	// NOTE: determine initializer parameter types while nested types are in scope,
	// and after declaring nested types as the initializer may use nested type in parameters

	interfaceType.InitializerParameters =
		checker.initializerParameters(declaration.Members.Initializers())
	interfaceType.InitializerPurity =
		checker.initializerPurity(declaration.Members.Initializers())

	// Declare nested declarations' members

	for _, nestedEntitlementDeclaration := range declaration.Members.Entitlements() {
		checker.declareEntitlementMembers(nestedEntitlementDeclaration)
	}

	for _, nestedInterfaceDeclaration := range declaration.Members.Interfaces() {
		checker.declareInterfaceMembers(nestedInterfaceDeclaration)
	}

	for _, nestedCompositeDeclaration := range declaration.Members.Composites() {
		checker.declareCompositeMembersAndValue(nestedCompositeDeclaration, ContainerKindInterface)
	}
}

func (checker *Checker) declareEntitlementType(declaration *ast.EntitlementDeclaration) *EntitlementType {
	identifier := declaration.Identifier

	entitlementType := &EntitlementType{
		Location:   checker.Location,
		Identifier: identifier.Identifier,
		Members:    &StringMemberOrderedMap{},
	}

	variable, err := checker.typeActivations.declareType(typeDeclaration{
		identifier:               identifier,
		ty:                       entitlementType,
		declarationKind:          declaration.DeclarationKind(),
		access:                   checker.accessFromAstAccess(declaration.Access),
		docString:                declaration.DocString,
		allowOuterScopeShadowing: false,
	})

	checker.report(err)
	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(
			identifier.Identifier,
			variable,
		)
	}

	checker.Elaboration.SetEntitlementDeclarationType(declaration, entitlementType)
	checker.Elaboration.SetEntitlementTypeDeclaration(entitlementType, declaration)

	return entitlementType
}

func (checker *Checker) declareEntitlementMembers(declaration *ast.EntitlementDeclaration) {
	entitlementType := checker.Elaboration.EntitlementDeclarationType(declaration)
	if entitlementType == nil {
		panic(errors.NewUnreachableError())
	}

	fields := declaration.Members.Fields()
	functions := declaration.Members.Functions()

	reportInvalidDeclaration := func(nestedDeclarationKind common.DeclarationKind, identifier ast.Identifier) {
		checker.report(
			&InvalidEntitlementNestedDeclarationError{
				NestedDeclarationKind: nestedDeclarationKind,
				Range:                 ast.NewRangeFromPositioned(checker.memoryGauge, identifier),
			},
		)
	}

	// reject all non-field or function declarations
	for _, nestedDecl := range declaration.Members.Declarations() {
		switch nestedDecl.(type) {
		case *ast.FieldDeclaration, *ast.FunctionDeclaration:
			break
		default:
			reportInvalidDeclaration(nestedDecl.DeclarationKind(), *nestedDecl.DeclarationIdentifier())
		}
	}

	members := &StringMemberOrderedMap{}
	// declare a member for each field
	for _, field := range fields {
		identifier := field.Identifier.Identifier
		fieldTypeAnnotation := checker.ConvertTypeAnnotation(field.TypeAnnotation)
		checker.checkTypeAnnotation(fieldTypeAnnotation, field.TypeAnnotation)
		const declarationKind = common.DeclarationKindField
		if field.Access != ast.AccessNotSpecified {
			checker.report(
				&InvalidEntitlementMemberAccessDeclaration{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, field),
				},
			)
		}

		checker.checkStaticModifier(field.IsStatic(), field.Identifier)
		checker.checkNativeModifier(field.IsNative(), field.Identifier)

		members.Set(
			identifier,
			&Member{
				ContainerType:   entitlementType,
				Access:          checker.accessFromAstAccess(field.Access),
				Identifier:      field.Identifier,
				DeclarationKind: declarationKind,
				TypeAnnotation:  fieldTypeAnnotation,
				VariableKind:    field.VariableKind,
				DocString:       field.DocString,
			})
	}

	// declare a member for each function
	for _, function := range functions {
		identifier := function.Identifier.Identifier
		functionType := checker.functionType(function.Purity, function.ParameterList, function.ReturnTypeAnnotation)
		argumentLabels := function.ParameterList.EffectiveArgumentLabels()
		fieldTypeAnnotation := NewTypeAnnotation(functionType)
		const declarationKind = common.DeclarationKindFunction

		if function.Access != ast.AccessNotSpecified {
			checker.report(
				&InvalidEntitlementMemberAccessDeclaration{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, function),
				},
			)
		}

		if function.FunctionBlock != nil {
			checker.report(
				&InvalidEntitlementFunctionDeclaration{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, function),
				},
			)
		}

		members.Set(
			identifier,
			&Member{
				ContainerType:     entitlementType,
				Access:            checker.accessFromAstAccess(function.Access),
				Identifier:        function.Identifier,
				DeclarationKind:   declarationKind,
				TypeAnnotation:    fieldTypeAnnotation,
				VariableKind:      ast.VariableKindConstant,
				ArgumentLabels:    argumentLabels,
				DocString:         function.DocString,
				HasImplementation: false,
			})
	}

	entitlementType.Members = members
}

func (checker *Checker) checkMemberEntitlementConformance(memberContainer CompositeKindedType, member *Member) {
	entitlementAccess, hasEntitlements := member.Access.(EntitlementAccess)
	if !hasEntitlements {
		return
	}
	entitlements := entitlementAccess.Entitlements

	for _, entitlement := range entitlements {
		entitlementMember, memberPresent := entitlement.Members.Get(member.Identifier.Identifier)
		if !memberPresent {
			checker.report(&EntitlementMemberNotDeclaredError{
				EntitlementType: entitlement,
				MemberContainer: memberContainer,
				Member:          member,
				Range:           ast.NewRangeFromPositioned(checker.memoryGauge, member.Identifier),
			})
			continue
		}
		if !entitlementMember.TypeAnnotation.Type.Equal(member.TypeAnnotation.Type) ||
			(entitlementMember.VariableKind != ast.VariableKindNotSpecified &&
				member.VariableKind != entitlementMember.VariableKind) {
			checker.report(&EntitlementConformanceError{
				EntitlementType: entitlement,
				MemberContainer: memberContainer,
				Member:          member,
				Range:           ast.NewRangeFromPositioned(checker.memoryGauge, member.Identifier),
			})
		}

	}
}

func (checker *Checker) VisitEntitlementDeclaration(declaration *ast.EntitlementDeclaration) (_ struct{}) {
	entitlementType := checker.Elaboration.EntitlementDeclarationType(declaration)
	if entitlementType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.checkDeclarationAccessModifier(
		declaration.Access,
		declaration.DeclarationKind(),
		nil,
		declaration.StartPos,
		true,
	)

	checker.checkNestedIdentifiers(declaration.Members)

	checker.checkInterfaceFunctions(
		declaration.Members.Functions(),
		entitlementType,
		declaration.DeclarationKind(),
		nil,
		declaration.DeclarationDocString(),
	)

	return
}
