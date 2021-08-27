package ast

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/onflow/cadence/runtime/common"
)

func jsonMarshalAndVerify(i interface{}) (b []byte, err error) {
	if b, err = json.Marshal(i); err == nil {
		expected := string(b)
		roundtripped := reflect.New(reflect.TypeOf(i)).Interface()
		if err = json.Unmarshal(b, &roundtripped); err == nil {
			if b, err = json.Marshal(roundtripped); err == nil {
				actual := string(b)
				if expected != actual {
					return nil, fmt.Errorf("un/marshal failed:\n%s\n%s\n", expected, actual)
				}
			}
		}
	}
	return
}

//////////////////////////////////////////////////////////////////////////////

var accessKindMap = map[string]Access{
	"AccessNotSpecified":   AccessNotSpecified,
	"AccessPrivate":        AccessPrivate,
	"AccessContract":       AccessContract,
	"AccessAccount":        AccessAccount,
	"AccessPublic":         AccessPublic,
	"AccessPublicSettable": AccessPublicSettable,
}

var conditionKindMap = map[string]ConditionKind{
	"ConditionKindUnknown": ConditionKindUnknown,
	"ConditionKindPre":     ConditionKindPre,
	"ConditionKindPost":    ConditionKindPost,
}

var expressionTypeMap = map[string]func() Expression{
	"ArrayExpression":       func() Expression { return &ArrayExpression{} },
	"BinaryExpression":      func() Expression { return &BinaryExpression{} },
	"BoolExpression":        func() Expression { return &BoolExpression{} },
	"CastingExpression":     func() Expression { return &CastingExpression{} },
	"ConditionalExpression": func() Expression { return &ConditionalExpression{} },
	"CreateExpression":      func() Expression { return &CreateExpression{} },
	"DestroyExpression":     func() Expression { return &DestroyExpression{} },
	"DictionaryExpression":  func() Expression { return &DictionaryExpression{} },
	"FixedPointExpression":  func() Expression { return &FixedPointExpression{} },
	"ForceExpression":       func() Expression { return &ForceExpression{} },
	"FunctionExpression":    func() Expression { return &FunctionExpression{} },
	"IdentifierExpression":  func() Expression { return &IdentifierExpression{} },
	"IndexExpression":       func() Expression { return &IndexExpression{} },
	"IntegerExpression":     func() Expression { return &IntegerExpression{} },
	"InvocationExpression":  func() Expression { return &InvocationExpression{} },
	"MemberExpression":      func() Expression { return &MemberExpression{} },
	"NilExpression":         func() Expression { return &NilExpression{} },
	"PathExpression":        func() Expression { return &PathExpression{} },
	"ReferenceExpression":   func() Expression { return &ReferenceExpression{} },
	"StringExpression":      func() Expression { return &StringExpression{} },
	"UnaryExpression":       func() Expression { return &UnaryExpression{} },
}

var locationKindMap = map[string]func() common.Location{
	"AddressLocation":     func() common.Location { return &common.AddressLocation{} },
	"REPLLocation":        func() common.Location { return &common.REPLLocation{} },
	"ScriptLocation":      func() common.Location { return &common.ScriptLocation{} },
	"TransactionLocation": func() common.Location { return &common.TransactionLocation{} },
	"StringLocation":      func() common.Location { return nil },
	"FlowLocation":        func() common.Location { return nil },
	"IdentifierLocation":  func() common.Location { return nil },
}

var operationMap = map[string]Operation{
	"OperationUnknown":           OperationUnknown,
	"OperationOr":                OperationOr,
	"OperationAnd":               OperationAnd,
	"OperationEqual":             OperationEqual,
	"OperationNotEqual":          OperationNotEqual,
	"OperationLess":              OperationLess,
	"OperationGreater":           OperationGreater,
	"OperationLessEqual":         OperationLessEqual,
	"OperationGreaterEqual":      OperationGreaterEqual,
	"OperationPlus":              OperationPlus,
	"OperationMinus":             OperationMinus,
	"OperationMul":               OperationMul,
	"OperationDiv":               OperationDiv,
	"OperationMod":               OperationMod,
	"OperationNegate":            OperationNegate,
	"OperationNilCoalesce":       OperationNilCoalesce,
	"OperationMove":              OperationMove,
	"OperationCast":              OperationCast,
	"OperationFailableCast":      OperationFailableCast,
	"OperationForceCast":         OperationForceCast,
	"OperationBitwiseOr":         OperationBitwiseOr,
	"OperationBitwiseXor":        OperationBitwiseXor,
	"OperationBitwiseAnd":        OperationBitwiseAnd,
	"OperationBitwiseLeftShift":  OperationBitwiseLeftShift,
	"OperationBitwiseRightShift": OperationBitwiseRightShift,
}

