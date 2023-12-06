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
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) (_ struct{}) {
	checker.visitCompositeLikeDeclaration(declaration)
	return
}

func (checker *Checker) checkAttachmentBaseType(attachmentType *CompositeType, astBaseType *ast.NominalType) {

	baseType := attachmentType.baseType

	if baseType == nil {
		panic(errors.NewUnreachableError())
	}

	switch ty := baseType.(type) {
	case *InterfaceType:
		if ty.CompositeKind.SupportsAttachments() {
			return
		}

	case *CompositeType:
		if ty.Location == nil {
			break
		}
		if ty.Kind.SupportsAttachments() {
			return
		}

	case *SimpleType:
		switch ty {
		case AnyResourceType, AnyStructType, InvalidType:
			return
		}
	}

	checker.report(&InvalidBaseTypeError{
		BaseType:   baseType,
		Attachment: attachmentType,
		Range:      ast.NewRangeFromPositioned(checker.memoryGauge, astBaseType),
	})
}

func (checker *Checker) checkAttachmentMemberAccess(
	attachmentType *CompositeType,
	member *Member,
	baseType Type,
	supportedBaseEntitlements *EntitlementOrderedSet,
) {
	var requestedEntitlements *EntitlementOrderedSet = &orderedmap.OrderedMap[*EntitlementType, struct{}]{}
	switch memberAccess := member.Access.(type) {
	case EntitlementSetAccess:
		requestedEntitlements = memberAccess.Entitlements
	// if the attachment field/function is declared with mapped access, the domain of the map must be a
	// subset of the supported entitlements on the base. This is because the attachment reference
	// will never be able to possess any entitlements other than these, so any map relations that map
	// from other entitlements will be unreachable
	case *EntitlementMapAccess:
		requestedEntitlements = memberAccess.Domain().Entitlements
	}

	requestedEntitlements.Foreach(func(entitlement *EntitlementType, _ struct{}) {
		if !supportedBaseEntitlements.Contains(entitlement) {
			checker.report(&InvalidAttachmentEntitlementError{
				Attachment:         attachmentType,
				BaseType:           baseType,
				InvalidEntitlement: entitlement,
				Pos:                member.Identifier.Pos,
			})
		}
	})
}

func (checker *Checker) checkAttachmentMembersAccess(attachmentType *CompositeType) {

	// all the access modifiers for attachment members must be valid entitlements for the base type
	var supportedBaseEntitlements *EntitlementOrderedSet = &orderedmap.OrderedMap[*EntitlementType, struct{}]{}
	baseType := attachmentType.GetBaseType()
	switch base := attachmentType.GetBaseType().(type) {
	case EntitlementSupportingType:
		supportedBaseEntitlements = base.SupportedEntitlements()
	}

	attachmentType.EffectiveInterfaceConformanceSet().ForEach(func(intf *InterfaceType) {
		intf.Members.Foreach(func(_ string, member *Member) {
			checker.checkAttachmentMemberAccess(attachmentType, member, baseType, supportedBaseEntitlements)
		})
	})

	attachmentType.Members.Foreach(func(_ string, member *Member) {
		checker.checkAttachmentMemberAccess(attachmentType, member, baseType, supportedBaseEntitlements)
	})

}

func (checker *Checker) VisitAttachmentDeclaration(declaration *ast.AttachmentDeclaration) (_ struct{}) {
	return checker.visitAttachmentDeclaration(declaration)
}

func (checker *Checker) visitAttachmentDeclaration(declaration *ast.AttachmentDeclaration) (_ struct{}) {

	if !checker.Config.AttachmentsEnabled {
		checker.report(&AttachmentsNotEnabledError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, declaration),
		})
	}

	checker.visitCompositeLikeDeclaration(declaration)
	attachmentType := checker.Elaboration.CompositeDeclarationType(declaration)
	checker.checkAttachmentMembersAccess(attachmentType)
	checker.checkAttachmentBaseType(
		attachmentType,
		declaration.BaseType,
	)
	return
}

// visitCompositeDeclaration checks a previously declared composite declaration.
//
// NOTE: This function assumes that the composite type was previously declared using
// `declareCompositeType` and exists in `checker.Elaboration.CompositeDeclarationTypes`,
// and that the members and nested declarations for the composite type were declared
// through `declareCompositeMembersAndValue`.
func (checker *Checker) visitCompositeLikeDeclaration(declaration ast.CompositeLikeDeclaration) {
	compositeType := checker.Elaboration.CompositeDeclarationType(declaration)
	if compositeType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[compositeType] = true
	defer func() {
		checker.containerTypes[compositeType] = false
	}()

	checker.checkDeclarationAccessModifier(
		checker.accessFromAstAccess(declaration.DeclarationAccess()),
		declaration.DeclarationKind(),
		compositeType,
		nil,
		declaration.StartPosition(),
		true,
	)

	members := declaration.DeclarationMembers()

	// NOTE: functions are checked separately
	declarationKind := declaration.Kind()
	checker.checkFieldsAccessModifier(members.Fields(), compositeType.Members, &declarationKind)

	checker.checkNestedIdentifiers(members)

	// Activate new scopes for nested types

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, false)

	checker.declareCompositeLikeNestedTypes(declaration, true)

	var initializationInfo *InitializationInfo
	// The initializer must initialize all members that are fields,
	// e.g. not composite functions (which are by definition constant and "initialized")

	fields := members.Fields()
	fieldMembers := orderedmap.New[MemberFieldDeclarationOrderedMap](len(fields))

	for _, field := range fields {
		fieldName := field.Identifier.Identifier
		member, ok := compositeType.Members.Get(fieldName)
		if !ok {
			continue
		}

		fieldMembers.Set(member, field)
	}

	initializationInfo = NewInitializationInfo(compositeType, fieldMembers)

	checker.checkInitializers(
		members.Initializers(),
		members.Fields(),
		compositeType,
		declaration.DeclarationDocString(),
		compositeType.ConstructorPurity,
		compositeType.ConstructorParameters,
		ContainerKindComposite,
		initializationInfo,
	)

	checker.checkUnknownSpecialFunctions(members.SpecialFunctions())

	checker.checkCompositeFunctions(
		members.Functions(),
		compositeType,
		declaration.DeclarationDocString(),
	)

	fieldPositionGetter := func(name string) ast.Position {
		return compositeType.FieldPosition(name, declaration)
	}

	checker.checkResourceFieldNesting(
		compositeType.Members,
		compositeType.Kind,
		compositeType.baseType,
		fieldPositionGetter,
	)

	// Check conformances

	inheritedMembers := make(map[string][]*Member)

	defaultFunctions := availableDefaultFunctions(compositeType)

	for _, conformance := range compositeType.EffectiveInterfaceConformances() {
		checker.checkCompositeLikeConformance(
			declaration,
			compositeType,
			conformance.InterfaceType,
			conformance.ConformanceChainRoot,
			compositeConformanceCheckOptions{
				checkMissingMembers: true,
			},
			inheritedMembers,
			defaultFunctions,
		)
	}

	if !compositeType.IsResourceType() && compositeType.DefaultDestroyEvent != nil {
		checker.report(&DefaultDestroyEventInNonResourceError{
			Kind:  declaration.DeclarationKind().Name(),
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, declaration),
		})
	}

	// NOTE: visit entitlements, then interfaces, then composites
	// DON'T use `nestedDeclarations`, because of non-deterministic order

	for _, nestedEntitlement := range members.Entitlements() {
		ast.AcceptDeclaration[struct{}](nestedEntitlement, checker)
	}

	for _, nestedEntitlement := range members.EntitlementMaps() {
		ast.AcceptDeclaration[struct{}](nestedEntitlement, checker)
	}

	for _, nestedInterface := range members.Interfaces() {
		ast.AcceptDeclaration[struct{}](nestedInterface, checker)
	}

	for _, nestedComposite := range members.Composites() {
		if compositeType.DefaultDestroyEvent != nil && nestedComposite.IsResourceDestructionDefaultEvent() {
			// we enforce elsewhere that each composite can have only one default destroy event
			checker.checkDefaultDestroyEvent(compositeType.DefaultDestroyEvent, nestedComposite, compositeType, declaration)
		}
		ast.AcceptDeclaration[struct{}](nestedComposite, checker)
	}

	for _, nestedAttachments := range members.Attachments() {
		ast.AcceptDeclaration[struct{}](nestedAttachments, checker)
	}
}

func availableDefaultFunctions(compositeType *CompositeType) map[string]struct{} {
	defaultFunctions := make(map[string]struct{})

	for _, conformance := range compositeType.EffectiveInterfaceConformances() {
		conformance.InterfaceType.Members.Foreach(func(name string, interfaceMember *Member) {
			if interfaceMember.DeclarationKind != common.DeclarationKindFunction {
				return
			}

			if interfaceMember.HasImplementation {
				defaultFunctions[name] = struct{}{}
			}
		})
	}

	return defaultFunctions
}

