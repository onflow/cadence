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
	goErrors "errors"
	"math"
	"math/big"

	"github.com/rivo/uniseg"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/persistent"
	"github.com/onflow/cadence/runtime/errors"
)

const ArgumentLabelNotRequired = "_"
const SelfIdentifier = "self"
const BaseIdentifier = "base"
const BeforeIdentifier = "before"
const ResultIdentifier = "result"

var beforeType = func() *FunctionType {

	typeParameter := &TypeParameter{
		Name:      "T",
		TypeBound: AnyStructType,
	}

	typeAnnotation := NewTypeAnnotation(
		&GenericType{
			TypeParameter: typeParameter,
		},
	)

	return &FunctionType{
		Purity: FunctionPurityView,
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: typeAnnotation,
			},
		},
		ReturnTypeAnnotation: typeAnnotation,
	}
}()

type ValidTopLevelDeclarationsHandlerFunc = func(common.Location) common.DeclarationKindSet

type CheckHandlerFunc func(checker *Checker, check func())

type ResolvedLocation struct {
	Location    common.Location
	Identifiers []ast.Identifier
}

type LocationHandlerFunc func(identifiers []ast.Identifier, location common.Location) ([]ResolvedLocation, error)

type ImportHandlerFunc func(checker *Checker, importedLocation common.Location, importRange ast.Range) (Import, error)

type MemberAccountAccessHandlerFunc func(checker *Checker, memberLocation common.Location) bool

type PurityCheckScope struct {
	// whether encountering an impure operation should cause an error
	EnforcePurity   bool
	ActivationDepth int
}

type ContractValueHandlerFunc func(
	checker *Checker,
	declaration *ast.CompositeDeclaration,
	compositeType *CompositeType,
) ValueDeclaration

// Checker

type Checker struct {
	// memoryGauge is used for metering memory usage
	memoryGauge             common.MemoryGauge
	Location                common.Location
	expectedType            Type
	resources               *Resources
	valueActivations        *VariableActivations
	currentMemberExpression *ast.MemberExpression
	typeActivations         *VariableActivations
	containerTypes          map[Type]bool
	Program                 *ast.Program
	PositionInfo            *PositionInfo
	Config                  *Config
	Elaboration             *Elaboration
	// initialized lazily. use beforeExtractor()
	_beforeExtractor                   *BeforeExtractor
	errors                             []error
	functionActivations                *FunctionActivations
	purityCheckScopes                  []PurityCheckScope
	entitlementMappingInScope          *EntitlementMapType
	inCondition                        bool
	allowSelfResourceFieldInvalidation bool
	inAssignment                       bool
	inInvocation                       bool
	inCreate                           bool
	isChecked                          bool
}

var _ ast.DeclarationVisitor[struct{}] = &Checker{}
var _ ast.StatementVisitor[struct{}] = &Checker{}
var _ ast.ExpressionVisitor[Type] = &Checker{}

var baseFunctionType = NewSimpleFunctionType(
	FunctionPurityImpure,
	nil,
	VoidTypeAnnotation,
)

func NewChecker(
	program *ast.Program,
	location common.Location,
	memoryGauge common.MemoryGauge,
	config *Config,
) (*Checker, error) {

	if location == nil {
		return nil, errors.NewDefaultUserError("missing location")
	}

	if config.AccessCheckMode == AccessCheckModeDefault {
		return nil, errors.NewDefaultUserError("invalid default access check mode")
	}

	functionActivations := &FunctionActivations{
		// Pre-allocate a common function depth
		activations: make([]*FunctionActivation, 0, 2),
	}
	functionActivations.EnterFunction(
		baseFunctionType,
		0,
	)

	elaboration := NewElaboration(memoryGauge)

	checker := &Checker{
		Program:             program,
		Location:            location,
		Config:              config,
		Elaboration:         elaboration,
		resources:           NewResources(),
		functionActivations: functionActivations,
		containerTypes:      map[Type]bool{},
		purityCheckScopes:   []PurityCheckScope{{}},
		memoryGauge:         memoryGauge,
	}

	// Initialize value activations

	baseValueActivation := config.BaseValueActivation
	if baseValueActivation == nil {
		baseValueActivation = BaseValueActivation
	}
	checker.valueActivations = NewVariableActivations(baseValueActivation)

	// Initialize type activations

	baseTypeActivation := config.BaseTypeActivation
	if baseTypeActivation == nil {
		baseTypeActivation = BaseTypeActivation
	}
	checker.typeActivations = NewVariableActivations(baseTypeActivation)

	// Initialize position info, if enabled
	if checker.Config.PositionInfoEnabled {
		checker.PositionInfo = NewPositionInfo()
	}

	return checker, nil
}

func (checker *Checker) SubChecker(program *ast.Program, location common.Location) (*Checker, error) {
	return NewChecker(
		program,
		location,
		checker.memoryGauge,
		checker.Config,
	)
}

func (checker *Checker) SetMemoryGauge(gauge common.MemoryGauge) {
	checker.memoryGauge = gauge
}

func (checker *Checker) IsChecked() bool {
	return checker.isChecked
}

func (checker *Checker) CurrentPurityScope() PurityCheckScope {
	return checker.purityCheckScopes[len(checker.purityCheckScopes)-1]
}

func (checker *Checker) PushNewPurityScope(enforce bool, depth int) {
	checker.purityCheckScopes = append(
		checker.purityCheckScopes,
		PurityCheckScope{
			EnforcePurity:   enforce,
			ActivationDepth: depth,
		},
	)
}

func (checker *Checker) PopPurityScope() PurityCheckScope {
	scope := checker.CurrentPurityScope()
	checker.purityCheckScopes = checker.purityCheckScopes[:len(checker.purityCheckScopes)-1]
	return scope
}

func (checker *Checker) EnforcePurity(operation ast.Element, purity FunctionPurity) {
	if purity == FunctionPurityImpure {
		checker.ObserveImpureOperation(operation)
	}
}

func (checker *Checker) ObserveImpureOperation(operation ast.Element) {
	scope := checker.CurrentPurityScope()
	if scope.EnforcePurity {
		checker.report(
			&PurityError{Range: ast.NewRangeFromPositioned(checker.memoryGauge, operation)},
		)
	}
}

func (checker *Checker) InNewPurityScope(enforce bool, f func()) {
	checker.PushNewPurityScope(enforce, checker.ValueActivationDepth())
	f()
	checker.PopPurityScope()
}

type stopChecking struct{}

func (checker *Checker) Check() error {
	if !checker.IsChecked() {
		checker.Elaboration.setIsChecking(true)
		checker.errors = nil
		check := func() {
			if checker.Config.ErrorShortCircuitingEnabled {
				defer func() {
					switch recovered := recover().(type) {
					case stopChecking:
						// checking should stop
						break
					case nil:
						// nothing was recovered
						break
					default:
						// re-panic what was recovered
						panic(recovered)
					}
				}()
			}

			checker.CheckProgram(checker.Program)
		}
		if checker.Config.CheckHandler != nil {
			checker.Config.CheckHandler(checker, check)
		} else {
			check()
		}

		if checker.PositionInfo != nil {
			checker.declareGlobalRanges()
		}

		checker.Elaboration.setIsChecking(false)
		checker.isChecked = true

		checker.resources.Reclaim()
		checker.resources = nil
	}
	err := checker.CheckerError()
	if err != nil {
		return err
	}
	return nil
}

func (checker *Checker) CheckerError() *CheckerError {
	if len(checker.errors) > 0 {
		return &CheckerError{
			Location: checker.Location,
			Errors:   checker.errors,
		}
	}
	return nil
}

func (checker *Checker) report(err error) {
	if err == nil {
		return
	}

	checker.errors = append(checker.errors, err)
	if checker.Config.ErrorShortCircuitingEnabled {
		panic(stopChecking{})
	}
}