var statementKindMap = map[string]func() Statement{
	"AssignmentStatement":        func() Statement { return &AssignmentStatement{} },
	"BreakStatement":             func() Statement { return &BreakStatement{} },
	"CompositeDeclaration":       func() Statement { return &CompositeDeclaration{} },
	"ContinueStatement":          func() Statement { return &ContinueStatement{} },
	"EmitStatement":              func() Statement { return &EmitStatement{} },
	"ExpressionStatement":        func() Statement { return &ExpressionStatement{} },
	"ForStatement":               func() Statement { return &ForStatement{} },
	"FunctionDeclaration":        func() Statement { return &FunctionDeclaration{} },
	"IfStatement":                func() Statement { return &IfStatement{} },
	"ImportDeclaration":          func() Statement { return &ImportDeclaration{} },
	"InterfaceDeclaration":       func() Statement { return &InterfaceDeclaration{} },
	"PragmaDeclaration":          func() Statement { return &PragmaDeclaration{} },
	"ReturnStatement":            func() Statement { return &ReturnStatement{} },
	"SpecialFunctionDeclaration": func() Statement { return &SpecialFunctionDeclaration{} },
	"SwapStatement":              func() Statement { return &SwapStatement{} },
	"SwitchStatement":            func() Statement { return &SwitchStatement{} },
	"TransactionDeclaration":     func() Statement { return &TransactionDeclaration{} },
	"VariableDeclaration":        func() Statement { return &VariableDeclaration{} },
	"WhileStatement":             func() Statement { return &WhileStatement{} },
}

var transferOperationMap = map[string]TransferOperation{
	"TransferOperationUnknown":    TransferOperationUnknown,
	"TransferOperationCopy":       TransferOperationCopy,
	"TransferOperationMove":       TransferOperationMove,
	"TransferOperationMoveForced": TransferOperationMoveForced,
}

var typeNameMap = map[string]func() Type{
	"ConstantSizedType": func() Type { return &ConstantSizedType{} },
	"DictionaryType":    func() Type { return &DictionaryType{} },
	"FunctionType":      func() Type { return &FunctionType{} },
	"InstantiationType": func() Type { return &InstantiationType{} },
	"NominalType":       func() Type { return &NominalType{} },
	"OptionalType":      func() Type { return &OptionalType{} },
	"ReferenceType":     func() Type { return &ReferenceType{} },
	"RestrictedType":    func() Type { return &RestrictedType{} },
	"VariableSizedType": func() Type { return &VariableSizedType{} },
}

var variableKindMap = map[string]VariableKind{
	"VariableKindNotSpecified": VariableKindNotSpecified,
	"VariableKindVariable":     VariableKindVariable,
	"VariableKindConstant":     VariableKindConstant,
}

//////////////////////////////////////////////////////////////////////////////

type BigIntUnmarshaler struct{ Int **big.Int }
type ExpressionUnmarshaler struct{ Expression *Expression }
type ExpressionsUnmarshaler struct{ Expressions *[]Expression }
type LocationUnmarshaler struct{ Location *common.Location }
type StatementUnmarshaler struct{ Statement *Statement }
type StatementsUnmarshaler struct{ Statements *[]Statement }
type TypeUnmarshaler struct{ Type *Type }

func (a *BigIntUnmarshaler) UnmarshalJSON(b []byte) (err error) {
	s := string(b)
	if s, err = strconv.Unquote(s); err != nil {
		return err
	} else if s == "null" {
		*a.Int = nil
		return nil
	}
	if z, ok := (&big.Int{}).SetString(s, 10); !ok {
		return fmt.Errorf("invalid BigInt %s", s)
	} else {
		*a.Int = z
		return nil
	}
}

func (a *ExpressionUnmarshaler) UnmarshalJSON(b []byte) error {
	var expressionTypeName string
	err := json.Unmarshal(b, &struct{ Type *string }{&expressionTypeName})
	if err != nil {
		return err
	}
	if et, ok := expressionTypeMap[expressionTypeName]; ok {
		*a.Expression = et()
		return json.Unmarshal(b, &a.Expression)
	}
	return fmt.Errorf("unknown Expression type")
}