// declareCompositeNestedTypes declares the types nested in a composite,
// and the constructors for them if `declareConstructors` is true
// and `kind` is `ContainerKindComposite`.
//
// It is used when declaring the composite's members (`declareCompositeMembersAndValue`)
// and checking the composite declaration (`visitCompositeDeclaration`).
//
// It assumes the types were previously added to the elaboration in `CompositeNestedDeclarations`,
// and the type for the declaration was added to the elaboration in `CompositeDeclarationTypes`.
func (checker *Checker) declareCompositeLikeNestedTypes(
	declaration ast.CompositeLikeDeclaration,
	declareConstructors bool,
) {
	compositeType := checker.Elaboration.CompositeDeclarationType(declaration)
	nestedDeclarations := checker.Elaboration.CompositeNestedDeclarations(declaration)

	compositeType.NestedTypes.Foreach(func(name string, nestedType Type) {

		nestedDeclaration := nestedDeclarations[name]

		identifier := nestedDeclaration.DeclarationIdentifier()
		if identifier == nil {
			// It should be impossible to have a nested declaration
			// that does not have an identifier

			panic(errors.NewUnreachableError())
		}

		// NOTE: We allow the shadowing of types here, because the type was already previously
		// declared without allowing shadowing before. This avoids a duplicate error message.

		_, err := checker.typeActivations.declareType(typeDeclaration{
			identifier:               *identifier,
			ty:                       nestedType,
			declarationKind:          nestedDeclaration.DeclarationKind(),
			access:                   checker.accessFromAstAccess(nestedDeclaration.DeclarationAccess()),
			docString:                nestedDeclaration.DeclarationDocString(),
			allowOuterScopeShadowing: true,
		})
		checker.report(err)

		if declareConstructors {

			// NOTE: Re-declare the constructor function for the nested composite declaration:
			// The constructor was previously declared in `declareCompositeMembersAndValue`
			// for this nested declaration, but the value activation for it was only temporary,
			// so that the constructor wouldn't be visible outside of the containing declaration

			nestedCompositeDeclaration, isCompositeDeclaration := nestedDeclaration.(ast.CompositeLikeDeclaration)

			if isCompositeDeclaration {

				nestedCompositeType, ok := nestedType.(*CompositeType)
				if !ok {
					// we just checked that this was a composite declaration
					panic(errors.NewUnreachableError())
				}

				// Always determine composite constructor type

				nestedConstructorType, nestedConstructorArgumentLabels :=
					CompositeLikeConstructorType(checker.Elaboration, nestedCompositeDeclaration, nestedCompositeType)

				switch nestedCompositeType.Kind {
				case common.CompositeKindContract:
					// not supported

				case common.CompositeKindEnum:
					checker.declareEnumConstructor(
						nestedCompositeDeclaration.(*ast.CompositeDeclaration),
						nestedCompositeType,
					)

				default:
					checker.declareCompositeLikeConstructor(
						nestedCompositeDeclaration,
						nestedConstructorType,
						nestedConstructorArgumentLabels,
					)
				}
			}
		}
	})
}

func (checker *Checker) declareNestedDeclarations(
	containerCompositeKind common.CompositeKind,
	containerDeclarationKind common.DeclarationKind,
	nestedCompositeDeclarations []*ast.CompositeDeclaration,
	nestedAttachmentDeclaration []*ast.AttachmentDeclaration,
	nestedInterfaceDeclarations []*ast.InterfaceDeclaration,
	nestedEntitlementDeclarations []*ast.EntitlementDeclaration,
	nestedEntitlementMappingDeclarations []*ast.EntitlementMappingDeclaration,
) (
	nestedDeclarations map[string]ast.Declaration,
	nestedInterfaceTypes []*InterfaceType,
	nestedCompositeTypes []*CompositeType,
	nestedEntitlementTypes []*EntitlementType,
	nestedEntitlementMapTypes []*EntitlementMapType,
) {
	nestedDeclarations = map[string]ast.Declaration{}

	// Only concrete contracts support nested composite declarations
	if containerCompositeKind != common.CompositeKindContract {

		reportInvalidNesting := func(nestedDeclarationKind common.DeclarationKind, identifier ast.Identifier) {
			checker.report(
				&InvalidNestedDeclarationError{
					NestedDeclarationKind:    nestedDeclarationKind,
					ContainerDeclarationKind: containerDeclarationKind,
					Range:                    ast.NewRangeFromPositioned(checker.memoryGauge, identifier),
				},
			)
		}

		if len(nestedCompositeDeclarations) > 0 {

			firstNestedCompositeDeclaration := nestedCompositeDeclarations[0]

			// we want to permit this nesting under 2 conditions: the container is a resource declaration,
			// and this nested declaration is a default destroy event

			if firstNestedCompositeDeclaration.IsResourceDestructionDefaultEvent() {
				if len(nestedCompositeDeclarations) > 1 {
					firstNestedCompositeDeclaration = nestedCompositeDeclarations[1]
					reportInvalidNesting(
						firstNestedCompositeDeclaration.DeclarationKind(),
						firstNestedCompositeDeclaration.Identifier,
					)
				}
			} else {
				reportInvalidNesting(
					firstNestedCompositeDeclaration.DeclarationKind(),
					firstNestedCompositeDeclaration.Identifier,
				)
			}

		} else if len(nestedInterfaceDeclarations) > 0 {

			firstNestedInterfaceDeclaration := nestedInterfaceDeclarations[0]

			reportInvalidNesting(
				firstNestedInterfaceDeclaration.DeclarationKind(),
				firstNestedInterfaceDeclaration.Identifier,
			)
		} else if len(nestedEntitlementDeclarations) > 0 {

			firstNestedEntitlementDeclaration := nestedEntitlementDeclarations[0]

			reportInvalidNesting(
				firstNestedEntitlementDeclaration.DeclarationKind(),
				firstNestedEntitlementDeclaration.Identifier,
			)
		} else if len(nestedEntitlementMappingDeclarations) > 0 {

			firstNestedEntitlementMappingDeclaration := nestedEntitlementMappingDeclarations[0]

			reportInvalidNesting(
				firstNestedEntitlementMappingDeclaration.DeclarationKind(),
				firstNestedEntitlementMappingDeclaration.Identifier,
			)
		} else if len(nestedAttachmentDeclaration) > 0 {

			firstNestedAttachmentDeclaration := nestedAttachmentDeclaration[0]

			reportInvalidNesting(
				firstNestedAttachmentDeclaration.DeclarationKind(),
				firstNestedAttachmentDeclaration.Identifier,
			)

		}

		// NOTE: don't return, so nested declarations / types are still declared
	} else {

		// Check contract's nested composite declarations and interface declarations
		// are a resource (interface) or a struct (interface)

		checkNestedDeclaration := func(
			nestedCompositeKind common.CompositeKind,
			nestedDeclarationKind common.DeclarationKind,
			identifier ast.Identifier,
		) {
			if containerDeclarationKind.IsInterfaceDeclaration() && !nestedDeclarationKind.IsInterfaceDeclaration() {
				switch nestedCompositeKind {
				case common.CompositeKindEvent:
					break

				default:
					checker.report(
						&InvalidNestedDeclarationError{
							NestedDeclarationKind:    nestedDeclarationKind,
							ContainerDeclarationKind: containerDeclarationKind,
							Range:                    ast.NewRangeFromPositioned(checker.memoryGauge, identifier),
						},
					)
				}
			} else {
				switch nestedCompositeKind {
				case common.CompositeKindResource,
					common.CompositeKindStructure,
					common.CompositeKindAttachment,
					common.CompositeKindEvent,
					common.CompositeKindEnum:
					break

				default:
					checker.report(
						&InvalidNestedDeclarationError{
							NestedDeclarationKind:    nestedDeclarationKind,
							ContainerDeclarationKind: containerDeclarationKind,
							Range:                    ast.NewRangeFromPositioned(checker.memoryGauge, identifier),
						},
					)
				}
			}
		}

		for _, nestedDeclaration := range nestedInterfaceDeclarations {
			checkNestedDeclaration(
				nestedDeclaration.CompositeKind,
				nestedDeclaration.DeclarationKind(),
				nestedDeclaration.Identifier,
			)
		}

		for _, nestedDeclaration := range nestedCompositeDeclarations {
			checkNestedDeclaration(
				nestedDeclaration.CompositeKind,
				nestedDeclaration.DeclarationKind(),
				nestedDeclaration.Identifier,
			)
		}

		for _, nestedDeclaration := range nestedAttachmentDeclaration {
			checkNestedDeclaration(
				common.CompositeKindAttachment,
				nestedDeclaration.DeclarationKind(),
				nestedDeclaration.Identifier,
			)
		}

		// NOTE: don't return, so nested declarations / types are still declared
	}

	// Declare nested entitlements

	for _, nestedDeclaration := range nestedEntitlementDeclarations {
		if _, exists := nestedDeclarations[nestedDeclaration.Identifier.Identifier]; !exists {
			nestedDeclarations[nestedDeclaration.Identifier.Identifier] = nestedDeclaration
		}

		nestedEntitlementType := checker.declareEntitlementType(nestedDeclaration)
		nestedEntitlementTypes = append(nestedEntitlementTypes, nestedEntitlementType)
	}

	// Declare nested entitlement mappings

	for _, nestedDeclaration := range nestedEntitlementMappingDeclarations {
		if _, exists := nestedDeclarations[nestedDeclaration.Identifier.Identifier]; !exists {
			nestedDeclarations[nestedDeclaration.Identifier.Identifier] = nestedDeclaration
		}

		nestedEntitlementMapType := checker.declareEntitlementMappingType(nestedDeclaration)
		nestedEntitlementMapTypes = append(nestedEntitlementMapTypes, nestedEntitlementMapType)
	}

	// Declare nested interfaces

	for _, nestedDeclaration := range nestedInterfaceDeclarations {
		identifier := nestedDeclaration.Identifier.Identifier
		if _, exists := nestedDeclarations[identifier]; !exists {
			nestedDeclarations[identifier] = nestedDeclaration
		}

		nestedInterfaceType := checker.declareInterfaceType(nestedDeclaration)
		nestedInterfaceTypes = append(nestedInterfaceTypes, nestedInterfaceType)
	}

	// Declare nested composites

	for _, nestedDeclaration := range nestedCompositeDeclarations {
		identifier := nestedDeclaration.Identifier.Identifier
		if _, exists := nestedDeclarations[identifier]; !exists {
			nestedDeclarations[identifier] = nestedDeclaration
		}

		nestedCompositeType := checker.declareCompositeType(nestedDeclaration)
		nestedCompositeTypes = append(nestedCompositeTypes, nestedCompositeType)
	}

	// Declare nested attachments

	for _, nestedDeclaration := range nestedAttachmentDeclaration {
		identifier := nestedDeclaration.Identifier.Identifier
		if _, exists := nestedDeclarations[identifier]; !exists {
			nestedDeclarations[identifier] = nestedDeclaration
		}

		nestedCompositeType := checker.declareAttachmentType(nestedDeclaration)
		nestedCompositeTypes = append(nestedCompositeTypes, nestedCompositeType)

	}

	return
}