func (checker *Checker) CheckProgram(program *ast.Program) {

	for _, declaration := range program.ImportDeclarations() {
		checker.declareImportDeclaration(declaration)
	}

	// Declare interface and composite types

	registerInElaboration := func(ty Type) {
		switch typedType := ty.(type) {
		case *InterfaceType:
			checker.Elaboration.SetInterfaceType(typedType.ID(), typedType)
		case *CompositeType:
			checker.Elaboration.SetCompositeType(typedType.ID(), typedType)
		case *EntitlementType:
			checker.Elaboration.SetEntitlementType(typedType.ID(), typedType)
		case *EntitlementMapType:
			checker.Elaboration.SetEntitlementMapType(typedType.ID(), typedType)
		default:
			panic(errors.NewUnreachableError())
		}
	}

	for _, declaration := range program.EntitlementDeclarations() {
		entitlementType := checker.declareEntitlementType(declaration)

		// NOTE: register types in elaboration
		// *after* the full container chain is fully set up

		VisitThisAndNested(entitlementType, registerInElaboration)
	}

	for _, declaration := range program.EntitlementMappingDeclarations() {
		entitlementType := checker.declareEntitlementMappingType(declaration)

		// NOTE: register types in elaboration
		// *after* the full container chain is fully set up

		VisitThisAndNested(entitlementType, registerInElaboration)
	}

	for _, declaration := range program.InterfaceDeclarations() {
		interfaceType := checker.declareInterfaceType(declaration)

		// NOTE: register types in elaboration
		// *after* the full container chain is fully set up

		VisitThisAndNested(interfaceType, registerInElaboration)
	}

	for _, declaration := range program.CompositeDeclarations() {
		compositeType := checker.declareCompositeType(declaration)

		// NOTE: register types in elaboration
		// *after* the full container chain is fully set up

		VisitThisAndNested(compositeType, registerInElaboration)
	}

	for _, declaration := range program.AttachmentDeclarations() {
		compositeType := checker.declareAttachmentType(declaration)

		// NOTE: register types in elaboration
		// *after* the full container chain is fully set up

		VisitThisAndNested(compositeType, registerInElaboration)
	}

	// Declare interfaces' and composites' members

	for _, declaration := range program.InterfaceDeclarations() {
		checker.declareInterfaceMembers(declaration)
	}

	for _, declaration := range program.CompositeDeclarations() {
		checker.declareCompositeLikeMembersAndValue(declaration, ContainerKindComposite)
	}

	for _, declaration := range program.AttachmentDeclarations() {
		checker.declareAttachmentMembersAndValue(declaration, ContainerKindComposite)
	}

	// Declare events, functions, and transactions

	for _, declaration := range program.FunctionDeclarations() {
		checker.declareGlobalFunctionDeclaration(declaration)
	}

	for _, declaration := range program.TransactionDeclarations() {
		checker.declareTransactionDeclaration(declaration)
	}

	// Check all declarations

	declarations := program.Declarations()

	checker.checkTopLevelDeclarationsValidity(declarations)

	var rejectAllowAccountLinkingPragma bool

	for _, declaration := range declarations {

		// A pragma declaration #allowAccountLinking determines
		// if the program is allowed to use account linking.
		//
		// It must appear as a top-level declaration (i.e. not nested in the program),
		// and must appear before all other declarations (i.e. at the top of the program).
		//
		// This is a temporary feature, which is planned to get replaced by capability controllers,
		// and a new Account type with entitlements.

		if pragmaDeclaration, isPragma := declaration.(*ast.PragmaDeclaration); isPragma {
			if IsAllowAccountLinkingPragma(pragmaDeclaration) {
				if rejectAllowAccountLinkingPragma {
					checker.reportInvalidNonHeaderPragma(pragmaDeclaration)
				}
				continue
			}
		}

		rejectAllowAccountLinkingPragma = true

		// Skip import declarations, they are already handled above
		if _, isImport := declaration.(*ast.ImportDeclaration); isImport {
			continue
		}

		ast.AcceptDeclaration[struct{}](declaration, checker)
		checker.declareGlobalDeclaration(declaration)
	}
}

func (checker *Checker) checkTopLevelDeclarationsValidity(declarations []ast.Declaration) {
	validTopLevelDeclarationsHandler := checker.Config.ValidTopLevelDeclarationsHandler

	if validTopLevelDeclarationsHandler == nil {
		return
	}

	validTopLevelDeclarations := validTopLevelDeclarationsHandler(checker.Location)

	for _, declaration := range declarations {
		checker.checkTopLevelDeclarationValidity(declaration, validTopLevelDeclarations)
	}
}

func (checker *Checker) checkTopLevelDeclarationValidity(
	declaration ast.Declaration,
	validTopLevelDeclarations common.DeclarationKindSet,
) {
	declarationKind := declaration.DeclarationKind()

	if validTopLevelDeclarations.Has(declarationKind) {
		return
	}

	var errorRange ast.Range

	identifier := declaration.DeclarationIdentifier()
	if identifier == nil {
		position := declaration.StartPosition()
		errorRange = ast.NewRange(
			checker.memoryGauge,
			position,
			position,
		)
	} else {
		errorRange = ast.NewRangeFromPositioned(checker.memoryGauge, identifier)
	}

	checker.report(
		&InvalidTopLevelDeclarationError{
			DeclarationKind: declarationKind,
			Range:           errorRange,
		},
	)
}

func (checker *Checker) declareGlobalFunctionDeclaration(declaration *ast.FunctionDeclaration) {
	functionType := checker.functionType(
		declaration.Purity,
		UnauthorizedAccess,
		declaration.ParameterList,
		declaration.ReturnTypeAnnotation,
	)
	checker.Elaboration.SetFunctionDeclarationFunctionType(declaration, functionType)
	checker.declareFunctionDeclaration(declaration, functionType)
}

func (checker *Checker) checkTransfer(transfer *ast.Transfer, valueType Type) {
	if valueType.IsResourceType() {
		if !transfer.Operation.IsMove() {
			checker.report(
				&IncorrectTransferOperationError{
					ActualOperation:   transfer.Operation,
					ExpectedOperation: ast.TransferOperationMove,
					Range:             ast.NewRangeFromPositioned(checker.memoryGauge, transfer),
				},
			)
		}
	} else if !valueType.IsInvalidType() {
		if transfer.Operation.IsMove() {
			checker.report(
				&IncorrectTransferOperationError{
					ActualOperation:   transfer.Operation,
					ExpectedOperation: ast.TransferOperationCopy,
					Range:             ast.NewRangeFromPositioned(checker.memoryGauge, transfer),
				},
			)
		}
	}
}

// This method is only used for checking invocation-expression.
// This is also temporary, until the type inferring support is added for func arguments
// TODO: Remove this method
func (checker *Checker) checkTypeCompatibility(expression ast.Expression, valueType Type, targetType Type) bool {
	switch typedExpression := expression.(type) {
	case *ast.IntegerExpression:
		unwrappedTargetType := UnwrapOptionalType(targetType)

		if IsSameTypeKind(unwrappedTargetType, IntegerType) {
			CheckIntegerLiteral(checker.memoryGauge, typedExpression, unwrappedTargetType, checker.report)

			return true

		} else if IsSameTypeKind(unwrappedTargetType, TheAddressType) {
			CheckAddressLiteral(checker.memoryGauge, typedExpression, checker.report)

			return true
		}

	case *ast.FixedPointExpression:
		unwrappedTargetType := UnwrapOptionalType(targetType)

		if IsSameTypeKind(unwrappedTargetType, FixedPointType) {
			valueTypeOK := CheckFixedPointLiteral(checker.memoryGauge, typedExpression, valueType, checker.report)
			if valueTypeOK {
				CheckFixedPointLiteral(checker.memoryGauge, typedExpression, unwrappedTargetType, checker.report)
			}
			return true
		}

	case *ast.ArrayExpression:

		// Variable sized array literals are compatible with constant sized target types
		// if their element type matches and the element count matches

		if variableSizedValueType, isVariableSizedValue :=
			valueType.(*VariableSizedType); isVariableSizedValue {

			if constantSizedTargetType, isConstantSizedTarget :=
				targetType.(*ConstantSizedType); isConstantSizedTarget {

				valueElementType := variableSizedValueType.ElementType(false)
				targetElementType := constantSizedTargetType.ElementType(false)

				literalCount := int64(len(typedExpression.Values))

				if IsSubType(valueElementType, targetElementType) {

					expectedSize := constantSizedTargetType.Size

					if literalCount == expectedSize {
						return true
					}

					checker.report(
						&ConstantSizedArrayLiteralSizeError{
							ExpectedSize: expectedSize,
							ActualSize:   literalCount,
							Range:        typedExpression.Range,
						},
					)
				}
			}
		}

	case *ast.StringExpression:
		unwrappedTargetType := UnwrapOptionalType(targetType)

		if IsSameTypeKind(unwrappedTargetType, CharacterType) {
			checker.checkCharacterLiteral(typedExpression)

			return true
		}
	}

	return IsSubType(valueType, targetType)
}

// CheckIntegerLiteral checks that the value of the integer literal
// fits into range of the target integer type
func CheckIntegerLiteral(memoryGauge common.MemoryGauge, expression *ast.IntegerExpression, targetType Type, report func(error)) bool {
	ranged, ok := targetType.(IntegerRangedType)

	// if this isn't an integer ranged type, report a mismatch
	if !ok {
		report(&TypeMismatchWithDescriptionError{
			ActualType:              targetType,
			ExpectedTypeDescription: "an integer type",
			Range:                   ast.NewRangeFromPositioned(memoryGauge, expression),
		})
	}
	minInt := ranged.MinInt()
	maxInt := ranged.MaxInt()

	if !checkIntegerRange(expression.Value, minInt, maxInt) {
		if report != nil {
			report(&InvalidIntegerLiteralRangeError{
				ExpectedType:   targetType,
				ExpectedMinInt: minInt,
				ExpectedMaxInt: maxInt,
				Range:          ast.NewRangeFromPositioned(memoryGauge, expression),
			})
		}

		return false
	}

	return true
}

// CheckFixedPointLiteral checks that the value of the fixed-point literal
// fits into range of the target fixed-point type
func CheckFixedPointLiteral(
	memoryGauge common.MemoryGauge,
	expression *ast.FixedPointExpression,
	targetType Type,
	report func(error),
) bool {

	// The target type might just be an integer type,
	// in which case only the integer range can be checked.

	switch targetType := targetType.(type) {
	case FractionalRangedType:
		minInt := targetType.MinInt()
		maxInt := targetType.MaxInt()
		scale := targetType.Scale()
		minFractional := targetType.MinFractional()
		maxFractional := targetType.MaxFractional()

		if expression.Scale > scale {
			if report != nil {
				report(&InvalidFixedPointLiteralScaleError{
					ExpectedType:  targetType,
					ExpectedScale: scale,
					Range:         ast.NewRangeFromPositioned(memoryGauge, expression),
				})
			}

			return false
		}

		if !fixedpoint.CheckRange(
			expression.Negative,
			expression.UnsignedInteger,
			expression.Fractional,
			minInt,
			minFractional,
			maxInt,
			maxFractional,
		) {
			if report != nil {
				report(&InvalidFixedPointLiteralRangeError{
					ExpectedType:          targetType,
					ExpectedMinInt:        minInt,
					ExpectedMinFractional: minFractional,
					ExpectedMaxInt:        maxInt,
					ExpectedMaxFractional: maxFractional,
					Range:                 ast.NewRangeFromPositioned(memoryGauge, expression),
				})
			}

			return false
		}

	case IntegerRangedType:
		minInt := targetType.MinInt()
		maxInt := targetType.MaxInt()

		integerValue := new(big.Int).Set(expression.UnsignedInteger)

		if expression.Negative {
			integerValue.Neg(expression.UnsignedInteger)
		}

		if !checkIntegerRange(integerValue, minInt, maxInt) {
			if report != nil {
				report(&InvalidIntegerLiteralRangeError{
					ExpectedType:   targetType,
					ExpectedMinInt: minInt,
					ExpectedMaxInt: maxInt,
					Range:          ast.NewRangeFromPositioned(memoryGauge, expression),
				})
			}

			return false
		}
	}

	return true
}