func (a *ExpressionsUnmarshaler) UnmarshalJSON(b []byte) error {
	var rms []json.RawMessage
	err := json.Unmarshal(b, &rms)
	if err != nil {
		return err
	}
	*a.Expressions = make([]Expression, len(rms))
	for i, v := range rms {
		err := json.Unmarshal(v, &ExpressionUnmarshaler{&(*a.Expressions)[i]})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *LocationUnmarshaler) UnmarshalJSON(b []byte) error {
	var locationKind string
	var stringLocation common.StringLocation = ""
	err := json.Unmarshal(b, &struct {
		Type   *string
		String *common.StringLocation
	}{Type: &locationKind, String: &stringLocation})
	if err != nil {
		return err
	}
	if stringLocation != "" {
		*a.Location = stringLocation
		return nil
	}
	if f, ok := locationKindMap[locationKind]; ok {
		*a.Location = f()
		return nil
	}
	return fmt.Errorf("unknown Location")
}

func (a *StatementUnmarshaler) UnmarshalJSON(b []byte) error {
	var statementKindName string
	if err := json.Unmarshal(b, &struct{ Type *string }{&statementKindName}); err != nil {
		return err
	}
	if sk, ok := statementKindMap[statementKindName]; ok {
		*a.Statement = sk()
		return json.Unmarshal(b, &a.Statement)
	}
	return fmt.Errorf("unknown Statement Kind")
}

func (a *StatementsUnmarshaler) UnmarshalJSON(b []byte) error {
	var rms []json.RawMessage
	err := json.Unmarshal(b, &rms)
	if err != nil {
		return err
	}
	*a.Statements = make([]Statement, len(rms))
	for i, v := range rms {
		err := json.Unmarshal(v, &StatementUnmarshaler{&(*a.Statements)[i]})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *TypeUnmarshaler) UnmarshalJSON(b []byte) error {
	var typeName string
	err := json.Unmarshal(b, &struct{ Type *string }{&typeName})
	if err != nil {
		return err
	}
	if f, ok := typeNameMap[typeName]; ok {
		*a.Type = f()
		return json.Unmarshal(b, &a.Type)
	}
	return fmt.Errorf("unknown Type")
}

//////////////////////////////////////////////////////////////////////////////

func (a *Access) UnmarshalJSON(b []byte) error {
	if c, ok := accessKindMap[strings.Trim(string(b), `"`)]; ok {
		*a = c
		return nil
	}
	return fmt.Errorf("unknown Access kind")
}

func (a *Argument) UnmarshalJSON(b []byte) error {
	type Alias Argument
	err := json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
	})
	return err
}

func (a *AssignmentStatement) UnmarshalJSON(b []byte) error {
	type Alias AssignmentStatement
	return json.Unmarshal(b, &struct {
		*Alias
		Target   ExpressionUnmarshaler
		Value    ExpressionUnmarshaler
		StartPos *Position
		EndPos   *Position
	}{
		Alias:  (*Alias)(a),
		Target: ExpressionUnmarshaler{&a.Target},
		Value:  ExpressionUnmarshaler{&a.Value},
	})
}

func (a *Block) UnmarshalJSON(bs []byte) error {
	type Alias Block
	err := json.Unmarshal(bs, &struct {
		*Alias
		Statements StatementsUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Statements: StatementsUnmarshaler{&a.Statements},
	})
	return err
}

func (a *BreakStatement) UnmarshalJSON(b []byte) error {
	type Alias BreakStatement
	return json.Unmarshal(b, &struct {
		*Alias
	}{
		Alias: (*Alias)(a)})
}

func (a *CompositeDeclaration) UnmarshalJSON(b []byte) error {
	type Alias CompositeDeclaration
	return json.Unmarshal(b, &struct {
		*Alias
	}{
		Alias: (*Alias)(a)})
}

func (a *Condition) UnmarshalJSON(b []byte) error {
	type Alias Condition
	err := json.Unmarshal(b, &struct {
		*Alias
		Test    ExpressionUnmarshaler
		Message ExpressionUnmarshaler
	}{
		Alias:   (*Alias)(a),
		Test:    ExpressionUnmarshaler{&a.Test},
		Message: ExpressionUnmarshaler{&a.Message},
	})
	return err
}

func (a *ConditionKind) UnmarshalJSON(b []byte) error {
	if c, ok := conditionKindMap[strings.Trim(string(b), `"`)]; ok {
		*a = c
		return nil
	}
	return fmt.Errorf("unknown ConditionKind")
}