func (checker *Checker) declareAttachmentType(declaration *ast.AttachmentDeclaration) *CompositeType {

	composite := checker.declareCompositeType(declaration)
	composite.baseType = checker.convertNominalType(declaration.BaseType)
	return composite
}

// declareCompositeType declares the type for the given composite declaration
// and records it in the elaboration. It also recursively declares all types
// for all nested declarations.
//
// NOTE: The function does *not* declare any members or nested declarations.
//
// See `declareCompositeMembersAndValue` for the declaration of the composite type members.
// See `visitCompositeDeclaration` for the checking of the composite declaration.
func (checker *Checker) declareCompositeType(declaration ast.CompositeLikeDeclaration) *CompositeType {

	identifier := *declaration.DeclarationIdentifier()

	compositeType := &CompositeType{
		Location:    checker.Location,
		Kind:        declaration.Kind(),
		Identifier:  identifier.Identifier,
		NestedTypes: &StringTypeOrderedMap{},
		Members:     &StringMemberOrderedMap{},
	}

	variable, err := checker.typeActivations.declareType(typeDeclaration{
		identifier:               identifier,
		ty:                       compositeType,
		declarationKind:          declaration.DeclarationKind(),
		access:                   checker.accessFromAstAccess(declaration.DeclarationAccess()),
		docString:                declaration.DeclarationDocString(),
		allowOuterScopeShadowing: false,
	})
	checker.report(err)

	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(
			identifier.Identifier,
			variable,
		)
	}

	// Resolve conformances

	if declaration.Kind() == common.CompositeKindEnum {
		compositeType.EnumRawType = checker.enumRawType(declaration.(*ast.CompositeDeclaration))
	} else {
		compositeType.ExplicitInterfaceConformances =
			checker.explicitInterfaceConformances(declaration, compositeType)
	}

	// Register in elaboration

	checker.Elaboration.SetCompositeDeclarationType(declaration, compositeType)
	checker.Elaboration.SetCompositeTypeDeclaration(compositeType, declaration)

	// Activate new scope for nested declarations

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, false)

	members := declaration.DeclarationMembers()

	// Check and declare nested types

	nestedDeclarations, nestedInterfaceTypes, nestedCompositeTypes, nestedEntitlementTypes, nestedEntitlementMapTypes :=
		checker.declareNestedDeclarations(
			declaration.Kind(),
			declaration.DeclarationKind(),
			members.Composites(),
			members.Attachments(),
			members.Interfaces(),
			members.Entitlements(),
			members.EntitlementMaps(),
		)

	checker.Elaboration.SetCompositeNestedDeclarations(declaration, nestedDeclarations)

	for _, nestedEntitlementType := range nestedEntitlementTypes {
		compositeType.SetNestedType(
			nestedEntitlementType.Identifier,
			nestedEntitlementType,
		)
	}

	for _, nestedEntitlementMapType := range nestedEntitlementMapTypes {
		compositeType.SetNestedType(
			nestedEntitlementMapType.Identifier,
			nestedEntitlementMapType,
		)
	}

	for _, nestedInterfaceType := range nestedInterfaceTypes {
		compositeType.SetNestedType(
			nestedInterfaceType.Identifier,
			nestedInterfaceType,
		)
	}

	for _, nestedCompositeType := range nestedCompositeTypes {
		compositeType.SetNestedType(
			nestedCompositeType.Identifier,
			nestedCompositeType,
		)
	}

	return compositeType
}

func (checker *Checker) declareAttachmentMembersAndValue(declaration *ast.AttachmentDeclaration) {
	checker.declareCompositeLikeMembersAndValue(declaration)
}

// declareCompositeMembersAndValue declares the members and the value
// (e.g. constructor function for non-contract types; instance for contracts)
// for the given composite declaration, and recursively for all nested declarations.
//
// NOTE: This function assumes that the composite type was previously declared using
// `declareCompositeType` and exists in `checker.Elaboration.CompositeDeclarationTypes`.
func (checker *Checker) declareCompositeLikeMembersAndValue(
	declaration ast.CompositeLikeDeclaration,
) {
	compositeType := checker.Elaboration.CompositeDeclarationType(declaration)
	if compositeType == nil {
		panic(errors.NewUnreachableError())
	}

	compositeKind := declaration.Kind()

	members := declaration.DeclarationMembers()

	nestedComposites := members.Composites()
	nestedAttachments := members.Attachments()
	declarationMembers := orderedmap.New[StringMemberOrderedMap](len(nestedComposites) + len(nestedAttachments))

	(func() {
		// Activate new scopes for nested types

		checker.typeActivations.Enter()
		defer checker.typeActivations.Leave(declaration.EndPosition)

		checker.enterValueScope()
		defer checker.leaveValueScope(declaration.EndPosition, false)

		checker.declareCompositeLikeNestedTypes(declaration, false)

		// NOTE: determine initializer parameter types while nested types are in scope,
		// and after declaring nested types as the initializer may use nested type in parameters

		initializers := members.Initializers()
		compositeType.ConstructorParameters = checker.initializerParameters(initializers)
		compositeType.ConstructorPurity = checker.initializerPurity(compositeKind, initializers)

		// Declare nested declarations' members

		for _, nestedInterfaceDeclaration := range members.Interfaces() {
			checker.declareInterfaceMembersAndValue(nestedInterfaceDeclaration)
		}

		// If this composite declaration has nested composite declaration,
		// then recursively declare the members and values of them.
		//
		// For instance, a structure `S`, defined within a contract `MyContract`,
		// as shown in the example code below, is a nested composite declaration
		// which has its own members:
		// ```
		// contract MyContract {
		//   struct S {
		//     var v: Int
		//   }
		// }
		// ```
		declareNestedComposite := func(nestedCompositeDeclaration ast.CompositeLikeDeclaration) {
			checker.declareCompositeLikeMembersAndValue(nestedCompositeDeclaration)

			// Declare nested composites' values (constructor/instance) as members of the containing composite

			identifier := *nestedCompositeDeclaration.DeclarationIdentifier()

			// Find the value declaration
			nestedCompositeDeclarationVariable :=
				checker.valueActivations.Find(identifier.Identifier)

			if ast.IsResourceDestructionDefaultEvent(identifier.Identifier) {
				// Find the default event's type declaration
				checker.Elaboration.SetDefaultDestroyDeclaration(declaration, nestedCompositeDeclaration)
				defaultEventType :=
					checker.typeActivations.Find(identifier.Identifier)
				defaultEventComposite, ok := defaultEventType.Type.(*CompositeType)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				compositeType.DefaultDestroyEvent = defaultEventComposite
			}

			declarationMembers.Set(
				nestedCompositeDeclarationVariable.Identifier,
				&Member{
					Identifier:            identifier,
					Access:                checker.accessFromAstAccess(nestedCompositeDeclaration.DeclarationAccess()),
					ContainerType:         compositeType,
					TypeAnnotation:        NewTypeAnnotation(nestedCompositeDeclarationVariable.Type),
					DeclarationKind:       nestedCompositeDeclarationVariable.DeclarationKind,
					VariableKind:          ast.VariableKindConstant,
					ArgumentLabels:        nestedCompositeDeclarationVariable.ArgumentLabels,
					IgnoreInSerialization: true,
					DocString:             nestedCompositeDeclaration.DeclarationDocString(),
				},
			)
		}
		for _, nestedInterfaceDeclaration := range members.Interfaces() {
			// resolve conformances
			nestedInterfaceType := checker.Elaboration.InterfaceDeclarationType(nestedInterfaceDeclaration)
			nestedInterfaceType.ExplicitInterfaceConformances =
				checker.explicitInterfaceConformances(nestedInterfaceDeclaration, nestedInterfaceType)
		}
		for _, nestedCompositeDeclaration := range nestedComposites {
			declareNestedComposite(nestedCompositeDeclaration)
		}
		for _, nestedAttachmentDeclaration := range nestedAttachments {
			declareNestedComposite(nestedAttachmentDeclaration)
		}

		compositeType.GetNestedTypes().Foreach(func(nestedTypeIdentifier string, nestedType Type) {

			nestedCompositeType, ok := nestedType.(*CompositeType)
			if !ok {
				return
			}

			var inheritedMembers StringMemberOrderedMap

			inheritedMembers.Foreach(func(memberName string, member *Member) {
				inheritedMember := *member
				inheritedMember.ContainerType = nestedCompositeType
				nestedCompositeType.Members.Set(memberName, &inheritedMember)
			})
		})

		// Declare members
		// NOTE: *After* declaring nested composite and interface declarations

		var members *StringMemberOrderedMap
		var fields []string
		var origins map[string]*Origin

		switch compositeKind {
		case common.CompositeKindEvent:
			// Event members are derived from the initializer's parameter list
			members, fields, origins = checker.eventMembersAndOrigins(
				initializers[0],
				compositeType,
			)

		case common.CompositeKindEnum:
			// Enum members are derived from the cases
			members, fields, origins = checker.enumMembersAndOrigins(
				declaration.DeclarationMembers(),
				compositeType,
				declaration.DeclarationKind(),
			)

		default:
			members, fields, origins = checker.defaultMembersAndOrigins(
				declaration.DeclarationMembers(),
				compositeType,
				ContainerKindComposite,
				declaration.DeclarationKind(),
			)
		}

		if compositeType.Kind == common.CompositeKindContract {
			checker.checkMemberStorability(members)
		}

		compositeType.Members = members
		compositeType.Fields = fields
		if checker.PositionInfo != nil {
			checker.PositionInfo.recordMemberOrigins(compositeType, origins)
		}
	})()

	// Always determine composite constructor type

	constructorType, constructorArgumentLabels := CompositeLikeConstructorType(checker.Elaboration, declaration, compositeType)
	constructorType.Members = declarationMembers

	// If the composite is a contract,
	// declare a value – the contract is a singleton.
	//
	// If the composite is an enum,
	// declare a special constructor which accepts the raw value,
	// and declare the enum cases as members on the constructor.
	//
	// For all other kinds, declare constructor.

	// NOTE: perform declarations after the nested scope, so they are visible after the declaration

	switch compositeType.Kind {
	case common.CompositeKindContract:
		checker.declareContractValue(
			declaration.(*ast.CompositeDeclaration),
			compositeType,
			declarationMembers,
		)

	case common.CompositeKindEnum:
		checker.declareEnumConstructor(
			declaration.(*ast.CompositeDeclaration),
			compositeType,
		)

	default:
		checker.declareCompositeLikeConstructor(
			declaration,
			constructorType,
			constructorArgumentLabels,
		)
	}
}