// CheckAddressLiteral checks that the value of the integer literal
// fits into the range of an address (64 bits), and is hexadecimal
func CheckAddressLiteral(memoryGauge common.MemoryGauge, expression *ast.IntegerExpression, report func(error)) bool {
	rangeMin := AddressTypeMinIntBig
	rangeMax := AddressTypeMaxIntBig

	valid := true

	if expression.Base != 16 {
		if report != nil {
			report(&InvalidAddressLiteralError{
				Range: ast.NewRangeFromPositioned(memoryGauge, expression),
			})
		}

		valid = false
	}

	if !checkIntegerRange(expression.Value, rangeMin, rangeMax) {
		if report != nil {
			report(&InvalidAddressLiteralError{
				Range: ast.NewRangeFromPositioned(memoryGauge, expression),
			})
		}

		valid = false
	}

	return valid
}

func checkIntegerRange(value, min, max *big.Int) bool {
	return (min == nil || value.Cmp(min) >= 0) &&
		(max == nil || value.Cmp(max) <= 0)
}

func (checker *Checker) declareGlobalDeclaration(declaration ast.Declaration) {
	identifier := declaration.DeclarationIdentifier()
	if identifier == nil {
		return
	}
	name := identifier.Identifier
	checker.declareGlobalValue(name)
	checker.declareGlobalType(name)
}

func (checker *Checker) declareGlobalValue(name string) {
	variable := checker.valueActivations.Find(name)
	if variable == nil {
		return
	}
	checker.Elaboration.SetGlobalValue(name, variable)
}

func (checker *Checker) declareGlobalType(name string) {
	ty := checker.typeActivations.Find(name)
	if ty == nil {
		return
	}
	checker.Elaboration.SetGlobalType(name, ty)
}

func (checker *Checker) checkResourceMoveOperation(valueExpression ast.Expression, valueType Type) {
	// The check is only necessary for resources.
	// Bail out early if the value is not a resource

	if !valueType.IsResourceType() {
		return
	}

	// Check the moved expression is wrapped in a unary expression with the move operation (<-).
	// Report an error if not and bail out if it is missing or another unary operator is used

	unaryExpression, ok := valueExpression.(*ast.UnaryExpression)
	if !ok || unaryExpression.Operation != ast.OperationMove {
		checker.report(
			&MissingMoveOperationError{
				Pos: valueExpression.StartPosition(),
			},
		)
		return
	}
}

func (checker *Checker) inLoop() bool {
	return checker.functionActivations.Current().InLoop()
}

func (checker *Checker) inSwitch() bool {
	return checker.functionActivations.Current().InSwitch()
}

func (checker *Checker) findAndCheckValueVariable(identifierExpression *ast.IdentifierExpression, recordOccurrence bool) *Variable {
	identifier := identifierExpression.Identifier
	variable := checker.valueActivations.Find(identifier.Identifier)
	if variable == nil {
		checker.report(
			&NotDeclaredError{
				ExpectedKind: common.DeclarationKindVariable,
				Name:         identifier.Identifier,
				Expression:   identifierExpression,
				Pos:          identifier.StartPosition(),
			},
		)
		return nil
	}

	if checker.PositionInfo != nil && recordOccurrence && identifier.Identifier != "" {
		checker.recordVariableReferenceOccurrence(
			identifier.StartPosition(),
			identifier.EndPosition(checker.memoryGauge),
			variable,
		)
	}

	return variable
}

// ConvertType converts an AST type representation to a sema type
func (checker *Checker) ConvertType(t ast.Type) Type {
	switch t := t.(type) {
	case *ast.NominalType:
		return checker.convertNominalType(t)

	case *ast.VariableSizedType:
		return checker.convertVariableSizedType(t)

	case *ast.ConstantSizedType:
		return checker.convertConstantSizedType(t)

	case *ast.FunctionType:
		return checker.convertFunctionType(t)

	case *ast.OptionalType:
		return checker.convertOptionalType(t)

	case *ast.DictionaryType:
		return checker.convertDictionaryType(t)

	case *ast.ReferenceType:
		return checker.convertReferenceType(t)

	case *ast.RestrictedType:
		return checker.convertRestrictedType(t)

	case *ast.InstantiationType:
		return checker.convertInstantiationType(t)

	case nil:
		// The AST might contain "holes" if parsing failed
		return InvalidType
	}

	panic(&astTypeConversionError{invalidASTType: t})
}

func CheckRestrictedType(
	memoryGauge common.MemoryGauge,
	restrictedType Type,
	restrictions []*InterfaceType,
	report func(func(*ast.RestrictedType) error),
) Type {
	restrictionRanges := make(map[*InterfaceType]func(*ast.RestrictedType) ast.Range, len(restrictions))
	restrictionsCompositeKind := common.CompositeKindUnknown
	memberSet := map[string]*InterfaceType{}

	for i, restrictionInterfaceType := range restrictions {
		restrictionCompositeKind := restrictionInterfaceType.CompositeKind

		if restrictionsCompositeKind == common.CompositeKindUnknown {
			restrictionsCompositeKind = restrictionCompositeKind

		} else if restrictionCompositeKind != restrictionsCompositeKind {
			report(func(t *ast.RestrictedType) error {
				return &RestrictionCompositeKindMismatchError{
					CompositeKind:         restrictionCompositeKind,
					PreviousCompositeKind: restrictionsCompositeKind,
					Range:                 ast.NewRangeFromPositioned(memoryGauge, t.Restrictions[i]),
				}
			})
		}

		// The restriction must not be duplicated

		if _, exists := restrictionRanges[restrictionInterfaceType]; exists {
			report(func(t *ast.RestrictedType) error {
				return &InvalidRestrictionTypeDuplicateError{
					Type:  restrictionInterfaceType,
					Range: ast.NewRangeFromPositioned(memoryGauge, t.Restrictions[i]),
				}
			})

		} else {
			restrictionRanges[restrictionInterfaceType] =
				func(t *ast.RestrictedType) ast.Range {
					return ast.NewRangeFromPositioned(memoryGauge, t.Restrictions[i])
				}
		}

		// The restrictions may not have clashing members

		// TODO: also include interface conformances' members
		//   once interfaces can have conformances

		restrictionInterfaceType.Members.Foreach(func(name string, member *Member) {
			if previousDeclaringInterfaceType, ok := memberSet[name]; ok {

				// If there is an overlap in members, ensure the members have the same type

				memberType := member.TypeAnnotation.Type

				prevMemberType, ok := previousDeclaringInterfaceType.Members.Get(name)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				previousMemberType := prevMemberType.TypeAnnotation.Type

				if !memberType.IsInvalidType() &&
					!previousMemberType.IsInvalidType() &&
					!memberType.Equal(previousMemberType) {

					report(func(t *ast.RestrictedType) error {
						return &RestrictionMemberClashError{
							Name:                  name,
							RedeclaringType:       restrictionInterfaceType,
							OriginalDeclaringType: previousDeclaringInterfaceType,
							Range:                 ast.NewRangeFromPositioned(memoryGauge, t.Restrictions[i]),
						}
					})
				}
			} else {
				memberSet[name] = restrictionInterfaceType
			}
		})
	}

	var hadExplicitType = restrictedType != nil

	if !hadExplicitType {
		// If no restricted type is given, infer `AnyResource`/`AnyStruct`
		// based on the composite kind of the restrictions.

		switch restrictionsCompositeKind {
		case common.CompositeKindUnknown:
			// If no restricted type is given, and also no restrictions,
			// the type is ambiguous.

			restrictedType = InvalidType

			report(func(t *ast.RestrictedType) error {
				return &AmbiguousRestrictedTypeError{Range: ast.NewRangeFromPositioned(memoryGauge, t)}
			})

		case common.CompositeKindResource:
			restrictedType = AnyResourceType

		case common.CompositeKindStructure:
			restrictedType = AnyStructType

		default:
			panic(errors.NewUnreachableError())
		}
	}

	// The restricted type must be a composite type
	// or `AnyResource`/`AnyStruct`

	reportInvalidRestrictedType := func() {
		report(func(t *ast.RestrictedType) error {
			return &InvalidRestrictedTypeError{
				Type:  restrictedType,
				Range: ast.NewRangeFromPositioned(memoryGauge, t.Type),
			}
		})
	}

	var compositeType *CompositeType

	if !restrictedType.IsInvalidType() {

		if typeResult, ok := restrictedType.(*CompositeType); ok {
			switch typeResult.Kind {

			case common.CompositeKindResource,
				common.CompositeKindStructure:

				compositeType = typeResult

			default:
				reportInvalidRestrictedType()
			}
		} else {

			switch restrictedType {
			case AnyResourceType, AnyStructType, AnyType:
				break

			default:
				if hadExplicitType {
					reportInvalidRestrictedType()
				}
			}
		}
	}

	// If the restricted type is a composite type,
	// check that the restrictions are conformances

	if compositeType != nil {

		// Prepare a set of all the conformances

		conformances := compositeType.ExplicitInterfaceConformanceSet()

		for _, restriction := range restrictions {
			// The restriction must be an explicit or implicit conformance
			// of the composite (restricted type)

			if !conformances.Contains(restriction) {
				report(func(t *ast.RestrictedType) error {
					return &InvalidNonConformanceRestrictionError{
						Type:  restriction,
						Range: restrictionRanges[restriction](t),
					}
				})
			}
		}
	}
	return restrictedType
}