func (a *ConstantSizedType) UnmarshalJSON(b []byte) error {
	type Alias ConstantSizedType
	return json.Unmarshal(b, &struct {
		*Alias
		ElementType TypeUnmarshaler
	}{
		Alias:       (*Alias)(a),
		ElementType: TypeUnmarshaler{Type: &a.Type},
	})
}

func (a *DictionaryType) UnmarshalJSON(b []byte) error {
	type Alias DictionaryType
	return json.Unmarshal(b, &struct {
		*Alias
		KeyType   TypeUnmarshaler
		ValueType TypeUnmarshaler
	}{
		Alias:     (*Alias)(a),
		KeyType:   TypeUnmarshaler{&a.KeyType},
		ValueType: TypeUnmarshaler{&a.ValueType},
	})
}

func (a *DictionaryEntry) UnmarshalJSON(b []byte) error {
	type Alias DictionaryEntry
	return json.Unmarshal(b, &struct {
		*Alias
		Key   ExpressionUnmarshaler
		Value ExpressionUnmarshaler
	}{
		Alias: (*Alias)(a),
		Key:   ExpressionUnmarshaler{&a.Key},
		Value: ExpressionUnmarshaler{&a.Value},
	})
}

func (a *EmitStatement) UnmarshalJSON(b []byte) error {
	type Alias EmitStatement
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.StartPos,
	})
}

func (a *ExpressionStatement) UnmarshalJSON(b []byte) error {
	type Alias ExpressionStatement
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
	})
}

func (a *ForStatement) UnmarshalJSON(b []byte) error {
	type Alias ForStatement
	return json.Unmarshal(b, &struct {
		*Alias
		Value    ExpressionUnmarshaler
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		Value:    ExpressionUnmarshaler{&a.Value},
		StartPos: &a.StartPos,
	})
}

func (a *FunctionDeclaration) UnmarshalJSON(b []byte) error {
	type Alias FunctionDeclaration
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.StartPos,
	})
}

func (a *IfStatement) UnmarshalJSON(b []byte) error {
	type Alias IfStatement
	a.Test = &BoolExpression{}
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.StartPos,
	})
}

func (a *ImportDeclaration) UnmarshalJSON(b []byte) error {
	type Alias ImportDeclaration
	return json.Unmarshal(b, &struct {
		*Alias
		Location LocationUnmarshaler
	}{
		Alias:    (*Alias)(a),
		Location: LocationUnmarshaler{&a.Location},
	})
}

func (a *InstantiationType) UnmarshalJSON(b []byte) error {
	type Alias InstantiationType
	return json.Unmarshal(b, &struct {
		*Alias
		InstantiatedType TypeUnmarshaler
		EndPos           *Position
	}{
		Alias:            (*Alias)(a),
		InstantiatedType: TypeUnmarshaler{&a.Type},
		EndPos:           &a.EndPos,
	})
}

func (a *Members) UnmarshalJSON(b []byte) error {
	type Alias Members
	return json.Unmarshal(b, &struct {
		*Alias
		Declarations *[]Declaration
	}{
		Alias:        (*Alias)(a),
		Declarations: &a.declarations,
	})
}

func (a *NominalType) UnmarshalJSON(b []byte) error {
	type Alias NominalType
	return json.Unmarshal(b, &struct {
		*Alias
	}{
		Alias: (*Alias)(a)})
}

func (a *Operation) UnmarshalJSON(b []byte) error {
	if op, ok := operationMap[strings.Trim(string(b), `"`)]; ok {
		*a = op
		return nil
	}
	return fmt.Errorf("unknown Operation %s", string(b))
}

func (a *OptionalType) UnmarshalJSON(b []byte) error {
	type Alias OptionalType
	return json.Unmarshal(b, &struct {
		*Alias
		ElementType TypeUnmarshaler
		EndPos      *Position
	}{
		Alias:       (*Alias)(a),
		ElementType: TypeUnmarshaler{&a.Type},
		EndPos:      &a.EndPos,
	})
}

func (a *PragmaDeclaration) UnmarshalJSON(b []byte) error {
	type Alias PragmaDeclaration
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
	})
}

func (a *Program) UnmarshalJSON(b []byte) error {
	type Alias Program
	return json.Unmarshal(b, &struct {
		*Alias
		Declarations *[]Declaration
	}{
		Alias:        (*Alias)(a),
		Declarations: &a.declarations,
	})
}