func (checker *Checker) declareCompositeLikeConstructor(
	declaration ast.CompositeLikeDeclaration,
	constructorType *FunctionType,
	constructorArgumentLabels []string,
) {
	// Resource and event constructors are effectively always private,
	// i.e. they should be only constructable by the locations that declare them.
	//
	// Instead of enforcing this by declaring the access as private here,
	// we allow the declared access level and check the construction in the respective
	// construction expressions, i.e. create expressions for resources
	// and emit statements for events.
	//
	// This improves the user experience for the developer:
	// If the access would be enforced as private, an import of the composite
	// would fail with an "not declared" error.

	_, err := checker.valueActivations.declare(variableDeclaration{
		identifier:               declaration.DeclarationIdentifier().Identifier,
		ty:                       constructorType,
		docString:                declaration.DeclarationDocString(),
		access:                   checker.accessFromAstAccess(declaration.DeclarationAccess()),
		kind:                     declaration.DeclarationKind(),
		pos:                      declaration.DeclarationIdentifier().Pos,
		isConstant:               true,
		argumentLabels:           constructorArgumentLabels,
		allowOuterScopeShadowing: false,
	})
	checker.report(err)
}

func (checker *Checker) declareContractValue(
	declaration *ast.CompositeDeclaration,
	compositeType *CompositeType,
	declarationMembers *StringMemberOrderedMap,
) {
	contractValueHandler := checker.Config.ContractValueHandler

	if contractValueHandler != nil {
		valueDeclaration := contractValueHandler(checker, declaration, compositeType)
		_, err := checker.valueActivations.DeclareValue(valueDeclaration)
		checker.report(err)
	} else {
		_, err := checker.valueActivations.declare(variableDeclaration{
			identifier: declaration.Identifier.Identifier,
			ty:         compositeType,
			docString:  declaration.DocString,
			// NOTE: contracts are always public
			access:     PrimitiveAccess(ast.AccessAll),
			kind:       common.DeclarationKindContract,
			pos:        declaration.Identifier.Pos,
			isConstant: true,
		})
		checker.report(err)
	}

	declarationMembers.Foreach(func(name string, declarationMember *Member) {
		if compositeType.Members.Contains(name) {
			return
		}
		compositeType.Members.Set(name, declarationMember)
	})
}

func (checker *Checker) declareEnumConstructor(
	declaration *ast.CompositeDeclaration,
	compositeType *CompositeType,
) {

	enumCases := declaration.Members.EnumCases()

	var constructorOrigins map[string]*Origin

	if checker.PositionInfo != nil {
		constructorOrigins = make(map[string]*Origin, len(enumCases))
	}

	constructorType := EnumConstructorType(compositeType)

	memberCaseTypeAnnotation := NewTypeAnnotation(compositeType)

	for _, enumCase := range enumCases {
		caseName := enumCase.Identifier.Identifier

		if constructorType.Members.Contains(caseName) {
			continue
		}

		constructorType.Members.Set(
			caseName,
			&Member{
				ContainerType: constructorType,
				// enum cases are always public
				Access:          PrimitiveAccess(ast.AccessAll),
				Identifier:      enumCase.Identifier,
				TypeAnnotation:  memberCaseTypeAnnotation,
				DeclarationKind: common.DeclarationKindField,
				VariableKind:    ast.VariableKindConstant,
				DocString:       enumCase.DocString,
			})

		if checker.PositionInfo != nil && constructorOrigins != nil {
			constructorOrigins[caseName] =
				checker.recordFieldDeclarationOrigin(
					enumCase.Identifier,
					compositeType,
					enumCase.DocString,
				)
		}
	}

	if checker.PositionInfo != nil {
		checker.PositionInfo.recordMemberOrigins(constructorType, constructorOrigins)
	}

	_, err := checker.valueActivations.declare(variableDeclaration{
		identifier: declaration.Identifier.Identifier,
		ty:         constructorType,
		docString:  declaration.DocString,
		// NOTE: enums are always public
		access:         PrimitiveAccess(ast.AccessAll),
		kind:           common.DeclarationKindEnum,
		pos:            declaration.Identifier.Pos,
		isConstant:     true,
		argumentLabels: []string{EnumRawValueFieldName},
	})
	checker.report(err)
}

func EnumConstructorType(compositeType *CompositeType) *FunctionType {
	return &FunctionType{
		Purity:        FunctionPurityView,
		IsConstructor: true,
		Parameters: []Parameter{
			{
				Identifier:     EnumRawValueFieldName,
				TypeAnnotation: NewTypeAnnotation(compositeType.EnumRawType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: compositeType,
			},
		),
		Members: &StringMemberOrderedMap{},
	}
}

// checkMemberStorability check that all fields have a type that is storable.
func (checker *Checker) checkMemberStorability(members *StringMemberOrderedMap) {

	storableResults := map[*Member]bool{}

	members.Foreach(func(_ string, member *Member) {

		if member.IsStorable(storableResults) {
			return
		}

		checker.report(
			&FieldTypeNotStorableError{
				Name: member.Identifier.Identifier,
				Type: member.TypeAnnotation.Type,
				Pos:  member.Identifier.Pos,
			},
		)
	})
}

func (checker *Checker) initializerPurity(
	compositeKind common.CompositeKind,
	initializers []*ast.SpecialFunctionDeclaration,
) FunctionPurity {
	if compositeKind == common.CompositeKindEvent {
		return FunctionPurityView
	}

	// TODO: support multiple overloaded initializers
	initializerCount := len(initializers)
	if initializerCount > 0 {
		firstInitializer := initializers[0]
		return PurityFromAnnotation(firstInitializer.FunctionDeclaration.Purity)
	}

	// a composite with no initializer is view because it runs no code
	return FunctionPurityView
}

func (checker *Checker) initializerParameters(initializers []*ast.SpecialFunctionDeclaration) []Parameter {
	// TODO: support multiple overloaded initializers
	var parameters []Parameter

	initializerCount := len(initializers)
	if initializerCount > 0 {

		firstInitializer := initializers[0]

		parameters = checker.parameters(firstInitializer.FunctionDeclaration.ParameterList)

		if initializerCount > 1 {

			secondInitializer := initializers[1]

			previousPos := firstInitializer.StartPosition()
			pos := secondInitializer.StartPosition()

			checker.report(
				&RedeclarationError{
					Kind:        common.DeclarationKindInitializer,
					Name:        common.DeclarationKindInitializer.Keywords(),
					PreviousPos: &previousPos,
					Pos:         pos,
				},
			)

		}
	}
	return parameters
}

func (checker *Checker) explicitInterfaceConformances(
	conformingDeclaration ast.ConformingDeclaration,
	compositeKindedType CompositeKindedType,
) []*InterfaceType {

	var interfaceTypes []*InterfaceType
	seenConformances := map[*InterfaceType]bool{}

	for _, conformance := range conformingDeclaration.ConformanceList() {
		convertedType := checker.ConvertType(conformance)

		if interfaceType, ok := convertedType.(*InterfaceType); ok {
			interfaceTypes = append(interfaceTypes, interfaceType)

			if seenConformances[interfaceType] {
				checker.report(
					&DuplicateConformanceError{
						CompositeKindedType: compositeKindedType,
						InterfaceType:       interfaceType,
						Range:               ast.NewRangeFromPositioned(checker.memoryGauge, conformance.Identifier),
					},
				)
			}

			seenConformances[interfaceType] = true

		} else if !convertedType.IsInvalidType() {
			checker.report(
				&InvalidConformanceError{
					Type:  convertedType,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, conformance),
				},
			)
		}
	}

	return interfaceTypes
}