func (checker *Checker) convertRestrictedType(t *ast.RestrictedType) Type {
	var restrictedType Type

	// Convert the restricted type, if any

	if t.Type != nil {
		restrictedType = checker.ConvertType(t.Type)
	}

	// Convert the restrictions

	var restrictions []*InterfaceType

	for _, restriction := range t.Restrictions {
		restrictionResult := checker.ConvertType(restriction)

		// The restriction must be a resource or structure interface type

		restrictionInterfaceType, ok := restrictionResult.(*InterfaceType)
		restrictionCompositeKind := common.CompositeKindUnknown
		if ok {
			restrictionCompositeKind = restrictionInterfaceType.CompositeKind
		}
		if !ok || (restrictionCompositeKind != common.CompositeKindResource &&
			restrictionCompositeKind != common.CompositeKindStructure) {

			if !restrictionResult.IsInvalidType() {
				checker.report(&InvalidRestrictionTypeError{
					Type:  restrictionResult,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, restriction),
				})
			}

			// NOTE: ignore this invalid type
			// and do not add it to the restrictions result
			continue
		}

		restrictions = append(restrictions, restrictionInterfaceType)
	}

	restrictedType = CheckRestrictedType(
		checker.memoryGauge,
		restrictedType,
		restrictions,
		func(getError func(*ast.RestrictedType) error) {
			checker.report(getError(t))
		},
	)

	return &RestrictedType{
		Type:         restrictedType,
		Restrictions: restrictions,
	}
}

func (checker *Checker) convertReferenceType(t *ast.ReferenceType) Type {

	var access Access = UnauthorizedAccess
	var ty Type

	if t.Authorization != nil {
		access = checker.accessFromAstAccess(ast.EntitlementAccess{EntitlementSet: t.Authorization.EntitlementSet})
		switch mapAccess := access.(type) {
		case EntitlementMapAccess:
			// mapped auth types are only allowed in the annotations of composite fields
			if checker.entitlementMappingInScope == nil || !checker.entitlementMappingInScope.Equal(mapAccess.Type) {
				checker.report(&InvalidMappedAuthorizationOutsideOfFieldError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, t),
					Map:   mapAccess.Type,
				})
				access = UnauthorizedAccess
			}
		}
	}

	ty = checker.ConvertType(t.Type)

	return &ReferenceType{
		Authorization: access,
		Type:          ty,
	}
}

func (checker *Checker) convertDictionaryType(t *ast.DictionaryType) Type {
	keyType := checker.ConvertType(t.KeyType)
	valueType := checker.ConvertType(t.ValueType)

	if !IsValidDictionaryKeyType(keyType) {
		checker.report(
			&InvalidDictionaryKeyTypeError{
				Type:  keyType,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, t.KeyType),
			},
		)
	}

	return &DictionaryType{
		KeyType:   keyType,
		ValueType: valueType,
	}
}

func (checker *Checker) convertOptionalType(t *ast.OptionalType) Type {
	// optional types annotations are special cased to not be considered nested so that
	// we can have mapped-entitlement optional reference fields
	ty := checker.ConvertType(t.Type)
	return &OptionalType{
		Type: ty,
	}
}

// convertFunctionType converts the given AST function type into a sema function type.
//
// NOTE: type annotations are *NOT* checked!
func (checker *Checker) convertFunctionType(t *ast.FunctionType) Type {
	parameterTypeAnnotations := t.ParameterTypeAnnotations

	var parameters []Parameter
	parameterCount := len(parameterTypeAnnotations)
	if parameterCount > 0 {
		parameters = make([]Parameter, 0, parameterCount)

		for _, parameterTypeAnnotation := range parameterTypeAnnotations {
			convertedParameterTypeAnnotation := checker.ConvertTypeAnnotation(parameterTypeAnnotation)
			parameters = append(
				parameters,
				Parameter{
					TypeAnnotation: convertedParameterTypeAnnotation,
				},
			)
		}
	}

	returnTypeAnnotation := checker.ConvertTypeAnnotation(t.ReturnTypeAnnotation)

	purity := PurityFromAnnotation(t.PurityAnnotation)

	return NewSimpleFunctionType(
		purity,
		parameters,
		returnTypeAnnotation,
	)
}

func (checker *Checker) convertConstantSizedType(t *ast.ConstantSizedType) Type {
	elementType := checker.ConvertType(t.Type)

	size := t.Size.Value

	if !t.Size.Value.IsInt64() || t.Size.Value.Sign() < 0 {
		minSize := new(big.Int)
		maxSize := new(big.Int).SetInt64(math.MaxInt64)

		checker.report(
			&InvalidConstantSizedTypeSizeError{
				ActualSize:     t.Size.Value,
				ExpectedMinInt: minSize,
				ExpectedMaxInt: maxSize,
				Range:          ast.NewRangeFromPositioned(checker.memoryGauge, t.Size),
			},
		)

		switch {
		case t.Size.Value.Cmp(minSize) < 0:
			size = minSize

		case t.Size.Value.Cmp(maxSize) > 0:
			size = maxSize
		}
	}

	finalSize := size.Int64()

	const expectedBase = 10
	if t.Size.Base != expectedBase {
		checker.report(
			&InvalidConstantSizedTypeBaseError{
				ActualBase:   t.Size.Base,
				ExpectedBase: expectedBase,
				Range:        ast.NewRangeFromPositioned(checker.memoryGauge, t.Size),
			},
		)
	}

	return &ConstantSizedType{
		Type: elementType,
		Size: finalSize,
	}
}

func (checker *Checker) convertVariableSizedType(t *ast.VariableSizedType) Type {
	elementType := checker.ConvertType(t.Type)
	return &VariableSizedType{
		Type: elementType,
	}
}

func (checker *Checker) findAndCheckTypeVariable(identifier ast.Identifier, recordOccurrence bool) *Variable {
	variable := checker.typeActivations.Find(identifier.Identifier)
	if variable == nil {
		checker.report(
			&NotDeclaredError{
				ExpectedKind: common.DeclarationKindType,
				Name:         identifier.Identifier,
				Pos:          identifier.StartPosition(),
			},
		)

		return nil
	}

	if checker.PositionInfo != nil && recordOccurrence && identifier.Identifier != "" {
		checker.recordVariableReferenceOccurrence(
			identifier.StartPosition(),
			identifier.EndPosition(checker.memoryGauge),
			variable,
		)
	}

	return variable
}

func (checker *Checker) convertNominalType(t *ast.NominalType) Type {
	variable := checker.findAndCheckTypeVariable(t.Identifier, true)
	if variable == nil {
		return InvalidType
	}

	ty := variable.Type

	var resolvedIdentifiers []ast.Identifier

	for _, identifier := range t.NestedIdentifiers {
		if containerType, ok := ty.(ContainerType); ok && containerType.IsContainerType() {
			ty, _ = containerType.GetNestedTypes().Get(identifier.Identifier)
		} else {
			if !ty.IsInvalidType() {
				checker.report(
					&InvalidNestedTypeError{
						Type: ast.NewNominalType(
							checker.memoryGauge,
							t.Identifier,
							resolvedIdentifiers,
						),
					},
				)
			}

			return InvalidType
		}

		resolvedIdentifiers = append(resolvedIdentifiers, identifier)

		if ty == nil {
			nonExistentType := ast.NewNominalType(
				checker.memoryGauge,
				t.Identifier,
				resolvedIdentifiers,
			)
			checker.report(
				&NotDeclaredError{
					ExpectedKind: common.DeclarationKindType,
					Name:         nonExistentType.String(),
					Pos:          t.StartPosition(),
				},
			)
			return InvalidType
		}
	}

	return ty
}

// ConvertTypeAnnotation converts an AST type annotation representation
// to a sema type annotation
//
// NOTE: type annotations are *NOT* checked!
func (checker *Checker) ConvertTypeAnnotation(typeAnnotation *ast.TypeAnnotation) TypeAnnotation {
	convertedType := checker.ConvertType(typeAnnotation.Type)
	return TypeAnnotation{
		IsResource: typeAnnotation.IsResource,
		Type:       convertedType,
	}
}

func (checker *Checker) functionType(
	purity ast.FunctionPurity,
	access Access,
	parameterList *ast.ParameterList,
	returnTypeAnnotation *ast.TypeAnnotation,
) *FunctionType {
	convertedParameters := checker.parameters(parameterList)

	convertedReturnTypeAnnotation := VoidTypeAnnotation
	if returnTypeAnnotation != nil {
		if mapAccess, isMapAccess := access.(EntitlementMapAccess); isMapAccess {
			checker.entitlementMappingInScope = mapAccess.Type
		}
		convertedReturnTypeAnnotation =
			checker.ConvertTypeAnnotation(returnTypeAnnotation)
		checker.entitlementMappingInScope = nil
	}

	return &FunctionType{
		Purity:               PurityFromAnnotation(purity),
		Parameters:           convertedParameters,
		ReturnTypeAnnotation: convertedReturnTypeAnnotation,
	}
}

