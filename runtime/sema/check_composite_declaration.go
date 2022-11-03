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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) (_ struct{}) {
	checker.visitCompositeDeclaration(declaration, ContainerKindComposite)
	return
}

// visitCompositeDeclaration checks a previously declared composite declaration.
// Checking behaviour depends on `kind`, i.e. if the composite declaration declares
// a composite (`kind` is `ContainerKindComposite`), or the composite declaration is
// nested in an interface and so acts as a type requirement (`kind` is `ContainerKindInterface`).
//
// NOTE: This function assumes that the composite type was previously declared using
// `declareCompositeType` and exists in `checker.Elaboration.CompositeDeclarationTypes`,
// and that the members and nested declarations for the composite type were declared
// through `declareCompositeMembersAndValue`.
func (checker *Checker) visitCompositeDeclaration(declaration *ast.CompositeDeclaration, kind ContainerKind) {

	compositeType := checker.Elaboration.CompositeDeclarationTypes[declaration]
	if compositeType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[compositeType] = true
	defer func() {
		checker.containerTypes[compositeType] = false
	}()

	checker.checkDeclarationAccessModifier(
		declaration.Access,
		declaration.DeclarationKind(),
		declaration.StartPos,
		true,
	)

	// NOTE: functions are checked separately
	checker.checkFieldsAccessModifier(declaration.Members.Fields())

	checker.checkNestedIdentifiers(declaration.Members)

	// Activate new scopes for nested types

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	if kind == ContainerKindComposite {
		checker.enterValueScope()
		defer checker.leaveValueScope(declaration.EndPosition, false)
	}

	checker.declareCompositeNestedTypes(declaration, kind, true)

	var initializationInfo *InitializationInfo

	if kind == ContainerKindComposite {
		// The initializer must initialize all members that are fields,
		// e.g. not composite functions (which are by definition constant and "initialized")

		fields := declaration.Members.Fields()
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
	}

	checker.checkInitializers(
		declaration.Members.Initializers(),
		declaration.Members.Fields(),
		compositeType,
		declaration.DeclarationDocString(),
		compositeType.ConstructorParameters,
		kind,
		initializationInfo,
	)

	checker.checkUnknownSpecialFunctions(declaration.Members.SpecialFunctions())

	switch kind {
	case ContainerKindComposite:
		checker.checkCompositeFunctions(
			declaration.Members.Functions(),
			compositeType,
			declaration.DocString,
		)

	case ContainerKindInterface:
		checker.checkSpecialFunctionDefaultImplementation(declaration, "type requirement")

		checker.checkInterfaceFunctions(
			declaration.Members.Functions(),
			compositeType,
			declaration.DeclarationKind(),
			declaration.DocString,
		)

	default:
		panic(errors.NewUnreachableError())
	}

	fieldPositionGetter := func(name string) ast.Position {
		return compositeType.FieldPosition(name, declaration)
	}

	checker.checkResourceFieldNesting(
		compositeType.Members,
		compositeType.Kind,
		fieldPositionGetter,
	)

	// Check conformances
	// NOTE: perform after completing composite type (e.g. setting constructor parameter types)

	// If the composite declaration is declaring a composite (`kind` is `ContainerKindComposite`),
	// rather than a type requirement (`kind` is `ContainerKindInterface`), check that the composite
	// conforms to all interfaces the composite declared it conforms to, i.e. all members match,
	// and no members are missing.

	// If the composite declaration is a type requirement (`kind` is `ContainerKindInterface`),
	// DON'T check that the composite conforms to all interfaces the composite declared it
	// conforms to – these are requirements that the composite declaration of the implementation
	// of the containing interface must conform to.
	//
	// Thus, missing members are valid, but still check that members that are declared as requirements
	// match the members of the conformances (members in the interface)

	checkMissingMembers := kind != ContainerKindInterface

	inheritedMembers := map[string]struct{}{}
	typeRequirementsInheritedMembers := map[string]map[string]struct{}{}

	for i, interfaceType := range compositeType.ExplicitInterfaceConformances {
		interfaceNominalType := declaration.Conformances[i]

		checker.checkCompositeConformance(
			declaration,
			compositeType,
			interfaceType,
			interfaceNominalType.Identifier,
			compositeConformanceCheckOptions{
				checkMissingMembers:            checkMissingMembers,
				interfaceTypeIsTypeRequirement: false,
			},
			inheritedMembers,
			typeRequirementsInheritedMembers,
		)
	}

	// NOTE: check destructors after initializer and functions

	checker.withSelfResourceInvalidationAllowed(func() {
		checker.checkDestructors(
			declaration.Members.Destructors(),
			declaration.Members.FieldsByIdentifier(),
			compositeType.Members,
			compositeType,
			declaration.DeclarationKind(),
			declaration.DeclarationDocString(),
			kind,
		)
	})

	// NOTE: visit interfaces first
	// DON'T use `nestedDeclarations`, because of non-deterministic order

	for _, nestedInterface := range declaration.Members.Interfaces() {
		ast.AcceptDeclaration[struct{}](nestedInterface, checker)
	}

	for _, nestedComposite := range declaration.Members.Composites() {
		ast.AcceptDeclaration[struct{}](nestedComposite, checker)
	}
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
func (checker *Checker) declareCompositeNestedTypes(
	declaration *ast.CompositeDeclaration,
	kind ContainerKind,
	declareConstructors bool,
) {
	compositeType := checker.Elaboration.CompositeDeclarationTypes[declaration]
	nestedDeclarations := checker.Elaboration.CompositeNestedDeclarations[declaration]

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
			access:                   nestedDeclaration.DeclarationAccess(),
			docString:                nestedDeclaration.DeclarationDocString(),
			allowOuterScopeShadowing: true,
		})
		checker.report(err)

		if declareConstructors && kind == ContainerKindComposite {

			// NOTE: Re-declare the constructor function for the nested composite declaration:
			// The constructor was previously declared in `declareCompositeMembersAndValue`
			// for this nested declaration, but the value activation for it was only temporary,
			// so that the constructor wouldn't be visible outside of the containing declaration

			if nestedCompositeDeclaration, isCompositeDeclaration :=
				nestedDeclaration.(*ast.CompositeDeclaration); isCompositeDeclaration {

				nestedCompositeType, ok := nestedType.(*CompositeType)
				if !ok {
					// we just checked that this was a composite declaration
					panic(errors.NewUnreachableError())
				}

				// Always determine composite constructor type

				nestedConstructorType, nestedConstructorArgumentLabels :=
					CompositeConstructorType(checker.Elaboration, nestedCompositeDeclaration, nestedCompositeType)

				switch nestedCompositeType.Kind {
				case common.CompositeKindContract:
					// not supported

				case common.CompositeKindEnum:
					checker.declareEnumConstructor(
						nestedCompositeDeclaration,
						nestedCompositeType,
					)

				default:
					checker.declareCompositeConstructor(
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
	nestedInterfaceDeclarations []*ast.InterfaceDeclaration,
) (
	nestedDeclarations map[string]ast.Declaration,
	nestedInterfaceTypes []*InterfaceType,
	nestedCompositeTypes []*CompositeType,
) {
	nestedDeclarations = map[string]ast.Declaration{}

	// Only contracts and contract interfaces support nested composite declarations
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

			reportInvalidNesting(
				firstNestedCompositeDeclaration.DeclarationKind(),
				firstNestedCompositeDeclaration.Identifier,
			)

		} else if len(nestedInterfaceDeclarations) > 0 {

			firstNestedInterfaceDeclaration := nestedInterfaceDeclarations[0]

			reportInvalidNesting(
				firstNestedInterfaceDeclaration.DeclarationKind(),
				firstNestedInterfaceDeclaration.Identifier,
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

			switch nestedCompositeKind {
			case common.CompositeKindResource,
				common.CompositeKindStructure,
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

		// NOTE: don't return, so nested declarations / types are still declared
	}

	// Declare nested interfaces

	for _, nestedDeclaration := range nestedInterfaceDeclarations {
		if _, exists := nestedDeclarations[nestedDeclaration.Identifier.Identifier]; !exists {
			nestedDeclarations[nestedDeclaration.Identifier.Identifier] = nestedDeclaration
		}

		nestedInterfaceType := checker.declareInterfaceType(nestedDeclaration)
		nestedInterfaceTypes = append(nestedInterfaceTypes, nestedInterfaceType)
	}

	// Declare nested composites

	for _, nestedDeclaration := range nestedCompositeDeclarations {
		if _, exists := nestedDeclarations[nestedDeclaration.Identifier.Identifier]; !exists {
			nestedDeclarations[nestedDeclaration.Identifier.Identifier] = nestedDeclaration
		}

		nestedCompositeType := checker.declareCompositeType(nestedDeclaration)
		nestedCompositeTypes = append(nestedCompositeTypes, nestedCompositeType)
	}

	return
}

// declareCompositeType declares the type for the given composite declaration
// and records it in the elaboration. It also recursively declares all types
// for all nested declarations.
//
// NOTE: The function does *not* declare any members or nested declarations.
//
// See `declareCompositeMembersAndValue` for the declaration of the composite type members.
// See `visitCompositeDeclaration` for the checking of the composite declaration.
func (checker *Checker) declareCompositeType(declaration *ast.CompositeDeclaration) *CompositeType {

	identifier := declaration.Identifier

	compositeType := &CompositeType{
		Location:    checker.Location,
		Kind:        declaration.CompositeKind,
		Identifier:  identifier.Identifier,
		NestedTypes: &StringTypeOrderedMap{},
		Members:     &StringMemberOrderedMap{},
	}

	variable, err := checker.typeActivations.declareType(typeDeclaration{
		identifier:               identifier,
		ty:                       compositeType,
		declarationKind:          declaration.DeclarationKind(),
		access:                   declaration.Access,
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

	// Resolve conformances

	if declaration.CompositeKind == common.CompositeKindEnum {
		compositeType.EnumRawType = checker.enumRawType(declaration)
	} else {
		compositeType.ExplicitInterfaceConformances =
			checker.explicitInterfaceConformances(declaration, compositeType)
	}

	// Register in elaboration

	checker.Elaboration.CompositeDeclarationTypes[declaration] = compositeType
	checker.Elaboration.CompositeTypeDeclarations[compositeType] = declaration

	// Activate new scope for nested declarations

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, false)

	// Check and declare nested types

	nestedDeclarations, nestedInterfaceTypes, nestedCompositeTypes :=
		checker.declareNestedDeclarations(
			declaration.CompositeKind,
			declaration.DeclarationKind(),
			declaration.Members.Composites(),
			declaration.Members.Interfaces(),
		)

	checker.Elaboration.CompositeNestedDeclarations[declaration] = nestedDeclarations

	for _, nestedInterfaceType := range nestedInterfaceTypes {
		compositeType.NestedTypes.Set(nestedInterfaceType.Identifier, nestedInterfaceType)
		nestedInterfaceType.SetContainerType(compositeType)
	}

	for _, nestedCompositeType := range nestedCompositeTypes {
		compositeType.NestedTypes.Set(nestedCompositeType.Identifier, nestedCompositeType)
		nestedCompositeType.SetContainerType(compositeType)
	}

	return compositeType
}

// declareCompositeMembersAndValue declares the members and the value
// (e.g. constructor function for non-contract types; instance for contracts)
// for the given composite declaration, and recursively for all nested declarations.
//
// NOTE: This function assumes that the composite type was previously declared using
// `declareCompositeType` and exists in `checker.Elaboration.CompositeDeclarationTypes`.
func (checker *Checker) declareCompositeMembersAndValue(
	declaration *ast.CompositeDeclaration,
	kind ContainerKind,
) {
	compositeType := checker.Elaboration.CompositeDeclarationTypes[declaration]
	if compositeType == nil {
		panic(errors.NewUnreachableError())
	}

	nestedComposites := declaration.Members.Composites()
	declarationMembers := orderedmap.New[StringMemberOrderedMap](len(nestedComposites))

	(func() {
		// Activate new scopes for nested types

		checker.typeActivations.Enter()
		defer checker.typeActivations.Leave(declaration.EndPosition)

		checker.enterValueScope()
		defer checker.leaveValueScope(declaration.EndPosition, false)

		checker.declareCompositeNestedTypes(declaration, kind, false)

		// NOTE: determine initializer parameter types while nested types are in scope,
		// and after declaring nested types as the initializer may use nested type in parameters

		initializers := declaration.Members.Initializers()
		compositeType.ConstructorParameters = checker.initializerParameters(initializers)

		// Declare nested declarations' members

		for _, nestedInterfaceDeclaration := range declaration.Members.Interfaces() {
			checker.declareInterfaceMembers(nestedInterfaceDeclaration)
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
		for _, nestedCompositeDeclaration := range nestedComposites {
			checker.declareCompositeMembersAndValue(nestedCompositeDeclaration, kind)

			// Declare nested composites' values (constructor/instance) as members of the containing composite

			identifier := nestedCompositeDeclaration.Identifier

			// Find the value declaration
			nestedCompositeDeclarationVariable :=
				checker.valueActivations.Find(identifier.Identifier)

			declarationMembers.Set(
				nestedCompositeDeclarationVariable.Identifier,
				&Member{
					Identifier:            identifier,
					Access:                nestedCompositeDeclaration.Access,
					ContainerType:         compositeType,
					TypeAnnotation:        NewTypeAnnotation(nestedCompositeDeclarationVariable.Type),
					DeclarationKind:       nestedCompositeDeclarationVariable.DeclarationKind,
					VariableKind:          ast.VariableKindConstant,
					IgnoreInSerialization: true,
					DocString:             nestedCompositeDeclaration.DocString,
				})
		}

		// Declare implicit type requirement conformances, if any,
		// after nested types are declared, and
		// after explicit conformances are declared.
		//
		// For each nested composite type, check if a conformance
		// declares a nested composite type with the same identifier,
		// in which case it is a type requirement,
		// and this nested composite type implicitly conforms to it.

		compositeType.GetNestedTypes().Foreach(func(nestedTypeIdentifier string, nestedType Type) {

			nestedCompositeType, ok := nestedType.(*CompositeType)
			if !ok {
				return
			}

			var inheritedMembers StringMemberOrderedMap

			for _, compositeTypeConformance := range compositeType.ExplicitInterfaceConformances {
				conformanceNestedTypes := compositeTypeConformance.GetNestedTypes()

				nestedType, ok := conformanceNestedTypes.Get(nestedTypeIdentifier)
				if !ok {
					continue
				}

				typeRequirement, ok := nestedType.(*CompositeType)
				if !ok {
					continue
				}

				nestedCompositeType.addImplicitTypeRequirementConformance(typeRequirement)

				// Add default functions

				typeRequirement.Members.Foreach(func(memberName string, member *Member) {

					if member.Predeclared ||
						member.DeclarationKind != common.DeclarationKindFunction {

						return
					}

					_, existing := nestedCompositeType.Members.Get(memberName)
					if existing {
						return
					}

					if _, ok := inheritedMembers.Get(memberName); ok {
						if member.HasImplementation {
							checker.report(
								&MultipleInterfaceDefaultImplementationsError{
									CompositeType: nestedCompositeType,
									Member:        member,
								},
							)
						} else {
							checker.report(
								&DefaultFunctionConflictError{
									CompositeType: nestedCompositeType,
									Member:        member,
								},
							)
						}

						return
					}

					if member.HasImplementation {
						inheritedMembers.Set(memberName, member)
					}
				})
			}

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

		switch declaration.CompositeKind {
		case common.CompositeKindEvent:
			// Event members are derived from the initializer's parameter list
			members, fields, origins = checker.eventMembersAndOrigins(
				initializers[0],
				compositeType,
			)

		case common.CompositeKindEnum:
			// Enum members are derived from the cases
			members, fields, origins = checker.enumMembersAndOrigins(
				declaration.Members,
				compositeType,
				declaration.DeclarationKind(),
			)

		default:
			members, fields, origins = checker.defaultMembersAndOrigins(
				declaration.Members,
				compositeType,
				kind,
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

	constructorType, constructorArgumentLabels := CompositeConstructorType(checker.Elaboration, declaration, compositeType)
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
			declaration,
			compositeType,
			declarationMembers,
		)

	case common.CompositeKindEnum:
		checker.declareEnumConstructor(
			declaration,
			compositeType,
		)

	default:
		checker.declareCompositeConstructor(
			declaration,
			constructorType,
			constructorArgumentLabels,
		)
	}
}

func (checker *Checker) declareCompositeConstructor(
	declaration *ast.CompositeDeclaration,
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
		identifier:               declaration.Identifier.Identifier,
		ty:                       constructorType,
		docString:                declaration.DocString,
		access:                   declaration.Access,
		kind:                     declaration.DeclarationKind(),
		pos:                      declaration.Identifier.Pos,
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
			access:     ast.AccessPublic,
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
				Access:          ast.AccessPublic,
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
		access:         ast.AccessPublic,
		kind:           common.DeclarationKindEnum,
		pos:            declaration.Identifier.Pos,
		isConstant:     true,
		argumentLabels: []string{EnumRawValueFieldName},
	})
	checker.report(err)
}

func EnumConstructorType(compositeType *CompositeType) *FunctionType {
	return &FunctionType{
		IsConstructor: true,
		Parameters: []*Parameter{
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

func (checker *Checker) initializerParameters(initializers []*ast.SpecialFunctionDeclaration) []*Parameter {
	// TODO: support multiple overloaded initializers
	var parameters []*Parameter

	initializerCount := len(initializers)
	if initializerCount > 0 {
		firstInitializer := initializers[0]
		parameters = checker.parameters(firstInitializer.FunctionDeclaration.ParameterList)

		if initializerCount > 1 {
			secondInitializer := initializers[1]

			checker.report(
				&UnsupportedOverloadingError{
					DeclarationKind: common.DeclarationKindInitializer,
					Range:           ast.NewRangeFromPositioned(checker.memoryGauge, secondInitializer),
				},
			)
		}
	}
	return parameters
}

func (checker *Checker) explicitInterfaceConformances(
	declaration *ast.CompositeDeclaration,
	compositeType *CompositeType,
) []*InterfaceType {

	var interfaceTypes []*InterfaceType
	seenConformances := map[*InterfaceType]bool{}

	for _, conformance := range declaration.Conformances {
		convertedType := checker.ConvertType(conformance)

		if interfaceType, ok := convertedType.(*InterfaceType); ok {
			interfaceTypes = append(interfaceTypes, interfaceType)

			if seenConformances[interfaceType] {
				checker.report(
					&DuplicateConformanceError{
						CompositeType: compositeType,
						InterfaceType: interfaceType,
						Range:         ast.NewRangeFromPositioned(checker.memoryGauge, conformance.Identifier),
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
	checkMissingMembers            bool
	interfaceTypeIsTypeRequirement bool
}

// checkCompositeConformance checks if the given composite declaration with the given composite type
// conforms to the specified interface type.
//
// inheritedMembers is an "input/output parameter":
// It tracks which members were inherited from the interface.
// It allows tracking this across conformance checks of multiple interfaces.
//
// typeRequirementsInheritedMembers is an "input/output parameter":
// It tracks which members were inherited in each nested type, which may be a conformance to a type requirement.
// It allows tracking this across conformance checks of multiple interfaces' type requirements.
func (checker *Checker) checkCompositeConformance(
	compositeDeclaration *ast.CompositeDeclaration,
	compositeType *CompositeType,
	interfaceType *InterfaceType,
	compositeKindMismatchIdentifier ast.Identifier,
	options compositeConformanceCheckOptions,
	inheritedMembers map[string]struct{},
// type requirement name -> inherited members
	typeRequirementsInheritedMembers map[string]map[string]struct{},
) {

	var missingMembers []*Member
	var memberMismatches []MemberMismatch
	var missingNestedCompositeTypes []*CompositeType
	var initializerMismatch *InitializerMismatch

	// Ensure the composite kinds match, e.g. a structure shouldn't be able
	// to conform to a resource interface

	if interfaceType.CompositeKind != compositeType.Kind {
		checker.report(
			&CompositeKindMismatchError{
				ExpectedKind: compositeType.Kind,
				ActualKind:   interfaceType.CompositeKind,
				Range:        ast.NewRangeFromPositioned(checker.memoryGauge, compositeKindMismatchIdentifier),
			},
		)
	}

	// Check initializer requirement

	// TODO: add support for overloaded initializers

	if interfaceType.InitializerParameters != nil {

		initializerType := &FunctionType{
			Parameters:           compositeType.ConstructorParameters,
			ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
		}
		interfaceInitializerType := &FunctionType{
			Parameters:           interfaceType.InitializerParameters,
			ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
		}

		// TODO: subtype?
		if !initializerType.Equal(interfaceInitializerType) {
			initializerMismatch = &InitializerMismatch{
				CompositeParameters: compositeType.ConstructorParameters,
				InterfaceParameters: interfaceType.InitializerParameters,
			}
		}
	}

	// Determine missing members and member conformance

	interfaceType.Members.Foreach(func(name string, interfaceMember *Member) {

		// Conforming types do not provide a concrete member
		// for the member in the interface if it is predeclared

		if interfaceMember.Predeclared {
			return
		}

		compositeMember, ok := compositeType.Members.Get(name)
		if ok {

			// If the composite member exists, check if it satisfies the mem

			if !checker.memberSatisfied(compositeMember, interfaceMember) {
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

				if _, ok := inheritedMembers[name]; ok {
					if interfaceMember.HasImplementation {
						checker.report(
							&MultipleInterfaceDefaultImplementationsError{
								CompositeType: compositeType,
								Member:        interfaceMember,
							},
						)
					} else {
						checker.report(
							&DefaultFunctionConflictError{
								CompositeType: compositeType,
								Member:        interfaceMember,
							},
						)
					}
					return
				}

				if interfaceMember.HasImplementation {
					inheritedMembers[name] = struct{}{}
					return
				}
			}

			missingMembers = append(missingMembers, interfaceMember)
		}

	})

	// Determine missing nested composite type definitions

	interfaceType.NestedTypes.Foreach(func(name string, typeRequirement Type) {

		// Only nested composite declarations are type requirements of the interface

		requiredCompositeType, ok := typeRequirement.(*CompositeType)
		if !ok {
			return
		}

		nestedCompositeType, ok := compositeType.NestedTypes.Get(name)
		if !ok {

			missingNestedCompositeTypes = append(missingNestedCompositeTypes, requiredCompositeType)
			return
		}

		inherited := typeRequirementsInheritedMembers[name]
		if inherited == nil {
			inherited = map[string]struct{}{}
			typeRequirementsInheritedMembers[name] = inherited
		}

		checker.checkTypeRequirement(nestedCompositeType, compositeDeclaration, requiredCompositeType, inherited)
	})

	if len(missingMembers) > 0 ||
		len(memberMismatches) > 0 ||
		len(missingNestedCompositeTypes) > 0 ||
		initializerMismatch != nil {

		checker.report(
			&ConformanceError{
				CompositeDeclaration:           compositeDeclaration,
				CompositeType:                  compositeType,
				InterfaceType:                  interfaceType,
				Pos:                            compositeDeclaration.Identifier.Pos,
				InitializerMismatch:            initializerMismatch,
				MissingMembers:                 missingMembers,
				MemberMismatches:               memberMismatches,
				MissingNestedCompositeTypes:    missingNestedCompositeTypes,
				InterfaceTypeIsTypeRequirement: options.interfaceTypeIsTypeRequirement,
			},
		)
	}

}

// TODO: return proper error
func (checker *Checker) memberSatisfied(compositeMember, interfaceMember *Member) bool {

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

			// Functions are invariant in their parameter types

			for i, subParameter := range compositeMemberFunctionType.Parameters {
				superParameter := interfaceMemberFunctionType.Parameters[i]
				if !subParameter.TypeAnnotation.Type.
					Equal(superParameter.TypeAnnotation.Type) {

					return false
				}
			}

			// Functions are covariant in their return type

			if compositeMemberFunctionType.ReturnTypeAnnotation != nil &&
				interfaceMemberFunctionType.ReturnTypeAnnotation != nil {

				if !IsSubType(
					compositeMemberFunctionType.ReturnTypeAnnotation.Type,
					interfaceMemberFunctionType.ReturnTypeAnnotation.Type,
				) {
					return false
				}
			}

			if (compositeMemberFunctionType.ReturnTypeAnnotation != nil &&
				interfaceMemberFunctionType.ReturnTypeAnnotation == nil) ||
				(compositeMemberFunctionType.ReturnTypeAnnotation == nil &&
					interfaceMemberFunctionType.ReturnTypeAnnotation != nil) {

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
	effectiveCompositeMemberAccess := checker.effectiveCompositeMemberAccess(compositeMember.Access)

	return !effectiveCompositeMemberAccess.IsLessPermissiveThan(effectiveInterfaceMemberAccess)
}

// checkTypeRequirement checks conformance of a nested type declaration
// to a type requirement of an interface.
func (checker *Checker) checkTypeRequirement(
	declaredType Type,
	containerDeclaration *ast.CompositeDeclaration,
	requiredCompositeType *CompositeType,
	inherited map[string]struct{},
) {

	// A nested interface doesn't satisfy the type requirement,
	// it must be a composite

	if declaredInterfaceType, ok := declaredType.(*InterfaceType); ok {

		// Find the interface declaration of the interface type

		var errorRange ast.Range
		var foundInterfaceDeclaration bool

		for _, nestedInterfaceDeclaration := range containerDeclaration.Members.Interfaces() {
			nestedInterfaceIdentifier := nestedInterfaceDeclaration.Identifier.Identifier
			if nestedInterfaceIdentifier == declaredInterfaceType.Identifier {
				foundInterfaceDeclaration = true
				errorRange = ast.NewRangeFromPositioned(checker.memoryGauge, nestedInterfaceDeclaration.Identifier)
				break
			}
		}

		if !foundInterfaceDeclaration {
			panic(errors.NewUnreachableError())
		}

		checker.report(
			&DeclarationKindMismatchError{
				ExpectedDeclarationKind: requiredCompositeType.Kind.DeclarationKind(false),
				ActualDeclarationKind:   declaredInterfaceType.CompositeKind.DeclarationKind(true),
				Range:                   errorRange,
			},
		)

		return
	}

	// If the nested type is neither an interface nor a composite,
	// something must be wrong in the checker

	declaredCompositeType, ok := declaredType.(*CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Find the composite declaration of the composite type

	var compositeDeclaration *ast.CompositeDeclaration
	var foundRedeclaration bool

	for _, nestedCompositeDeclaration := range containerDeclaration.Members.Composites() {
		nestedCompositeIdentifier := nestedCompositeDeclaration.Identifier.Identifier
		if nestedCompositeIdentifier == declaredCompositeType.Identifier {
			// If we detected a second nested composite declaration with the same identifier,
			// report an error and stop further type requirement checking
			if compositeDeclaration != nil {
				foundRedeclaration = true
				checker.report(&RedeclarationError{
					Kind:        nestedCompositeDeclaration.DeclarationKind(),
					Name:        nestedCompositeDeclaration.Identifier.Identifier,
					Pos:         nestedCompositeDeclaration.Identifier.Pos,
					PreviousPos: &compositeDeclaration.Identifier.Pos,
				})
			}
			compositeDeclaration = nestedCompositeDeclaration
			// NOTE: Do not break / stop iteration, but keep looking for
			// another (invalid) nested composite declaration with the same identifier,
			// as the first found declaration is not necessarily the correct one
		}
	}

	if foundRedeclaration {
		return
	}

	if compositeDeclaration == nil {
		panic(errors.NewUnreachableError())
	}

	// Check that the composite declaration declares at least the conformances
	// that the type requirement stated

	for _, requiredConformance := range requiredCompositeType.ExplicitInterfaceConformances {
		found := false
		for _, conformance := range declaredCompositeType.ExplicitInterfaceConformances {
			if conformance == requiredConformance {
				found = true
				break
			}
		}
		if !found {
			checker.report(
				&MissingConformanceError{
					CompositeType: declaredCompositeType,
					InterfaceType: requiredConformance,
					Range:         ast.NewRangeFromPositioned(checker.memoryGauge, compositeDeclaration.Identifier),
				},
			)
		}
	}

	// Check the conformance of the composite to the type requirement
	// like a top-level composite declaration to an interface type

	requiredInterfaceType := requiredCompositeType.InterfaceType()

	checker.checkCompositeConformance(
		compositeDeclaration,
		declaredCompositeType,
		requiredInterfaceType,
		compositeDeclaration.Identifier,
		compositeConformanceCheckOptions{
			checkMissingMembers:            true,
			interfaceTypeIsTypeRequirement: true,
		},
		inherited,
		map[string]map[string]struct{}{},
	)
}

func CompositeConstructorType(
	elaboration *Elaboration,
	compositeDeclaration *ast.CompositeDeclaration,
	compositeType *CompositeType,
) (
	constructorFunctionType *FunctionType,
	argumentLabels []string,
) {

	constructorFunctionType = &FunctionType{
		IsConstructor:        true,
		ReturnTypeAnnotation: NewTypeAnnotation(compositeType),
	}

	// TODO: support multiple overloaded initializers

	initializers := compositeDeclaration.Members.Initializers()
	if len(initializers) > 0 {
		firstInitializer := initializers[0]

		argumentLabels = firstInitializer.
			FunctionDeclaration.
			ParameterList.
			EffectiveArgumentLabels()

		constructorFunctionType.Parameters = compositeType.ConstructorParameters

		// NOTE: Don't use `constructorFunctionType`, as it has a return type.
		//   The initializer itself has a `Void` return type.

		elaboration.ConstructorFunctionTypes[firstInitializer] =
			&FunctionType{
				IsConstructor:        true,
				Parameters:           constructorFunctionType.Parameters,
				ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
			}
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

		fieldTypeAnnotation := checker.ConvertTypeAnnotation(field.TypeAnnotation)
		checker.checkTypeAnnotation(fieldTypeAnnotation, field.TypeAnnotation)

		const declarationKind = common.DeclarationKindField

		effectiveAccess := checker.effectiveMemberAccess(field.Access, containerKind)

		if requireNonPrivateMemberAccess &&
			effectiveAccess == ast.AccessPrivate {

			checker.report(
				&InvalidAccessModifierError{
					DeclarationKind: declarationKind,
					Access:          field.Access,
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
				Access:          field.Access,
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

		functionType := checker.functionType(function.ParameterList, function.ReturnTypeAnnotation)

		argumentLabels := function.ParameterList.EffectiveArgumentLabels()

		fieldTypeAnnotation := NewTypeAnnotation(functionType)

		const declarationKind = common.DeclarationKindFunction

		effectiveAccess := checker.effectiveMemberAccess(function.Access, containerKind)

		if requireNonPrivateMemberAccess &&
			effectiveAccess == ast.AccessPrivate {

			checker.report(
				&InvalidAccessModifierError{
					DeclarationKind: declarationKind,
					Access:          function.Access,
					Explanation:     "private functions can never be used",
					Pos:             function.StartPos,
				},
			)
		}

		hasImplementation := function.FunctionBlock.HasStatements()

		members.Set(
			identifier,
			&Member{
				ContainerType:     containerType,
				Access:            function.Access,
				Identifier:        function.Identifier,
				DeclarationKind:   declarationKind,
				TypeAnnotation:    fieldTypeAnnotation,
				VariableKind:      ast.VariableKindConstant,
				ArgumentLabels:    argumentLabels,
				DocString:         function.DocString,
				HasImplementation: hasImplementation,
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
				Access:          ast.AccessPublic,
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

		if checker.effectiveCompositeMemberAccess(enumCase.Access) != ast.AccessPublic {
			checker.report(
				&InvalidAccessModifierError{
					DeclarationKind: enumCase.DeclarationKind(),
					Access:          enumCase.Access,
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
			Access:        ast.AccessPublic,
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

func (checker *Checker) checkInitializers(
	initializers []*ast.SpecialFunctionDeclaration,
	fields []*ast.FieldDeclaration,
	containerType Type,
	containerDocString string,
	initializerParameters []*Parameter,
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
	containerType Type,
	containerDocString string,
	parameters []*Parameter,
	containerKind ContainerKind,
	initializationInfo *InitializationInfo,
) {
	// NOTE: new activation, so `self`
	// is only visible inside the special function

	checkResourceLoss := containerKind != ContainerKindInterface

	checker.enterValueScope()
	defer checker.leaveValueScope(specialFunction.EndPosition, checkResourceLoss)

	checker.declareSelfValue(containerType, containerDocString)

	functionType := &FunctionType{
		Parameters:           parameters,
		ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
	}

	checker.checkFunction(
		specialFunction.FunctionDeclaration.ParameterList,
		nil,
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

			checker.declareSelfValue(selfType, selfDocString)

			checker.visitFunctionDeclaration(
				function,
				functionDeclarationOptions{
					mustExit:          true,
					declareFunction:   false,
					checkResourceLoss: true,
				},
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

func (checker *Checker) declareSelfValue(selfType Type, selfDocString string) {

	// NOTE: declare `self` one depth lower ("inside" function),
	// so it can't be re-declared by the function's parameters

	depth := checker.valueActivations.Depth() + 1

	self := &Variable{
		Identifier:      SelfIdentifier,
		Access:          ast.AccessPublic,
		DeclarationKind: common.DeclarationKindSelf,
		Type:            selfType,
		IsConstant:      true,
		ActivationDepth: depth,
		Pos:             nil,
		DocString:       selfDocString,
	}
	checker.valueActivations.Set(SelfIdentifier, self)
	if checker.PositionInfo != nil {
		checker.recordVariableDeclarationOccurrence(SelfIdentifier, self)
	}
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
	case common.DeclarationKindInitializer.Keywords(),
		common.DeclarationKindDestructor.Keywords():

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
		case common.DeclarationKindInitializer, common.DeclarationKindDestructor:
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

func (checker *Checker) checkDestructors(
	destructors []*ast.SpecialFunctionDeclaration,
	fields map[string]*ast.FieldDeclaration,
	members *StringMemberOrderedMap,
	containerType Type,
	containerDeclarationKind common.DeclarationKind,
	containerDocString string,
	containerKind ContainerKind,
) {
	count := len(destructors)

	// only resource and resource interface declarations may
	// declare a destructor

	if !containerType.IsResourceType() {
		if count > 0 {
			firstDestructor := destructors[0]

			checker.report(
				&InvalidDestructorError{
					Range: ast.NewRangeFromPositioned(
						checker.memoryGauge,
						firstDestructor.FunctionDeclaration.Identifier,
					),
				},
			)
		}

		return
	}

	if count == 0 {
		checker.checkNoDestructorNoResourceFields(members, fields, containerType, containerKind)
		return
	}

	firstDestructor := destructors[0]
	checker.checkDestructor(
		firstDestructor,
		containerType,
		containerDocString,
		containerKind,
	)

	// destructor overloading is not supported

	if count > 1 {
		secondDestructor := destructors[1]

		checker.report(
			&UnsupportedOverloadingError{
				DeclarationKind: common.DeclarationKindDestructor,
				Range:           ast.NewRangeFromPositioned(checker.memoryGauge, secondDestructor),
			},
		)
	}
}

// checkNoDestructorNoResourceFields checks that if there is no destructor there are
// also no fields which have a resource type – otherwise those fields will be lost.
// In interfaces this is allowed.
func (checker *Checker) checkNoDestructorNoResourceFields(
	members *StringMemberOrderedMap,
	fields map[string]*ast.FieldDeclaration,
	containerType Type,
	containerKind ContainerKind,
) {
	if containerKind == ContainerKindInterface {
		return
	}

	for pair := members.Oldest(); pair != nil; pair = pair.Next() {
		member := pair.Value
		memberName := pair.Key

		// NOTE: check type, not resource annotation:
		// the field could have a wrong annotation
		if !member.TypeAnnotation.Type.IsResourceType() {
			continue
		}

		checker.report(
			&MissingDestructorError{
				ContainerType:  containerType,
				FirstFieldName: memberName,
				FirstFieldPos:  fields[memberName].Identifier.Pos,
			},
		)

		// only report for first member
		return
	}
}

func (checker *Checker) checkDestructor(
	destructor *ast.SpecialFunctionDeclaration,
	containerType Type,
	containerDocString string,
	containerKind ContainerKind,
) {

	if len(destructor.FunctionDeclaration.ParameterList.Parameters) != 0 {
		checker.report(
			&InvalidDestructorParametersError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, destructor.FunctionDeclaration.ParameterList),
			},
		)
	}

	parameters := checker.parameters(destructor.FunctionDeclaration.ParameterList)

	checker.checkSpecialFunction(
		destructor,
		containerType,
		containerDocString,
		parameters,
		containerKind,
		nil,
	)

	checker.checkCompositeResourceInvalidated(containerType)
}

// checkCompositeResourceInvalidated checks that if the container is a resource,
// that all resource fields are invalidated (moved or destroyed)
func (checker *Checker) checkCompositeResourceInvalidated(containerType Type) {
	compositeType, isComposite := containerType.(*CompositeType)
	if !isComposite || compositeType.Kind != common.CompositeKindResource {
		return
	}

	checker.checkResourceFieldsInvalidated(containerType, compositeType.Members)
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