func (a *ReferenceType) UnmarshalJSON(b []byte) error {
	type Alias ReferenceType
	return json.Unmarshal(b, &struct {
		*Alias
		ReferencedType TypeUnmarshaler
		StartPos       *Position
	}{
		Alias:          (*Alias)(a),
		ReferencedType: TypeUnmarshaler{&a.Type},
		StartPos:       &a.StartPos,
	})
}

func (a *RestrictedType) UnmarshalJSON(b []byte) error {
	type Alias RestrictedType
	return json.Unmarshal(b, &struct {
		*Alias
		RestrictedType TypeUnmarshaler
	}{
		Alias:          (*Alias)(a),
		RestrictedType: TypeUnmarshaler{&a.Type},
	})
}

func (a *ReturnStatement) UnmarshalJSON(b []byte) error {
	type Alias ReturnStatement
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
	})
}

func (a *SwapStatement) UnmarshalJSON(b []byte) error {
	type Alias SwapStatement
	return json.Unmarshal(b, &struct {
		*Alias
		Left  ExpressionUnmarshaler
		Right ExpressionUnmarshaler
	}{
		Alias: (*Alias)(a),
		Left:  ExpressionUnmarshaler{&a.Left},
		Right: ExpressionUnmarshaler{&a.Right},
	})
}

func (a *SwitchStatement) UnmarshalJSON(b []byte) error {
	type Alias SwitchStatement
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
	})
}

func (a *Transfer) UnmarshalJSON(b []byte) error {
	type Alias Transfer
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.Pos,
	})
}

func (a *TransferOperation) UnmarshalJSON(b []byte) error {
	if op, ok := transferOperationMap[strings.Trim(string(b), `"`)]; ok {
		*a = op
		return nil
	}
	return fmt.Errorf("unknown TransferOperation")
}

func (a *TypeAnnotation) UnmarshalJSON(b []byte) error {
	type Alias TypeAnnotation
	return json.Unmarshal(b, &struct {
		*Alias
		AnnotatedType TypeUnmarshaler
		StartPos      *Position
	}{
		Alias:         (*Alias)(a),
		AnnotatedType: TypeUnmarshaler{&a.Type},
		StartPos:      &a.StartPos,
	})
}

func (a *VariableDeclaration) UnmarshalJSON(b []byte) error {
	type Alias VariableDeclaration
	return json.Unmarshal(b, &struct {
		*Alias
		Value       ExpressionUnmarshaler
		SecondValue ExpressionUnmarshaler
		StartPos    *Position
	}{
		Alias:       (*Alias)(a),
		Value:       ExpressionUnmarshaler{&a.Value},
		SecondValue: ExpressionUnmarshaler{&a.SecondValue},
		StartPos:    &a.StartPos,
	})
}

func (a *VariableKind) UnmarshalJSON(b []byte) error {
	if vk, ok := variableKindMap[strings.Trim(string(b), `"`)]; ok {
		*a = vk
		return nil
	}
	return fmt.Errorf("unknown VariableKind")
}

func (a *VariableSizedType) UnmarshalJSON(b []byte) error {
	type Alias VariableSizedType
	return json.Unmarshal(b, &struct {
		*Alias
		ElementType TypeUnmarshaler
	}{
		Alias:       (*Alias)(a),
		ElementType: TypeUnmarshaler{&a.Type},
	})
}

func (a *WhileStatement) UnmarshalJSON(b []byte) error {
	type Alias WhileStatement
	return json.Unmarshal(b, &struct {
		*Alias
		Test     ExpressionUnmarshaler
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		Test:     ExpressionUnmarshaler{&a.Test},
		StartPos: &a.StartPos,
	})
}

//////////////////////////////////////////////////////////////////////////////

func (a *ArrayExpression) UnmarshalJSON(b []byte) error {
	type Alias ArrayExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Values ExpressionsUnmarshaler
	}{
		Alias:  (*Alias)(a),
		Values: ExpressionsUnmarshaler{&a.Values},
	})
}

func (a *BinaryExpression) UnmarshalJSON(b []byte) error {
	type Alias BinaryExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Left  ExpressionUnmarshaler
		Right ExpressionUnmarshaler
	}{
		Alias: (*Alias)(a),
		Left:  ExpressionUnmarshaler{&a.Left},
		Right: ExpressionUnmarshaler{&a.Right},
	})
}

func (a *CastingExpression) UnmarshalJSON(b []byte) error {
	type Alias CastingExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
	})
}