func (checker *Checker) parameters(parameterList *ast.ParameterList) []Parameter {

	// TODO: required for initializer conformance checking at the moment, optimize/refactor
	var parameters = make([]Parameter, len(parameterList.Parameters))

	if len(parameterList.Parameters) > 0 {

		for i, parameter := range parameterList.Parameters {
			convertedParameterType := checker.ConvertType(parameter.TypeAnnotation.Type)

			// NOTE: copying resource annotation from source type annotation as-is,
			// so a potential error is properly reported

			parameters[i] = Parameter{
				Label:      parameter.Label,
				Identifier: parameter.Identifier.Identifier,
				TypeAnnotation: TypeAnnotation{
					IsResource: parameter.TypeAnnotation.IsResource,
					Type:       convertedParameterType,
				},
			}
		}
	}

	return parameters
}

func (checker *Checker) recordVariableReferenceOccurrence(startPos, endPos ast.Position, variable *Variable) {
	checker.PositionInfo.recordVariableReferenceOccurrence(
		checker.memoryGauge,
		startPos,
		endPos,
		variable,
	)
}

func (checker *Checker) recordVariableDeclarationOccurrence(name string, variable *Variable) {
	checker.PositionInfo.recordVariableDeclarationOccurrence(
		checker.memoryGauge,
		name,
		variable,
	)
}

func (checker *Checker) recordFieldDeclarationOrigin(
	identifier ast.Identifier,
	fieldType Type,
	docString string,
) *Origin {
	return checker.PositionInfo.recordFieldDeclarationOrigin(
		checker.memoryGauge,
		identifier,
		fieldType,
		docString,
	)
}

func (checker *Checker) recordFunctionDeclarationOrigin(
	function *ast.FunctionDeclaration,
	functionType *FunctionType,
) *Origin {
	return checker.PositionInfo.recordFunctionDeclarationOrigin(
		checker.memoryGauge,
		function,
		functionType,
	)
}

func (checker *Checker) enterValueScope() {
	//fmt.Printf("ENTER: %d\n", checker.valueActivations.Depth())
	checker.valueActivations.Enter()
}

func (checker *Checker) leaveValueScope(getEndPosition EndPositionGetter, checkResourceLoss bool) {
	if checkResourceLoss {
		checker.checkResourceLoss(checker.valueActivations.Depth())
	}

	checker.valueActivations.Leave(getEndPosition)
}

// TODO: prune resource variables declared in function's scope
//    from `checker.resources`, so they don't get checked anymore
//    when detecting resource use after invalidation in loops

// checkResourceLoss reports an error if there is a variable in the current scope
// that has a resource type and which was not moved or destroyed
func (checker *Checker) checkResourceLoss(depth int) {

	returnInfo := checker.functionActivations.Current().ReturnInfo
	if returnInfo.IsUnreachable() {
		return
	}

	checker.valueActivations.ForEachVariableDeclaredInAndBelow(depth, func(name string, variable *Variable) {

		if variable.Type.IsResourceType() &&
			variable.DeclarationKind != common.DeclarationKindSelf &&
			!checker.resources.Get(Resource{Variable: variable}).DefinitivelyInvalidated() {

			checker.report(
				&ResourceLossError{
					Range: ast.NewRange(
						checker.memoryGauge,
						*variable.Pos,
						variable.Pos.Shifted(checker.memoryGauge, len(name)-1),
					),
				},
			)
		}
	})
}

type recordedResourceInvalidation struct {
	resource     Resource
	invalidation ResourceInvalidation
}

func (checker *Checker) recordResourceInvalidation(
	expression ast.Expression,
	valueType Type,
	invalidationKind ResourceInvalidationKind,
) *recordedResourceInvalidation {

	if !(invalidationKind.IsDefinite() ||
		invalidationKind == ResourceInvalidationKindMoveTemporary) {

		panic(errors.NewUnexpectedError("invalidation should be recorded as definite or temporary"))
	}

	if !valueType.IsResourceType() {
		return nil
	}

	reportInvalidNestedMove := func() {
		checker.report(
			&InvalidNestedResourceMoveError{
				Range: ast.NewRangeFromPositioned(
					checker.memoryGauge,
					expression,
				),
			},
		)
	}

	accessedSelfMember := checker.accessedSelfMember(expression)

	switch expression.(type) {
	case *ast.MemberExpression:

		if accessedSelfMember == nil ||
			!checker.allowSelfResourceFieldInvalidation {

			reportInvalidNestedMove()
			return nil
		}

	case *ast.IndexExpression:
		reportInvalidNestedMove()
		return nil
	}

	invalidation := ResourceInvalidation{
		Kind:     invalidationKind,
		StartPos: expression.StartPosition(),
		EndPos:   expression.EndPosition(checker.memoryGauge),
	}

	if checker.allowSelfResourceFieldInvalidation && accessedSelfMember != nil {
		res := Resource{Member: accessedSelfMember}

		checker.maybeAddResourceInvalidation(res, invalidation)

		return &recordedResourceInvalidation{
			resource:     res,
			invalidation: invalidation,
		}
	}

	identifierExpression, ok := expression.(*ast.IdentifierExpression)
	if !ok {
		return nil
	}

	variable := checker.findAndCheckValueVariable(identifierExpression, false)
	if variable == nil {
		return nil
	}

	if invalidationKind != ResourceInvalidationKindMoveTemporary &&
		variable.DeclarationKind == common.DeclarationKindSelf {

		checker.report(
			&InvalidSelfInvalidationError{
				InvalidationKind: invalidationKind,
				Range: ast.NewRangeFromPositioned(
					checker.memoryGauge,
					expression,
				),
			},
		)
	}

	res := Resource{Variable: variable}

	checker.maybeAddResourceInvalidation(res, invalidation)

	return &recordedResourceInvalidation{
		resource:     res,
		invalidation: invalidation,
	}
}

func (checker *Checker) checkWithResources(
	check TypeCheckFunc,
	temporaryResources *Resources,
) Type {
	originalResources := checker.resources
	checker.resources = temporaryResources
	defer func() {
		checker.resources = originalResources
	}()

	return check()
}

func (checker *Checker) checkWithReturnInfo(
	check TypeCheckFunc,
	temporaryReturnInfo *ReturnInfo,
) Type {
	functionActivation := checker.functionActivations.Current()
	initialReturnInfo := functionActivation.ReturnInfo
	functionActivation.ReturnInfo = temporaryReturnInfo
	defer func() {
		functionActivation.ReturnInfo = initialReturnInfo
	}()

	return check()
}

func (checker *Checker) checkWithInitializedMembers(
	check TypeCheckFunc,
	temporaryInitializedMembers *persistent.OrderedSet[*Member],
) Type {
	if temporaryInitializedMembers != nil {
		functionActivation := checker.functionActivations.Current()
		initializationInfo := functionActivation.InitializationInfo
		initialInitializedMembers := initializationInfo.InitializedFieldMembers
		initializationInfo.InitializedFieldMembers = temporaryInitializedMembers
		defer func() {
			initializationInfo.InitializedFieldMembers = initialInitializedMembers
		}()
	}

	return check()
}

// checkUnusedExpressionResourceLoss checks for a resource loss caused by an expression
// which has a resource type, but will not be used after its evaluation,
// i.e. implicitly dropped at this point.
//
// For example, function invocations, array literals, or dictionary literals will cause a resource loss
// if the expression is accessed immediately: e.g.
//   - `returnResource()[0]`
//   - `[<-create R(), <-create R()][0]`
//   - in an equality binary expression: `f() != nil`
//
// Basically any expression that does not have an identifier as its "base" expression
// will cause the resource to be lost.
//
// Safe expressions are identifier expressions,
// an indexing expression into a safe expression,
// or a member access on a safe expression.
func (checker *Checker) checkUnusedExpressionResourceLoss(expressionType Type, expression ast.Expression) {
	if !expressionType.IsResourceType() {
		return
	}

	// Get the base expression of the given expression, i.e. get the accessed expression
	// as long as there is one.
	//
	// For example, in the expression `foo[0].bar`, both the wrapping member access
	// expression `bar` and the wrapping indexing expression `[0]` are removed,
	// leaving the base expression `foo`

	baseExpression := expression

	for {
		accessExpression, isAccess := baseExpression.(ast.AccessExpression)
		if !isAccess {
			break
		}
		baseExpression = accessExpression.AccessedExpression()
	}

	if _, isIdentifier := baseExpression.(*ast.IdentifierExpression); isIdentifier {
		return
	}

	checker.report(
		&ResourceLossError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
		},
	)
}

// checkResourceFieldNesting checks if any resource fields are nested
// in non resource composites (concrete or interface)
func (checker *Checker) checkResourceFieldNesting(
	members *StringMemberOrderedMap,
	compositeKind common.CompositeKind,
	baseType Type,
	fieldPositionGetter func(name string) ast.Position,
) {
	// Resource fields are only allowed in resources and contracts

	switch compositeKind {
	case common.CompositeKindResource,
		common.CompositeKindContract:

		return
	case common.CompositeKindAttachment:
		if baseType != nil && baseType.IsResourceType() {
			return
		}
	}

	// The field is not a resource or contract.
	// Check if there are any fields that have a resource type and report them

	members.Foreach(func(name string, member *Member) {
		// NOTE: check type, not resource annotation:
		// the field could have a wrong annotation

		if !member.TypeAnnotation.Type.IsResourceType() {
			return
		}

		// Skip enums' implicit rawValue field.
		// If a resource type is used as the enum raw type,
		// it is already reported

		if compositeKind == common.CompositeKindEnum &&
			name == EnumRawValueFieldName {

			return
		}

		pos := fieldPositionGetter(name)

		checker.report(
			&InvalidResourceFieldError{
				Name:          name,
				CompositeKind: compositeKind,
				Pos:           pos,
			},
		)
	})
}