func (checker *Checker) enumRawType(declaration *ast.CompositeDeclaration) Type {

	conformanceCount := len(declaration.Conformances)

	// Enums must have exactly one conformance, the raw type

	if conformanceCount == 0 {
		checker.report(
			&MissingEnumRawTypeError{
				Pos: declaration.Identifier.EndPosition(checker.memoryGauge).Shifted(checker.memoryGauge, 1),
			},
		)

		return InvalidType
	}

	// Enums may not conform to interfaces,
	// i.e. only have one conformance, the raw type

	if conformanceCount > 1 {
		secondConformance := declaration.Conformances[1]
		lastConformance := declaration.Conformances[conformanceCount-1]

		checker.report(
			&InvalidEnumConformancesError{
				Range: ast.NewRange(
					checker.memoryGauge,
					secondConformance.StartPosition(),
					lastConformance.EndPosition(checker.memoryGauge),
				),
			},
		)

		// NOTE: do not return, the first conformance should
		// still be considered as a raw type
	}

	// The single conformance is considered the raw type.
	// It must be an `Integer`-subtype for now.

	conformance := declaration.Conformances[0]
	rawType := checker.ConvertType(conformance)

	if !rawType.IsInvalidType() &&
		!IsSameTypeKind(rawType, IntegerType) {

		checker.report(
			&InvalidEnumRawTypeError{
				Type:  rawType,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, conformance),
			},
		)
	}

	return rawType
}

type compositeConformanceCheckOptions struct {
	checkMissingMembers bool
}

// checkCompositeLikeConformance checks if the given composite declaration with the given composite type
// conforms to the specified interface type.
//
// inheritedMembers is an "input/output parameter":
// It tracks which members were inherited from the interface.
// It allows tracking this across conformance checks of multiple interfaces.
func (checker *Checker) checkCompositeLikeConformance(
	compositeDeclaration ast.CompositeLikeDeclaration,
	compositeType *CompositeType,
	conformance *InterfaceType,
	conformanceChainRoot *InterfaceType,
	options compositeConformanceCheckOptions,
	inheritedMembers map[string][]*Member,
	defaultFunctions map[string]struct{},
) {

	var missingMembers []*Member
	var memberMismatches []MemberMismatch
	var missingNestedCompositeTypes []*CompositeType
	var initializerMismatch *InitializerMismatch

	// Ensure the composite kinds match, e.g. a structure shouldn't be able
	// to conform to a resource interface
	checker.checkConformanceKindMatch(compositeDeclaration, compositeType, conformance)

	// Check initializer requirement

	// TODO: add support for overloaded initializers

	if conformance.InitializerParameters != nil {

		initializerType := NewSimpleFunctionType(
			compositeType.ConstructorPurity,
			compositeType.ConstructorParameters,
			VoidTypeAnnotation,
		)
		interfaceInitializerType := NewSimpleFunctionType(
			conformance.InitializerPurity,
			conformance.InitializerParameters,
			VoidTypeAnnotation,
		)

		// TODO: subtype?
		if !initializerType.Equal(interfaceInitializerType) {
			initializerMismatch = &InitializerMismatch{
				CompositePurity:     compositeType.ConstructorPurity,
				InterfacePurity:     conformance.InitializerPurity,
				CompositeParameters: compositeType.ConstructorParameters,
				InterfaceParameters: conformance.InitializerParameters,
			}
		}
	}

	// Determine missing members and member conformance

	conformance.Members.Foreach(func(name string, interfaceMember *Member) {

		// Conforming types do not provide a concrete member
		// for the member in the interface if it is predeclared

		if interfaceMember.Predeclared {
			return
		}

		compositeMember, ok := compositeType.Members.Get(name)
		if ok {

			// If the composite member exists, check if it satisfies the mem

			if !checker.memberSatisfied(compositeType, compositeMember, interfaceMember) {
				memberMismatches = append(
					memberMismatches,
					MemberMismatch{
						CompositeMember: compositeMember,
						InterfaceMember: interfaceMember,
					},
				)
			}

		} else if options.checkMissingMembers {

			// If the composite member does not exist, the interface may provide a default function.
			// However, only one of the composite's conformances (interfaces)
			// may provide a default function.

			if interfaceMember.DeclarationKind == common.DeclarationKindFunction {
				existingMembers, ok := inheritedMembers[name]
				if ok {
					hasConflicts := checker.checkMemberConflicts(
						compositeDeclaration,
						existingMembers,
						interfaceMember,
						compositeType,
					)

					if hasConflicts {
						return
					}
				}

				existingMembers = append(existingMembers, interfaceMember)
				inheritedMembers[name] = existingMembers

			}

			if _, ok := defaultFunctions[name]; !ok {
				missingMembers = append(missingMembers, interfaceMember)
			}
		}

	})

	if len(missingMembers) > 0 ||
		len(memberMismatches) > 0 ||
		len(missingNestedCompositeTypes) > 0 ||
		initializerMismatch != nil {

		checker.report(
			&ConformanceError{
				CompositeDeclaration:        compositeDeclaration,
				CompositeType:               compositeType,
				InterfaceType:               conformanceChainRoot,
				Pos:                         compositeDeclaration.DeclarationIdentifier().Pos,
				InitializerMismatch:         initializerMismatch,
				MissingMembers:              missingMembers,
				MemberMismatches:            memberMismatches,
				MissingNestedCompositeTypes: missingNestedCompositeTypes,
				NestedInterfaceType:         conformance,
			},
		)
	}

}

func (checker *Checker) checkMemberConflicts(
	compositeDeclaration ast.CompositeLikeDeclaration,
	existingMembers []*Member,
	newMember *Member,
	compositeType *CompositeType,
) (hasConflicts bool) {

	errorRange := ast.NewRangeFromPositioned(checker.memoryGauge, compositeDeclaration.DeclarationIdentifier())

	for _, existingMember := range existingMembers {

		// Both have default impls. That's an error.
		if newMember.HasImplementation && existingMember.HasImplementation {
			checker.report(
				&MultipleInterfaceDefaultImplementationsError{
					CompositeKindedType: compositeType,
					Member:              newMember,
					Range:               errorRange,
				},
			)

			return true
		}
	}

	return false
}

// checkConformanceKindMatch ensures the composite kinds match.
// e.g. a structure shouldn't be able to conform to a resource interface.
func (checker *Checker) checkConformanceKindMatch(
	conformingDeclaration ast.ConformingDeclaration,
	compositeKindedType CompositeKindedType,
	interfaceConformance *InterfaceType,
) {

	// Check if the conformance kind matches the declaration type's kind.
	if interfaceConformance.CompositeKind == compositeKindedType.GetCompositeKind() {
		return
	}

	// For attachments, check if the conformance kind matches the base type's kind.
	if compositeType, ok := compositeKindedType.(*CompositeType); ok &&
		interfaceConformance.CompositeKind == compositeType.getBaseCompositeKind() {
		return
	}

	// If not a match, then report an error.

	var compositeKindMismatchIdentifier *ast.Identifier

	conformances := conformingDeclaration.ConformanceList()

	if len(conformances) == 0 {
		// If there are no explicit conformances, log the error at the declaration identifier
		compositeKindMismatchIdentifier = conformingDeclaration.DeclarationIdentifier()
	} else {
		// Otherwise, find the conformance which resulted in the mismatch,
		// and log the error there.
		for _, conformance := range conformances {
			if conformance.Identifier.Identifier == interfaceConformance.Identifier {
				compositeKindMismatchIdentifier = &conformance.Identifier
				break
			}
		}

		// If not found, then that means, the mismatching interface is a grandparent.
		// Then it should have already been reported when checking the parent.
		// Hence, no need to report an error here again.
	}

	if compositeKindMismatchIdentifier == nil {
		return
	}

	checker.report(
		&CompositeKindMismatchError{
			ExpectedKind: compositeKindedType.GetCompositeKind(),
			ActualKind:   interfaceConformance.CompositeKind,
			Range:        ast.NewRangeFromPositioned(checker.memoryGauge, compositeKindMismatchIdentifier),
		},
	)
}

// TODO: return proper error
func (checker *Checker) memberSatisfied(
	compositeKindedType CompositeKindedType,
	compositeMember, interfaceMember *Member,
) bool {

	// Check declaration kind
	if compositeMember.DeclarationKind != interfaceMember.DeclarationKind {
		return false
	}

	// Check type

	compositeMemberType := compositeMember.TypeAnnotation.Type
	interfaceMemberType := interfaceMember.TypeAnnotation.Type

	if !compositeMemberType.IsInvalidType() &&
		!interfaceMemberType.IsInvalidType() {

		switch interfaceMember.DeclarationKind {
		case common.DeclarationKindField:
			// If the member is just a field, check the types are equal

			// TODO: subtype?
			if !compositeMemberType.Equal(interfaceMemberType) {
				return false
			}

		case common.DeclarationKindFunction:
			// If the member is a function, check that the argument labels are equal,
			// the parameter types are equal (they are invariant),
			// and that the return types are subtypes (the return type is covariant).
			//
			// This is different from subtyping for functions,
			// where argument labels are not considered,
			// and parameters are contravariant.

			interfaceMemberFunctionType, isInterfaceMemberFunctionType := interfaceMemberType.(*FunctionType)
			compositeMemberFunctionType, isCompositeMemberFunctionType := compositeMemberType.(*FunctionType)

			if !isInterfaceMemberFunctionType || !isCompositeMemberFunctionType {
				return false
			}

			if !interfaceMemberFunctionType.HasSameArgumentLabels(compositeMemberFunctionType) {
				return false
			}

			// Functions are covariant in their purity
			if compositeMemberFunctionType.Purity != interfaceMemberFunctionType.Purity &&
				compositeMemberFunctionType.Purity != FunctionPurityView {

				return false
			}

			// Functions are invariant in their parameter types

			for i, subParameter := range compositeMemberFunctionType.Parameters {
				superParameter := interfaceMemberFunctionType.Parameters[i]
				if !subParameter.TypeAnnotation.Type.
					Equal(superParameter.TypeAnnotation.Type) {

					return false
				}
			}

			// Functions are covariant in their return type

			if compositeMemberFunctionType.ReturnTypeAnnotation.Type != nil &&
				interfaceMemberFunctionType.ReturnTypeAnnotation.Type != nil {

				if !IsSubType(
					compositeMemberFunctionType.ReturnTypeAnnotation.Type,
					interfaceMemberFunctionType.ReturnTypeAnnotation.Type,
				) {
					return false
				}
			}

			if (compositeMemberFunctionType.ReturnTypeAnnotation.Type != nil &&
				interfaceMemberFunctionType.ReturnTypeAnnotation.Type == nil) ||
				(compositeMemberFunctionType.ReturnTypeAnnotation.Type == nil &&
					interfaceMemberFunctionType.ReturnTypeAnnotation.Type != nil) {

				return false
			}
		}
	}

	// Check variable kind

	if interfaceMember.VariableKind != ast.VariableKindNotSpecified &&
		compositeMember.VariableKind != interfaceMember.VariableKind {

		return false
	}

	// Check access

	effectiveInterfaceMemberAccess := checker.effectiveInterfaceMemberAccess(interfaceMember.Access)
	effectiveCompositeMemberAccess := checker.EffectiveCompositeMemberAccess(compositeMember.Access)

	return !effectiveCompositeMemberAccess.IsLessPermissiveThan(effectiveInterfaceMemberAccess)
}