func (a *ConditionalExpression) UnmarshalJSON(b []byte) error {
	type Alias ConditionalExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Test ExpressionUnmarshaler
		Then ExpressionUnmarshaler
		Else ExpressionUnmarshaler
	}{
		Alias: (*Alias)(a),
		Test:  ExpressionUnmarshaler{&a.Test},
		Then:  ExpressionUnmarshaler{&a.Then},
		Else:  ExpressionUnmarshaler{&a.Else},
	})
}

func (a *CreateExpression) UnmarshalJSON(b []byte) error {
	type Alias CreateExpression
	return json.Unmarshal(b, &struct {
		*Alias
		InvocationExpression **InvocationExpression
		StartPos             *Position
	}{
		Alias:                (*Alias)(a),
		InvocationExpression: &a.InvocationExpression,
		StartPos:             &a.StartPos,
	})
}

func (a *DestroyExpression) UnmarshalJSON(b []byte) error {
	type Alias DestroyExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
		StartPos   *Position
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
		StartPos:   &a.StartPos,
	})
}

func (a *FixedPointExpression) UnmarshalJSON(b []byte) error {
	type Alias FixedPointExpression
	return json.Unmarshal(b, &struct {
		*Alias
		UnsignedInteger BigIntUnmarshaler
		Fractional      BigIntUnmarshaler
	}{
		Alias:           (*Alias)(a),
		UnsignedInteger: BigIntUnmarshaler{&a.UnsignedInteger},
		Fractional:      BigIntUnmarshaler{&a.Fractional},
	})
}

func (a *ForceExpression) UnmarshalJSON(b []byte) error {
	type Alias ForceExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
		EndPos     *Position
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
		EndPos:     &a.EndPos,
	})
}

func (a *FunctionExpression) UnmarshalJSON(b []byte) error {
	type Alias FunctionExpression
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.StartPos,
	})
}

func (a *Identifier) UnmarshalJSON(b []byte) error {
	type Alias Identifier
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.Pos,
	})
}

func (a *IndexExpression) UnmarshalJSON(b []byte) error {
	type Alias IndexExpression
	return json.Unmarshal(b, &struct {
		*Alias
		TargetExpression   ExpressionUnmarshaler
		IndexingExpression ExpressionUnmarshaler
	}{
		Alias:              (*Alias)(a),
		TargetExpression:   ExpressionUnmarshaler{&a.TargetExpression},
		IndexingExpression: ExpressionUnmarshaler{&a.IndexingExpression},
	})
}

func (a *IntegerExpression) UnmarshalJSON(b []byte) error {
	type Alias IntegerExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Value BigIntUnmarshaler
	}{
		Alias: (*Alias)(a),
		Value: BigIntUnmarshaler{Int: &a.Value},
	})
}

func (a *InvocationExpression) UnmarshalJSON(b []byte) error {
	type Alias InvocationExpression
	return json.Unmarshal(b, &struct {
		*Alias
		InvokedExpression ExpressionUnmarshaler
		EndPos            *Position
	}{
		Alias:             (*Alias)(a),
		InvokedExpression: ExpressionUnmarshaler{&a.InvokedExpression},
		EndPos:            &a.EndPos,
	})
}

func (a *MemberExpression) UnmarshalJSON(b []byte) error {
	type Alias MemberExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
	})
}

func (a *NilExpression) UnmarshalJSON(b []byte) error {
	type Alias NilExpression
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.Pos,
	})
}

func (a *PathExpression) UnmarshalJSON(b []byte) error {
	type Alias PathExpression
	return json.Unmarshal(b, &struct {
		*Alias
		StartPos *Position
	}{
		Alias:    (*Alias)(a),
		StartPos: &a.StartPos,
	})
}

func (a *ReferenceExpression) UnmarshalJSON(b []byte) error {
	type Alias ReferenceExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
		TargetType TypeUnmarshaler
		StartPos   *Position
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
		TargetType: TypeUnmarshaler{&a.Type},
		StartPos:   &a.StartPos,
	})
}

func (a *UnaryExpression) UnmarshalJSON(b []byte) error {
	type Alias UnaryExpression
	return json.Unmarshal(b, &struct {
		*Alias
		Expression ExpressionUnmarshaler
		StartPos   *Position
	}{
		Alias:      (*Alias)(a),
		Expression: ExpressionUnmarshaler{&a.Expression},
		StartPos:   &a.StartPos,
	})
}