// checkPotentiallyUnevaluated runs the given type checking function
// under the assumption that the checked expression might not be evaluated.
// That means that resource invalidation and returns are not definite,
// but only potential
func (checker *Checker) checkPotentiallyUnevaluated(check TypeCheckFunc) Type {
	functionActivation := checker.functionActivations.Current()

	initialReturnInfo := functionActivation.ReturnInfo
	temporaryReturnInfo := initialReturnInfo.Clone()

	var temporaryInitializedMembers *persistent.OrderedSet[*Member]
	if functionActivation.InitializationInfo != nil {
		initialInitializedMembers := functionActivation.InitializationInfo.InitializedFieldMembers
		temporaryInitializedMembers = initialInitializedMembers.Clone()
	}

	initialResources := checker.resources
	temporaryResources := initialResources.Clone()
	defer temporaryResources.Reclaim()

	result := checker.checkBranch(
		check,
		temporaryReturnInfo,
		temporaryInitializedMembers,
		temporaryResources,
	)

	functionActivation.ReturnInfo.MergePotentiallyUnevaluated(temporaryReturnInfo)

	checker.resources.MergeBranches(
		temporaryResources,
		temporaryReturnInfo,
		nil,
		nil,
	)

	return result
}

func (checker *Checker) ResetErrors() {
	checker.errors = nil
}

const invalidTypeDeclarationAccessModifierExplanation = "type declarations must be public"

func (checker *Checker) checkDeclarationAccessModifier(
	access Access,
	declarationKind common.DeclarationKind,
	declarationType Type,
	containerKind *common.CompositeKind,
	startPos ast.Position,
	isConstant bool,
) {
	if checker.functionActivations.IsLocal() {

		if !access.Equal(PrimitiveAccess(ast.AccessNotSpecified)) {
			checker.report(
				&InvalidAccessModifierError{
					Access:          access,
					Explanation:     "local declarations may not have an access modifier",
					DeclarationKind: declarationKind,
					Pos:             startPos,
				},
			)
		}
	} else {

		isTypeDeclaration := declarationKind.IsTypeDeclaration()

		switch access := access.(type) {
		case PrimitiveAccess:
			switch ast.PrimitiveAccess(access) {
			case ast.AccessPublicSettable:
				// Public settable access for a constant is not sensible
				// and type declarations must be public for now

				if isConstant || isTypeDeclaration {
					var explanation string
					switch {
					case isConstant:
						explanation = "constants can never be set"
					case isTypeDeclaration:
						explanation = invalidTypeDeclarationAccessModifierExplanation
					}

					checker.report(
						&InvalidAccessModifierError{
							Access:          access,
							Explanation:     explanation,
							DeclarationKind: declarationKind,
							Pos:             startPos,
						},
					)
				}

			case ast.AccessPrivate:
				// Type declarations must be public for now

				if isTypeDeclaration {

					checker.report(
						&InvalidAccessModifierError{
							Access:          access,
							Explanation:     invalidTypeDeclarationAccessModifierExplanation,
							DeclarationKind: declarationKind,
							Pos:             startPos,
						},
					)
				}

			case ast.AccessContract,
				ast.AccessAccount:

				// Type declarations must be public for now

				if isTypeDeclaration {
					checker.report(
						&InvalidAccessModifierError{
							Access:          access,
							Explanation:     invalidTypeDeclarationAccessModifierExplanation,
							DeclarationKind: declarationKind,
							Pos:             startPos,
						},
					)
				}

			case ast.AccessNotSpecified:

				// Type declarations cannot be effectively private for now

				if isTypeDeclaration &&
					checker.Config.AccessCheckMode == AccessCheckModeNotSpecifiedRestricted {

					checker.report(
						&MissingAccessModifierError{
							DeclarationKind: declarationKind,
							Explanation:     invalidTypeDeclarationAccessModifierExplanation,
							Pos:             startPos,
						},
					)
				}

				// In strict mode, access modifiers must be given

				if checker.Config.AccessCheckMode == AccessCheckModeStrict {
					checker.report(
						&MissingAccessModifierError{
							DeclarationKind: declarationKind,
							Pos:             startPos,
						},
					)
				}
			}

		case EntitlementMapAccess:
			// attachments may be declared with an entitlement map access
			if declarationKind == common.DeclarationKindAttachment {
				return
			}

			// otherwise, mapped entitlements may only be used in structs and resources
			if containerKind == nil ||
				(*containerKind != common.CompositeKindResource &&
					*containerKind != common.CompositeKindStructure) {
				checker.report(
					&InvalidMappedEntitlementMemberError{
						Pos: startPos,
					},
				)
				return
			}

			// mapped entitlement fields must be (optional) references that are authorized to the same mapped entitlement,
			// or functions that return an (optional) reference authorized to the same mapped entitlement
			requireIsPotentiallyOptionalReference := func(typ Type) {
				switch ty := typ.(type) {
				case *ReferenceType:
					if ty.Authorization.Equal(access) {
						return
					}
				case *OptionalType:
					switch optionalType := ty.Type.(type) {
					case *ReferenceType:
						if optionalType.Authorization.Equal(access) {
							return
						}
					}
				}
				checker.report(
					&InvalidMappedEntitlementMemberError{
						Pos: startPos,
					},
				)
			}

			switch ty := declarationType.(type) {
			case *FunctionType:
				if declarationKind == common.DeclarationKindFunction {
					requireIsPotentiallyOptionalReference(ty.ReturnTypeAnnotation.Type)
				} else {
					requireIsPotentiallyOptionalReference(ty)
				}
			default:
				requireIsPotentiallyOptionalReference(ty)
			}

		case EntitlementSetAccess:
			if containerKind == nil ||
				(*containerKind != common.CompositeKindResource &&
					*containerKind != common.CompositeKindStructure &&
					*containerKind != common.CompositeKindAttachment) {
				checker.report(
					&InvalidEntitlementAccessError{
						Pos: startPos,
					},
				)
				return
			}

			// when using entitlement set access, it is not permitted for the value to be declared with a mapped entitlement
			switch ty := declarationType.(type) {
			case *ReferenceType:
				if _, isMap := ty.Authorization.(EntitlementMapAccess); isMap {
					checker.report(
						&InvalidMappedEntitlementMemberError{
							Pos: startPos,
						},
					)
				}
			case *OptionalType:
				switch optionalType := ty.Type.(type) {
				case *ReferenceType:
					if _, isMap := optionalType.Authorization.(EntitlementMapAccess); isMap {
						checker.report(
							&InvalidMappedEntitlementMemberError{
								Pos: startPos,
							},
						)
					}
				}
			}
		}
	}
}

func (checker *Checker) checkFieldsAccessModifier(
	fields []*ast.FieldDeclaration,
	members *StringMemberOrderedMap,
	containerKind *common.CompositeKind,
) {
	for _, field := range fields {
		isConstant := field.VariableKind == ast.VariableKindConstant
		member, present := members.Get(field.Identifier.Identifier)
		if present {
			checker.checkDeclarationAccessModifier(
				member.Access,
				field.DeclarationKind(),
				member.TypeAnnotation.Type,
				containerKind,
				field.StartPos,
				isConstant,
			)
		}
	}
}

// checkCharacterLiteral checks that the string literal is a valid character,
// i.e. it has exactly one grapheme cluster.
func (checker *Checker) checkCharacterLiteral(expression *ast.StringExpression) {
	if IsValidCharacter(expression.Value) {
		return
	}

	checker.report(
		&InvalidCharacterLiteralError{
			Length: uniseg.GraphemeClusterCount(expression.Value),
			Range:  ast.NewRangeFromPositioned(checker.memoryGauge, expression),
		},
	)
}

func (checker *Checker) accessFromAstAccess(access ast.Access) (result Access) {

	switch access := access.(type) {
	case ast.PrimitiveAccess:
		return PrimitiveAccess(access)
	case ast.EntitlementAccess:

		semaAccess, hasAccess := checker.Elaboration.GetSemanticAccess(access)
		if hasAccess {
			return semaAccess
		}
		defer func() {
			checker.Elaboration.SetSemanticAccess(access, result)
		}()

		astEntitlements := access.EntitlementSet.Entitlements()
		nominalType := checker.convertNominalType(astEntitlements[0])

		switch nominalType := nominalType.(type) {
		case *EntitlementType:
			semanticEntitlements := make([]*EntitlementType, 0, len(astEntitlements))
			semanticEntitlements = append(semanticEntitlements, nominalType)

			for _, entitlement := range astEntitlements[1:] {
				nominalType := checker.convertNominalType(entitlement)
				entitlementType, ok := nominalType.(*EntitlementType)
				if !ok {
					// don't duplicate errors when the type here is invalid, as this will have triggered an error before
					if nominalType != InvalidType {
						checker.report(
							&InvalidNonEntitlementAccessError{
								Range: ast.NewRangeFromPositioned(checker.memoryGauge, entitlement),
							},
						)
					}
					result = PrimitiveAccess(ast.AccessNotSpecified)
					return
				}
				semanticEntitlements = append(semanticEntitlements, entitlementType)
			}
			if access.EntitlementSet.Separator() == "," {
				result = NewEntitlementSetAccess(semanticEntitlements, Conjunction)
				return
			}
			result = NewEntitlementSetAccess(semanticEntitlements, Disjunction)
			return
		case *EntitlementMapType:
			if len(astEntitlements) != 1 {
				checker.report(
					&InvalidMultipleMappedEntitlementError{
						Pos: astEntitlements[1].Identifier.Pos,
					},
				)
				result = PrimitiveAccess(ast.AccessNotSpecified)
				return
			}
			result = NewEntitlementMapAccess(nominalType)
			return
		default:
			// don't duplicate errors when the type here is invalid, as this will have triggered an error before
			if nominalType != InvalidType {
				checker.report(
					&InvalidNonEntitlementAccessError{
						Range: ast.NewRangeFromPositioned(checker.memoryGauge, astEntitlements[0]),
					},
				)
			}
			result = PrimitiveAccess(ast.AccessNotSpecified)
			return
		}
	}
	panic(errors.NewUnreachableError())
}