func CompositeLikeConstructorType(
	elaboration *Elaboration,
	compositeDeclaration ast.CompositeLikeDeclaration,
	compositeType *CompositeType,
) (
	constructorFunctionType *FunctionType,
	argumentLabels []string,
) {

	constructorFunctionType = &FunctionType{
		Purity:               compositeType.ConstructorPurity,
		IsConstructor:        true,
		ReturnTypeAnnotation: NewTypeAnnotation(compositeType),
	}

	// TODO: support multiple overloaded initializers

	initializers := compositeDeclaration.DeclarationMembers().Initializers()
	if len(initializers) > 0 {
		firstInitializer := initializers[0]

		argumentLabels = firstInitializer.
			FunctionDeclaration.
			ParameterList.
			EffectiveArgumentLabels()

		constructorFunctionType.Parameters = compositeType.ConstructorParameters

		// NOTE: Don't use `constructorFunctionType`, as it has a return type.
		//   The initializer itself has a `Void` return type.

		elaboration.SetConstructorFunctionType(
			firstInitializer,
			&FunctionType{
				IsConstructor:        true,
				Parameters:           constructorFunctionType.Parameters,
				ReturnTypeAnnotation: VoidTypeAnnotation,
			},
		)
	}

	return constructorFunctionType, argumentLabels
}

func (checker *Checker) defaultMembersAndOrigins(
	allMembers *ast.Members,
	containerType Type,
	containerKind ContainerKind,
	containerDeclarationKind common.DeclarationKind,
) (
	members *StringMemberOrderedMap,
	fieldNames []string,
	origins map[string]*Origin,
) {
	fields := allMembers.Fields()
	functions := allMembers.Functions()

	// Enum cases are invalid
	enumCases := allMembers.EnumCases()
	if len(enumCases) > 0 && containerDeclarationKind != common.DeclarationKindUnknown {
		checker.report(
			&InvalidEnumCaseError{
				ContainerDeclarationKind: containerDeclarationKind,
				Range:                    ast.NewRangeFromPositioned(checker.memoryGauge, enumCases[0]),
			},
		)
	}

	requireVariableKind := containerKind != ContainerKindInterface
	requireNonPrivateMemberAccess := containerKind == ContainerKindInterface

	memberCount := len(fields) + len(functions)
	members = &StringMemberOrderedMap{}
	if checker.PositionInfo != nil {
		origins = make(map[string]*Origin, memberCount)
	}

	predeclaredMembers := checker.predeclaredMembers(containerType)
	invalidIdentifiers := make(map[string]bool, len(predeclaredMembers))

	for _, predeclaredMember := range predeclaredMembers {
		name := predeclaredMember.Identifier.Identifier
		members.Set(name, predeclaredMember)
		invalidIdentifiers[name] = true

		if predeclaredMember.DeclarationKind == common.DeclarationKindField {
			fieldNames = append(fieldNames, name)
		}
	}

	checkInvalidIdentifier := func(declaration ast.Declaration) bool {
		identifier := declaration.DeclarationIdentifier()
		if invalidIdentifiers == nil || !invalidIdentifiers[identifier.Identifier] {
			return true
		}

		checker.report(
			&InvalidDeclarationError{
				Identifier: identifier.Identifier,
				Kind:       declaration.DeclarationKind(),
				Range:      ast.NewRangeFromPositioned(checker.memoryGauge, identifier),
			},
		)

		return false
	}

	// declare a member for each field
	for _, field := range fields {

		if !checkInvalidIdentifier(field) {
			continue
		}

		identifier := field.Identifier.Identifier

		fieldNames = append(fieldNames, identifier)

		fieldAccess := checker.accessFromAstAccess(field.Access)

		if entitlementMapAccess, ok := fieldAccess.(*EntitlementMapAccess); ok {
			checker.entitlementMappingInScope = entitlementMapAccess.Type
		}
		fieldTypeAnnotation := checker.ConvertTypeAnnotation(field.TypeAnnotation)
		checker.entitlementMappingInScope = nil
		checker.checkTypeAnnotation(fieldTypeAnnotation, field.TypeAnnotation)

		const declarationKind = common.DeclarationKindField

		effectiveAccess := checker.effectiveMemberAccess(fieldAccess, containerKind)

		if requireNonPrivateMemberAccess &&
			effectiveAccess.Equal(PrimitiveAccess(ast.AccessSelf)) {

			checker.report(
				&InvalidAccessModifierError{
					DeclarationKind: declarationKind,
					Access:          fieldAccess,
					Explanation:     "private fields can never be used",
					Pos:             field.StartPos,
				},
			)
		}

		checker.checkStaticModifier(field.IsStatic(), field.Identifier)
		checker.checkNativeModifier(field.IsNative(), field.Identifier)

		members.Set(
			identifier,
			&Member{
				ContainerType:   containerType,
				Access:          fieldAccess,
				Identifier:      field.Identifier,
				DeclarationKind: declarationKind,
				TypeAnnotation:  fieldTypeAnnotation,
				VariableKind:    field.VariableKind,
				DocString:       field.DocString,
			})

		if checker.PositionInfo != nil && origins != nil {
			origins[identifier] =
				checker.recordFieldDeclarationOrigin(
					field.Identifier,
					fieldTypeAnnotation.Type,
					field.DocString,
				)
		}

		if requireVariableKind &&
			field.VariableKind == ast.VariableKindNotSpecified {

			checker.report(
				&InvalidVariableKindError{
					Kind:  field.VariableKind,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, field.Identifier),
				},
			)
		}
	}

	// declare a member for each function
	for _, function := range functions {
		if !checkInvalidIdentifier(function) {
			continue
		}

		identifier := function.Identifier.Identifier

		functionAccess := checker.accessFromAstAccess(function.Access)

		functionType := checker.functionType(
			function.IsNative(),
			function.Purity,
			functionAccess,
			function.TypeParameterList,
			function.ParameterList,
			function.ReturnTypeAnnotation,
		)

		checker.Elaboration.SetFunctionDeclarationFunctionType(function, functionType)

		argumentLabels := function.ParameterList.EffectiveArgumentLabels()

		fieldTypeAnnotation := NewTypeAnnotation(functionType)

		const declarationKind = common.DeclarationKindFunction

		effectiveAccess := checker.effectiveMemberAccess(functionAccess, containerKind)

		if requireNonPrivateMemberAccess &&
			effectiveAccess.Equal(PrimitiveAccess(ast.AccessSelf)) {

			checker.report(
				&InvalidAccessModifierError{
					DeclarationKind: declarationKind,
					Access:          functionAccess,
					Explanation:     "private functions can never be used",
					Pos:             function.StartPos,
				},
			)
		}

		hasImplementation := function.FunctionBlock.HasStatements()
		hasConditions := function.FunctionBlock.HasConditions()

		members.Set(
			identifier,
			&Member{
				ContainerType:     containerType,
				Access:            functionAccess,
				Identifier:        function.Identifier,
				DeclarationKind:   declarationKind,
				TypeAnnotation:    fieldTypeAnnotation,
				VariableKind:      ast.VariableKindConstant,
				ArgumentLabels:    argumentLabels,
				DocString:         function.DocString,
				HasImplementation: hasImplementation,
				HasConditions:     hasConditions,
			})

		if checker.PositionInfo != nil && origins != nil {
			origins[identifier] = checker.recordFunctionDeclarationOrigin(function, functionType)
		}
	}

	return members, fieldNames, origins
}

func (checker *Checker) eventMembersAndOrigins(
	initializer *ast.SpecialFunctionDeclaration,
	containerType *CompositeType,
) (
	members *StringMemberOrderedMap,
	fieldNames []string,
	origins map[string]*Origin,
) {
	parameters := initializer.FunctionDeclaration.ParameterList.Parameters

	members = &StringMemberOrderedMap{}

	if checker.PositionInfo != nil {
		origins = make(map[string]*Origin, len(parameters))
	}

	for i, parameter := range parameters {
		typeAnnotation := containerType.ConstructorParameters[i].TypeAnnotation

		identifier := parameter.Identifier

		fieldNames = append(fieldNames, identifier.Identifier)

		members.Set(
			identifier.Identifier,
			&Member{
				ContainerType:   containerType,
				Access:          PrimitiveAccess(ast.AccessAll),
				Identifier:      identifier,
				DeclarationKind: common.DeclarationKindField,
				TypeAnnotation:  typeAnnotation,
				VariableKind:    ast.VariableKindConstant,
			})

		if checker.PositionInfo != nil && origins != nil {
			origins[identifier.Identifier] =
				checker.recordFieldDeclarationOrigin(
					identifier,
					typeAnnotation.Type,
					"",
				)
		}
	}

	return
}