func (checker *Checker) withSelfResourceInvalidationAllowed(f func()) {
	allowSelfResourceFieldInvalidation := checker.allowSelfResourceFieldInvalidation
	checker.allowSelfResourceFieldInvalidation = true
	defer func() {
		checker.allowSelfResourceFieldInvalidation = allowSelfResourceFieldInvalidation
	}()

	f()
}

const ResourceOwnerFieldName = "owner"
const ResourceUUIDFieldName = "uuid"

const ContractAccountFieldName = "account"

const contractAccountFieldDocString = `
The account where the contract is deployed in
`

const resourceOwnerFieldDocString = `
The account owning the resource, i.e. the account that stores the account, or nil if the resource is not currently in storage
`

const resourceUUIDFieldDocString = `
The automatically generated, unique ID of the resource
`

func (checker *Checker) predeclaredMembers(containerType Type) []*Member {
	var predeclaredMembers []*Member

	addPredeclaredMember := func(
		identifier string,
		fieldType Type,
		declarationKind common.DeclarationKind,
		access ast.PrimitiveAccess,
		ignoreInSerialization bool,
		docString string,
	) {
		predeclaredMembers = append(predeclaredMembers, &Member{
			ContainerType:         containerType,
			Access:                PrimitiveAccess(access),
			Identifier:            ast.NewIdentifier(checker.memoryGauge, identifier, ast.EmptyPosition),
			DeclarationKind:       declarationKind,
			VariableKind:          ast.VariableKindConstant,
			TypeAnnotation:        NewTypeAnnotation(fieldType),
			Predeclared:           true,
			IgnoreInSerialization: ignoreInSerialization,
			DocString:             docString,
		})
	}

	// All types have a predeclared member `fun isInstance(_ type: Type): Bool`

	addPredeclaredMember(
		IsInstanceFunctionName,
		IsInstanceFunctionType,
		common.DeclarationKindFunction,
		ast.AccessPublic,
		true,
		isInstanceFunctionDocString,
	)

	// All types have a predeclared member `fun getType(): Type`

	addPredeclaredMember(
		GetTypeFunctionName,
		GetTypeFunctionType,
		common.DeclarationKindFunction,
		ast.AccessPublic,
		true,
		getTypeFunctionDocString,
	)

	if compositeKindedType, ok := containerType.(CompositeKindedType); ok {

		switch compositeKindedType.GetCompositeKind() {
		case common.CompositeKindContract:

			// All contracts have a predeclared member
			// `priv let account: AuthAccount`,
			// which is ignored in serialization

			addPredeclaredMember(
				ContractAccountFieldName,
				AuthAccountType,
				common.DeclarationKindField,
				ast.AccessPrivate,
				true,
				contractAccountFieldDocString,
			)

		case common.CompositeKindResource:

			// All resources have two predeclared fields:

			// `pub let owner: PublicAccount?`,
			// ignored in serialization

			addPredeclaredMember(
				ResourceOwnerFieldName,
				&OptionalType{
					Type: PublicAccountType,
				},
				common.DeclarationKindField,
				ast.AccessPublic,
				true,
				resourceOwnerFieldDocString,
			)

			// `pub let uuid: UInt64`,
			// included in serialization

			addPredeclaredMember(
				ResourceUUIDFieldName,
				UInt64Type,
				common.DeclarationKindField,
				ast.AccessPublic,
				false,
				resourceUUIDFieldDocString,
			)
		}

		if compositeKindedType.GetCompositeKind().SupportsAttachments() {
			addPredeclaredMember(
				CompositeForEachAttachmentFunctionName,
				CompositeForEachAttachmentFunctionType(compositeKindedType.GetCompositeKind()),
				common.DeclarationKindFunction,
				ast.AccessPublic,
				true,
				compositeForEachAttachmentFunctionDocString,
			)
		}
	}

	return predeclaredMembers
}

func (checker *Checker) checkVariableMove(expression ast.Expression) {

	identifierExpression, ok := expression.(*ast.IdentifierExpression)
	if !ok {
		return
	}

	variable := checker.valueActivations.Find(identifierExpression.Identifier.Identifier)
	if variable == nil {
		return
	}

	reportInvalidMove := func(declarationKind common.DeclarationKind) {
		checker.report(
			&InvalidMoveError{
				Name:            variable.Identifier,
				DeclarationKind: declarationKind,
				Pos:             identifierExpression.StartPosition(),
			},
		)
	}

	switch ty := variable.Type.(type) {
	case *TransactionType:
		reportInvalidMove(common.DeclarationKindTransaction)

	case CompositeKindedType:
		kind := ty.GetCompositeKind()
		if kind == common.CompositeKindContract {
			reportInvalidMove(common.DeclarationKindContract)
		}
	}
}

func (checker *Checker) rewritePostConditions(postConditions []*ast.Condition) PostConditionsRewrite {

	var beforeStatements []ast.Statement

	var rewrittenPostConditions []*ast.Condition

	count := len(postConditions)
	if count > 0 {
		rewrittenPostConditions = make([]*ast.Condition, count)

		beforeExtractor := checker.beforeExtractor()

		for i, postCondition := range postConditions {

			// copy condition and set expression to rewritten one
			newPostCondition := *postCondition

			testExtraction := beforeExtractor.ExtractBefore(postCondition.Test)

			extractedExpressions := testExtraction.ExtractedExpressions

			newPostCondition.Test = testExtraction.RewrittenExpression

			if postCondition.Message != nil {
				messageExtraction := beforeExtractor.ExtractBefore(postCondition.Message)

				newPostCondition.Message = messageExtraction.RewrittenExpression

				extractedExpressions = append(
					extractedExpressions,
					messageExtraction.ExtractedExpressions...,
				)
			}

			for _, extractedExpression := range extractedExpressions {
				expression := extractedExpression.Expression
				startPos := expression.StartPosition()

				// NOTE: no need to check the before statements or update elaboration here:
				// The before statements are visited/checked later
				variableDeclaration := ast.NewEmptyVariableDeclaration(checker.memoryGauge)
				variableDeclaration.StartPos = startPos
				variableDeclaration.Identifier = extractedExpression.Identifier
				variableDeclaration.Transfer = ast.NewTransfer(
					checker.memoryGauge,
					ast.TransferOperationCopy,
					startPos,
				)
				variableDeclaration.Value = expression

				beforeStatements = append(beforeStatements,
					variableDeclaration,
				)
			}

			rewrittenPostConditions[i] = &newPostCondition
		}
	}

	return PostConditionsRewrite{
		BeforeStatements:        beforeStatements,
		RewrittenPostConditions: rewrittenPostConditions,
	}
}

func (checker *Checker) checkTypeAnnotation(typeAnnotation TypeAnnotation, pos ast.HasPosition) {

	switch typeAnnotation.TypeAnnotationState() {
	case TypeAnnotationStateMissingResourceAnnotation:
		checker.report(
			&MissingResourceAnnotationError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, pos),
			},
		)

	case TypeAnnotationStateInvalidResourceAnnotation:
		checker.report(
			&InvalidResourceAnnotationError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, pos),
			},
		)

	case TypeAnnotationStateDirectEntitlementTypeAnnotation:
		checker.report(
			&DirectEntitlementAnnotationError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, pos),
			},
		)

	case TypeAnnotationStateDirectAttachmentTypeAnnotation:
		checker.report(
			&InvalidAttachmentAnnotationError{

				Range: ast.NewRangeFromPositioned(checker.memoryGauge, pos),
			},
		)
	}

	checker.checkInvalidInterfaceAsType(typeAnnotation.Type, pos)
}

func (checker *Checker) checkInvalidInterfaceAsType(ty Type, pos ast.HasPosition) {
	rewrittenType, rewritten := ty.RewriteWithRestrictedTypes()
	if rewritten {
		checker.report(
			&InvalidInterfaceTypeError{
				ActualType:   ty,
				ExpectedType: rewrittenType,
				Range: ast.NewRange(
					checker.memoryGauge,
					pos.StartPosition(),
					pos.EndPosition(checker.memoryGauge),
				),
			},
		)
	}
}

func (checker *Checker) ValueActivationDepth() int {
	return checker.valueActivations.Depth()
}

func (checker *Checker) TypeActivationDepth() int {
	return checker.typeActivations.Depth()
}

func (checker *Checker) effectiveMemberAccess(access Access, containerKind ContainerKind) Access {
	switch containerKind {
	case ContainerKindComposite:
		return checker.effectiveCompositeMemberAccess(access)
	case ContainerKindInterface:
		return checker.effectiveInterfaceMemberAccess(access)
	default:
		panic(errors.NewUnreachableError())
	}
}

func (checker *Checker) effectiveInterfaceMemberAccess(access Access) Access {
	if access.Equal(PrimitiveAccess(ast.AccessNotSpecified)) {
		return PrimitiveAccess(ast.AccessPublic)
	} else {
		return access
	}
}

func (checker *Checker) effectiveCompositeMemberAccess(access Access) Access {
	if !access.Equal(PrimitiveAccess(ast.AccessNotSpecified)) {
		return access
	}

	switch checker.Config.AccessCheckMode {
	case AccessCheckModeStrict, AccessCheckModeNotSpecifiedRestricted:
		return PrimitiveAccess(ast.AccessPrivate)

	case AccessCheckModeNotSpecifiedUnrestricted, AccessCheckModeNone:
		return PrimitiveAccess(ast.AccessPublic)

	default:
		panic(errors.NewUnreachableError())
	}
}

func (checker *Checker) convertInstantiationType(t *ast.InstantiationType) Type {

	ty := checker.ConvertType(t.Type)

	// Always convert (check) the type arguments,
	// even if the instantiated type is invalid

	var typeArgumentAnnotations []TypeAnnotation
	typeArgumentCount := len(t.TypeArguments)
	if typeArgumentCount > 0 {
		typeArgumentAnnotations = make([]TypeAnnotation, typeArgumentCount)

		for i, rawTypeArgument := range t.TypeArguments {
			typeArgument := checker.ConvertTypeAnnotation(rawTypeArgument)
			checker.checkTypeAnnotation(typeArgument, rawTypeArgument)
			typeArgumentAnnotations[i] = typeArgument
		}
	}

	parameterizedType, ok := ty.(ParameterizedType)
	if !ok {

		// The type is not parameterized,
		// report an error for all type arguments

		checker.report(
			&UnparameterizedTypeInstantiationError{
				Range: ast.NewRange(
					checker.memoryGauge,
					t.TypeArgumentsStartPos,
					t.EndPosition(checker.memoryGauge),
				),
			},
		)

		// Just return the converted instantiated type as-is

		return ty
	}

	typeParameters := parameterizedType.TypeParameters()
	typeParameterCount := len(typeParameters)

	typeArgumentAnnotationCount := len(typeArgumentAnnotations)
	var typeArguments []Type
	if typeArgumentAnnotationCount > 0 {
		typeArguments = make([]Type, typeArgumentAnnotationCount)

		for i, typeAnnotation := range typeArgumentAnnotations {
			typeArgument := typeAnnotation.Type
			typeArguments[i] = typeArgument

			// If the type parameter corresponding to the type argument (if any) has a type bound,
			// then check that the argument is a subtype of the type bound.

			if i < typeParameterCount {
				typeParameter := typeParameters[i]
				rawTypeArgument := t.TypeArguments[i]

				err := typeParameter.checkTypeBound(
					typeArgument,
					ast.NewRangeFromPositioned(checker.memoryGauge, rawTypeArgument),
				)
				checker.report(err)
			}
		}
	}

	if typeArgumentCount != typeParameterCount {

		// The instantiation has an incorrect number of type arguments

		checker.report(
			&InvalidTypeArgumentCountError{
				TypeParameterCount: typeParameterCount,
				TypeArgumentCount:  typeArgumentCount,
				Range: ast.NewRange(
					checker.memoryGauge,
					t.TypeArgumentsStartPos,
					t.EndPos,
				),
			},
		)

		// Just return the converted instantiated type as-is

		return ty
	}

	return parameterizedType.Instantiate(typeArguments, checker.report)
}

func (checker *Checker) VisitExpression(expr ast.Expression, expectedType Type) Type {
	// Always return 'visibleType' as the type of the expression,
	// to avoid bubbling up type-errors of inner expressions.
	visibleType, _ := checker.visitExpression(expr, expectedType)
	return visibleType
}

func (checker *Checker) visitExpression(expr ast.Expression, expectedType Type) (visibleType Type, actualType Type) {
	return checker.visitExpressionWithForceType(expr, expectedType, true)
}

func (checker *Checker) VisitExpressionWithForceType(expr ast.Expression, expectedType Type, forceType bool) Type {
	// Always return 'visibleType' as the type of the expression,
	// to avoid bubbling up type-errors of inner expressions.
	visibleType, _ := checker.visitExpressionWithForceType(expr, expectedType, forceType)
	return visibleType
}

// visitExpressionWithForceType
//
// Parameters:
// expr         - Expression to check
// expectedType - Contextually expected type of the expression
// forceType    - Specifies whether to use the expected type as a hard requirement (forceType = true)
//
//	or whether to use the expected type for type inferring only (forceType = false)
//
// Return types:
// visibleType - The type that others should 'see' as the type of this expression. This could be
//
//	used as the type of the expression to avoid the type errors being delegated up.
//
// actualType  - The actual type of the expression.
func (checker *Checker) visitExpressionWithForceType(
	expr ast.Expression,
	expectedType Type,
	forceType bool,
) (visibleType Type, actualType Type) {

	// Cache the current contextually expected type, and set the `expectedType`
	// as the new contextually expected type.
	prevExpectedType := checker.expectedType

	checker.expectedType = expectedType
	defer func() {
		// Restore the prev contextually expected type
		checker.expectedType = prevExpectedType
	}()

	actualType = ast.AcceptExpression[Type](expr, checker)

	if checker.Config.ExtendedElaborationEnabled {
		checker.Elaboration.SetExpressionTypes(
			expr,
			ExpressionTypes{
				ActualType:   actualType,
				ExpectedType: expectedType,
			},
		)
	}

	if forceType &&
		expectedType != nil &&
		!expectedType.IsInvalidType() &&
		actualType != InvalidType &&
		!IsSubType(actualType, expectedType) {

		checker.report(
			&TypeMismatchError{
				ExpectedType: expectedType,
				ActualType:   actualType,
				Expression:   expr,
				Range:        checker.expressionRange(expr),
			},
		)

		// If there are type mismatch errors, return the expected type as the visible-type of the expression.
		// This is done to avoid the same error getting delegated up.
		// i.e: Impact of the mismatched type would be local to that expression only.
		return expectedType, actualType
	}

	return actualType, actualType
}

func (checker *Checker) expressionRange(expression ast.Expression) ast.Range {
	if indexExpr, ok := expression.(*ast.IndexExpression); ok {
		return ast.NewRange(
			checker.memoryGauge,
			indexExpr.TargetExpression.StartPosition(),
			indexExpr.EndPosition(checker.memoryGauge),
		)
	} else {
		return ast.NewRangeFromPositioned(checker.memoryGauge, expression)
	}
}

func (checker *Checker) declareGlobalRanges() {
	memoryGauge := checker.memoryGauge

	_ = BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
		checker.PositionInfo.recordGlobalRange(memoryGauge, name, variable)
		return nil
	})

	_ = BaseValueActivation.ForEach(func(name string, variable *Variable) error {
		checker.PositionInfo.recordGlobalRange(memoryGauge, name, variable)
		return nil
	})

	checker.Elaboration.ForEachGlobalType(func(name string, variable *Variable) {
		checker.PositionInfo.recordGlobalRange(memoryGauge, name, variable)
	})

	checker.Elaboration.ForEachGlobalValue(func(name string, variable *Variable) {
		checker.PositionInfo.recordGlobalRange(memoryGauge, name, variable)
	})
}

var errFoundJump = goErrors.New("jump found")

func (checker *Checker) maybeAddResourceInvalidation(resource Resource, invalidation ResourceInvalidation) {
	functionActivation := checker.functionActivations.Current()

	returnInfo := functionActivation.ReturnInfo

	// Resource invalidations are only definite
	// if the invalidation can be definitely reached.

	if returnInfo.IsUnreachable() {
		return
	}

	var onlyPotential bool
	switch {
	case resource.Member != nil:
		onlyPotential = returnInfo.MaybeReturned || returnInfo.MaybeJumped()

	case resource.Variable != nil &&
		resource.Variable.DeclarationKind != common.DeclarationKindSelf:

		declarationOffset := resource.Variable.Pos.Offset
		invalidationOffset := invalidation.StartPos.Offset

		err := returnInfo.JumpOffsets.ForEach(func(jumpOffset int) error {
			if declarationOffset < jumpOffset && jumpOffset < invalidationOffset {
				return errFoundJump
			}
			return nil
		})

		onlyPotential = err == errFoundJump
	}

	if onlyPotential {
		invalidation.Kind = invalidation.Kind.AsPotential()
	}

	// Maybe record the invalidation.
	// If there had already been an invalidation before, the new invalidation is ignored.
	// However, the repeated invalidation is still reported as an error,
	// but as a use-after invalidation error.

	checker.resources.MaybeRecordInvalidation(resource, invalidation)
}

func (checker *Checker) beforeExtractor() *BeforeExtractor {
	if checker._beforeExtractor == nil {
		checker._beforeExtractor = NewBeforeExtractor(checker.memoryGauge, checker.report)
	}
	return checker._beforeExtractor
}

func wrapWithOptionalIfNotNil(typ Type) Type {
	if typ == nil {
		return nil
	}

	return &OptionalType{
		Type: typ,
	}
}

func (checker *Checker) CheckStatement(element ast.Statement) {
	ast.AcceptStatement[struct{}](element, checker)
}

func (checker *Checker) checkStaticModifier(isStatic bool, position ast.HasPosition) {
	if isStatic && !checker.Config.AllowStaticDeclarations {
		checker.report(
			&InvalidStaticModifierError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, position),
			},
		)
	}
}

func (checker *Checker) checkNativeModifier(isNative bool, position ast.HasPosition) {
	if isNative && !checker.Config.AllowNativeDeclarations {
		checker.report(
			&InvalidNativeModifierError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, position),
			},
		)
	}
}

func (checker *Checker) isAvailableMember(expressionType Type, identifier string) bool {
	if expressionType == AuthAccountType &&
		identifier == AuthAccountTypeLinkAccountFunctionName {

		return checker.Config.AccountLinkingEnabled
	}

	return true
}