const EnumRawValueFieldName = "rawValue"
const enumRawValueFieldDocString = `
The raw value of the enum case
`

func (checker *Checker) enumMembersAndOrigins(
	allMembers *ast.Members,
	containerType *CompositeType,
	containerDeclarationKind common.DeclarationKind,
) (
	members *StringMemberOrderedMap,
	fieldNames []string,
	origins map[string]*Origin,
) {
	for _, declaration := range allMembers.Declarations() {

		// Enum declarations may only contain enum cases

		enumCase, ok := declaration.(*ast.EnumCaseDeclaration)
		if !ok {
			checker.report(
				&InvalidNonEnumCaseError{
					ContainerDeclarationKind: containerDeclarationKind,
					Range:                    ast.NewRangeFromPositioned(checker.memoryGauge, declaration),
				},
			)
			continue
		}

		// Enum cases must be effectively public
		enumAccess := checker.accessFromAstAccess(enumCase.Access)

		if !checker.EffectiveCompositeMemberAccess(enumAccess).Equal(PrimitiveAccess(ast.AccessAll)) {
			checker.report(
				&InvalidAccessModifierError{
					DeclarationKind: enumCase.DeclarationKind(),
					Access:          enumAccess,
					Explanation:     "enum cases must be public",
					Pos:             enumCase.StartPos,
				},
			)
		}
	}

	// Members of the enum type are *not* the enum cases!
	// Each individual enum case is an instance of the enum type,
	// so only has a single member, the raw value field

	members = &StringMemberOrderedMap{}
	members.Set(
		EnumRawValueFieldName,
		&Member{
			ContainerType: containerType,
			Access:        PrimitiveAccess(ast.AccessAll),
			Identifier: ast.NewIdentifier(
				checker.memoryGauge,
				EnumRawValueFieldName,
				ast.EmptyPosition,
			),
			DeclarationKind: common.DeclarationKindField,
			TypeAnnotation:  NewTypeAnnotation(containerType.EnumRawType),
			VariableKind:    ast.VariableKindConstant,
			DocString:       enumRawValueFieldDocString,
		})

	// No origins available for the only member which was declared above

	origins = map[string]*Origin{}

	// Gather the field names from the members declared above

	members.Foreach(func(name string, member *Member) {
		if member.DeclarationKind == common.DeclarationKindField {
			fieldNames = append(fieldNames, name)
		}
	})

	return
}

func (checker *Checker) checkDefaultDestroyParamExpressionKind(
	arg ast.Expression,
) {
	switch arg := arg.(type) {
	case *ast.StringExpression,
		*ast.BoolExpression,
		*ast.NilExpression,
		*ast.IntegerExpression,
		*ast.FixedPointExpression,
		*ast.PathExpression:

		break

	case *ast.IdentifierExpression:

		identifier := arg.Identifier.Identifier
		// these are guaranteed to exist at time of destruction, so we allow them
		if identifier == SelfIdentifier || identifier == BaseIdentifier {
			break
		}
		// if it's an attachment, then it's also okay
		if checker.typeActivations.Find(identifier) != nil {
			break
		}
		checker.report(&DefaultDestroyInvalidArgumentError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, arg),
		})

	case *ast.MemberExpression:

		checker.checkDefaultDestroyParamExpressionKind(arg.Expression)

	case *ast.IndexExpression:

		checker.checkDefaultDestroyParamExpressionKind(arg.TargetExpression)
		checker.checkDefaultDestroyParamExpressionKind(arg.IndexingExpression)

		// indexing expressions on arrays can fail, and must be disallowed, but
		// indexing expressions on dicts, or composites (for attachments) will return `nil` and thus never fail
		targetExprType := checker.Elaboration.IndexExpressionTypes(arg).IndexedType
		// `nil` indicates that the index is a type-index (i.e. for an attachment access)
		if targetExprType == nil {
			return
		}

		switch targetExprType := targetExprType.(type) {
		case *DictionaryType:
			return
		case *ReferenceType:
			if _, isDictionaryType := targetExprType.Type.(*DictionaryType); isDictionaryType {
				return
			}
		}

		checker.report(&DefaultDestroyInvalidArgumentError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, arg),
		})

	default:

		checker.report(&DefaultDestroyInvalidArgumentError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, arg),
		})

	}
}

func (checker *Checker) checkDefaultDestroyEventParam(
	param Parameter,
	astParam *ast.Parameter,
	containerType EntitlementSupportingType,
	containerDeclaration ast.Declaration,
) {
	paramType := param.TypeAnnotation.Type
	paramDefaultArgument := astParam.DefaultArgument

	// make `self` and `base` available when checking default arguments so the fields of the composite are available
	// as this event is emitted when the resource is destroyed, these values should be fully entitled
	fullyEntitledAccess := NewAccessFromEntitlementSet(containerType.SupportedEntitlements(), Conjunction)
	checker.declareSelfValue(fullyEntitledAccess, containerType, containerDeclaration.DeclarationDocString())
	if compositeContainer, isComposite := containerType.(*CompositeType); isComposite && compositeContainer.Kind == common.CompositeKindAttachment {
		checker.declareBaseValue(
			fullyEntitledAccess,
			compositeContainer.baseType,
			compositeContainer,
			compositeContainer.baseTypeDocString)
	}
	param.DefaultArgument = checker.VisitExpression(paramDefaultArgument, paramType)

	unwrappedParamType := UnwrapOptionalType(paramType)
	// default events must have default arguments for all their parameters; this is enforced in the parser
	// we want to check that these arguments are all either literals or field accesses, and have primitive types
	if !IsSubType(unwrappedParamType, StringType) &&
		!IsSubType(unwrappedParamType, NumberType) &&
		!IsSubType(unwrappedParamType, TheAddressType) &&
		!IsSubType(unwrappedParamType, PathType) &&
		!IsSubType(unwrappedParamType, BoolType) {
		checker.report(&DefaultDestroyInvalidParameterError{
			ParamType: paramType,
			Range:     ast.NewRangeFromPositioned(checker.memoryGauge, astParam),
		})
	}

	checker.checkDefaultDestroyParamExpressionKind(paramDefaultArgument)
}

func (checker *Checker) checkDefaultDestroyEvent(
	eventType *CompositeType,
	eventDeclaration ast.CompositeLikeDeclaration,
	containerType EntitlementSupportingType,
	containerDeclaration ast.Declaration,
) {

	// an event definition always has one "constructor" function in its declaration list
	members := eventDeclaration.DeclarationMembers()
	functions := members.Initializers()
	constructorFunctionParameters := functions[0].FunctionDeclaration.ParameterList.Parameters

	checker.enterValueScope()
	defer checker.leaveValueScope(eventDeclaration.EndPosition, true)

	for index, param := range eventType.ConstructorParameters {
		checker.checkDefaultDestroyEventParam(param, constructorFunctionParameters[index], containerType, containerDeclaration)
	}
}

func (checker *Checker) checkInitializers(
	initializers []*ast.SpecialFunctionDeclaration,
	fields []*ast.FieldDeclaration,
	containerType CompositeKindedType,
	containerDocString string,
	initializerPurity FunctionPurity,
	initializerParameters []Parameter,
	containerKind ContainerKind,
	initializationInfo *InitializationInfo,
) {
	count := len(initializers)

	if count == 0 {
		checker.checkNoInitializerNoFields(fields, containerType, containerKind)
		return
	}

	// TODO: check all initializers:
	//  parameter initializerParameterTypeAnnotations needs to be a slice

	initializer := initializers[0]
	checker.checkSpecialFunction(
		initializer,
		containerType,
		containerDocString,
		initializerPurity,
		initializerParameters,
		containerKind,
		initializationInfo,
	)

	// If the initializer is for an event,
	// ensure all parameters are valid

	if compositeType, ok := containerType.(*CompositeType); ok &&
		compositeType.Kind == common.CompositeKindEvent {

		checker.checkEventParameters(
			initializer.FunctionDeclaration.ParameterList,
			initializerParameters,
		)
	}
}

// checkNoInitializerNoFields checks that if there are no initializers,
// then there should also be no fields. Otherwise, the fields will be uninitialized.
// In interfaces this is allowed.
func (checker *Checker) checkNoInitializerNoFields(
	fields []*ast.FieldDeclaration,
	containerType Type,
	containerKind ContainerKind,
) {
	// If there are no fields, or the container is an interface,
	// no initializer needs to be declared

	if len(fields) == 0 || containerKind == ContainerKindInterface {
		return
	}

	// An initializer should be declared but does not exist.
	// Report an error for the first field

	firstField := fields[0]

	checker.report(
		&MissingInitializerError{
			ContainerType:  containerType,
			FirstFieldName: firstField.Identifier.Identifier,
			FirstFieldPos:  firstField.Identifier.Pos,
		},
	)
}

// checkSpecialFunction checks special functions, like initializers and destructors
func (checker *Checker) checkSpecialFunction(
	specialFunction *ast.SpecialFunctionDeclaration,
	containerType CompositeKindedType,
	containerDocString string,
	purity FunctionPurity,
	parameters []Parameter,
	containerKind ContainerKind,
	initializationInfo *InitializationInfo,
) {
	// NOTE: new activation, so `self`
	// is only visible inside the special function

	checkResourceLoss := containerKind != ContainerKindInterface

	checker.enterValueScope()
	defer checker.leaveValueScope(specialFunction.EndPosition, checkResourceLoss)

	// initializers and destructors are considered fully entitled to their container type
	fnAccess := NewAccessFromEntitlementSet(containerType.SupportedEntitlements(), Conjunction)

	checker.declareSelfValue(fnAccess, containerType, containerDocString)
	if containerType.GetCompositeKind() == common.CompositeKindAttachment {
		// attachments cannot be interfaces, so this cast must succeed
		attachmentType, ok := containerType.(*CompositeType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		checker.declareBaseValue(
			fnAccess,
			attachmentType.baseType,
			attachmentType,
			attachmentType.baseTypeDocString)
	}

	functionType := NewSimpleFunctionType(
		purity,
		parameters,
		VoidTypeAnnotation,
	)

	checker.checkFunction(
		specialFunction.FunctionDeclaration.ParameterList,
		nil,
		fnAccess,
		functionType,
		specialFunction.FunctionDeclaration.FunctionBlock,
		true,
		initializationInfo,
		checkResourceLoss,
	)

	if containerKind == ContainerKindComposite {
		compositeType, ok := containerType.(*CompositeType)
		if !ok {
			// we just checked that the container was a composite
			panic(errors.NewUnreachableError())
		}

		// Event declarations have an empty initializer as it is synthesized
		if compositeType.Kind != common.CompositeKindEvent &&
			specialFunction.FunctionDeclaration.FunctionBlock == nil {

			checker.report(
				&MissingFunctionBodyError{
					Pos: specialFunction.EndPosition(checker.memoryGauge),
				},
			)
		}
	}
}

func (checker *Checker) checkCompositeFunctions(
	functions []*ast.FunctionDeclaration,
	selfType *CompositeType,
	selfDocString string,
) {
	for _, function := range functions {
		// NOTE: new activation, as function declarations
		// shouldn't be visible in other function declarations,
		// and `self` is only visible inside function

		func() {
			checker.enterValueScope()
			defer checker.leaveValueScope(function.EndPosition, true)

			fnAccess := checker.effectiveMemberAccess(checker.accessFromAstAccess(function.Access), ContainerKindComposite)
			// all non-entitlement functions produce unauthorized references in attachments
			if fnAccess.IsPrimitiveAccess() {
				fnAccess = UnauthorizedAccess
			}

			checker.declareSelfValue(fnAccess, selfType, selfDocString)
			if selfType.GetCompositeKind() == common.CompositeKindAttachment {
				checker.declareBaseValue(
					fnAccess,
					selfType.baseType,
					selfType,
					selfType.baseTypeDocString,
				)
			}

			checker.visitFunctionDeclaration(
				function,
				functionDeclarationOptions{
					mustExit:          true,
					declareFunction:   false,
					checkResourceLoss: true,
				},
				&selfType.Kind,
			)
		}()

		if function.FunctionBlock == nil {
			checker.report(
				&MissingFunctionBodyError{
					Pos: function.EndPosition(checker.memoryGauge),
				},
			)
		}
	}
}

// declares a value one scope lower than the current.
// This is useful particularly in the cases of creating `self`
// and `base` parameters to composite/attachment functions.

func (checker *Checker) declareLowerScopedValue(
	ty Type,
	docString string,
	identifier string,
	kind common.DeclarationKind,
) {

	depth := checker.valueActivations.Depth() + 1

	variable := &Variable{
		Identifier:      identifier,
		Access:          PrimitiveAccess(ast.AccessAll),
		DeclarationKind: kind,
		Type:            ty,
		IsConstant:      true,
		ActivationDepth: depth,
		Pos:             nil,
		DocString:       docString,
	}
	checker.valueActivations.Set(identifier, variable)
	if checker.PositionInfo != nil {
		checker.recordVariableDeclarationOccurrence(identifier, variable)
	}
}

func (checker *Checker) declareSelfValue(fnAccess Access, selfType Type, selfDocString string) {
	// inside of an attachment, self is a reference to the attachment's type, because
	// attachments are never first class values, they must always exist inside references
	if typedSelfType, ok := selfType.(*CompositeType); ok && typedSelfType.Kind == common.CompositeKindAttachment {
		// the `self` value in an attachment is entitled to the same entitlements required by the containing function
		selfType = NewReferenceType(checker.memoryGauge, fnAccess, typedSelfType)
	}
	checker.declareLowerScopedValue(selfType, selfDocString, SelfIdentifier, common.DeclarationKindSelf)
}

func (checker *Checker) declareBaseValue(fnAccess Access, baseType Type, attachmentType *CompositeType, superDocString string) {
	if typedBaseType, ok := baseType.(*InterfaceType); ok {
		// we can't actually have a value of an interface type I, so instead we create a value of {I}
		// to be referenced by `base`
		baseType = NewIntersectionType(checker.memoryGauge, []*InterfaceType{typedBaseType})
	}
	// the `base` value in an attachment is entitled to the same entitlements required by the containing function
	base := NewReferenceType(checker.memoryGauge, fnAccess, baseType)
	checker.declareLowerScopedValue(base, superDocString, BaseIdentifier, common.DeclarationKindBase)
}

// checkNestedIdentifiers checks that nested identifiers, i.e. fields, functions,
// and nested interfaces and composites, are unique and aren't named `init` or `destroy`
func (checker *Checker) checkNestedIdentifiers(members *ast.Members) {
	positions := map[string]ast.Position{}

	for _, declaration := range members.Declarations() {

		if _, ok := declaration.(*ast.SpecialFunctionDeclaration); ok {
			continue
		}

		identifier := declaration.DeclarationIdentifier()
		if identifier == nil {
			continue
		}

		checker.checkNestedIdentifier(
			*identifier,
			declaration.DeclarationKind(),
			positions,
		)
	}
}

// checkNestedIdentifier checks that the nested identifier is unique
// and isn't named `init` or `destroy`
func (checker *Checker) checkNestedIdentifier(
	identifier ast.Identifier,
	kind common.DeclarationKind,
	positions map[string]ast.Position,
) {
	name := identifier.Identifier
	pos := identifier.Pos

	// TODO: provide a more helpful error

	switch name {
	case common.DeclarationKindInitializer.Keywords():

		checker.report(
			&InvalidNameError{
				Name: name,
				Pos:  pos,
			},
		)
	}

	if previousPos, ok := positions[name]; ok {
		checker.report(
			&RedeclarationError{
				Name:        name,
				Pos:         pos,
				Kind:        kind,
				PreviousPos: &previousPos,
			},
		)
	} else {
		positions[name] = pos
	}
}

func (checker *Checker) VisitFieldDeclaration(_ *ast.FieldDeclaration) struct{} {
	// NOTE: field type is already checked when determining composite function in `compositeType`

	panic(errors.NewUnreachableError())
}

func (checker *Checker) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) struct{} {
	// NOTE: already checked when checking the composite

	panic(errors.NewUnreachableError())
}

// checkUnknownSpecialFunctions checks that the special function declarations
// are supported, i.e., they are either initializers or destructors
func (checker *Checker) checkUnknownSpecialFunctions(functions []*ast.SpecialFunctionDeclaration) {
	for _, function := range functions {
		switch function.Kind {
		case common.DeclarationKindInitializer:
			continue

		default:
			checker.report(
				&UnknownSpecialFunctionError{
					Pos: function.FunctionDeclaration.Identifier.Pos,
				},
			)
		}
	}
}

func (checker *Checker) checkSpecialFunctionDefaultImplementation(declaration ast.Declaration, kindName string) {
	for _, specialFunction := range declaration.DeclarationMembers().SpecialFunctions() {
		if !specialFunction.FunctionDeclaration.FunctionBlock.HasStatements() {
			continue
		}

		checker.report(
			&SpecialFunctionDefaultImplementationError{
				Identifier: specialFunction.DeclarationIdentifier(),
				Container:  declaration,
				KindName:   kindName,
			},
		)
	}
}

// checkResourceFieldsInvalidated checks that all resource fields for a container
// type are invalidated.
func (checker *Checker) checkResourceFieldsInvalidated(
	containerType Type,
	members *StringMemberOrderedMap,
) {
	members.Foreach(func(_ string, member *Member) {

		// NOTE: check the of the type annotation, not the type annotation's
		// resource marker: the field could have an incorrect type annotation
		// that is missing the resource marker even though it is required

		if !member.TypeAnnotation.Type.IsResourceType() {
			return
		}

		info := checker.resources.Get(Resource{Member: member})
		if !info.DefinitivelyInvalidated() {
			checker.report(
				&ResourceFieldNotInvalidatedError{
					FieldName: member.Identifier.Identifier,
					Type:      containerType,
					Pos:       member.Identifier.StartPosition(),
				},
			)
		}
	})
}

// checkResourceUseAfterInvalidation checks if a resource (variable or composite member)
// is used after it was previously invalidated (moved or destroyed)
func (checker *Checker) checkResourceUseAfterInvalidation(resource Resource, usePosition ast.HasPosition) {
	resourceInfo := checker.resources.Get(resource)
	invalidation := resourceInfo.Invalidation()
	if invalidation == nil {
		return
	}

	checker.report(
		&ResourceUseAfterInvalidationError{
			Invalidation: *invalidation,
			Range: ast.NewRangeFromPositioned(
				checker.memoryGauge,
				usePosition,
			),
		},
	)
}
