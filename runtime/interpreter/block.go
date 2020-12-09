package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	runtimeErrors "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// Block

type BlockValue struct {
	Height    UInt64Value
	View      UInt64Value
	ID        *ArrayValue
	Timestamp Fix64Value
}

func (BlockValue) IsValue() {}

func (v BlockValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitValue(interpreter, v)
}

func (BlockValue) DynamicType(_ *Interpreter) DynamicType {
	return BlockDynamicType{}
}

func (BlockValue) StaticType() StaticType {
	return PrimitiveStaticTypeBlock
}

func (v BlockValue) Copy() Value {
	return v
}

func (BlockValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (BlockValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (BlockValue) IsModified() bool {
	return false
}

func (BlockValue) SetModified(_ bool) {
	// NO-OP
}

func (v BlockValue) GetMember(_ *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case "height":
		return v.Height

	case "view":
		return v.View

	case "id":
		return v.ID

	case "timestamp":
		return v.Timestamp
	}

	return nil
}

func (v BlockValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) {
	panic(runtimeErrors.NewUnreachableError())
}

func (v BlockValue) IDAsByteArray() [sema.BlockIDSize]byte {
	var byteArray [sema.BlockIDSize]byte
	for i, b := range v.ID.Values {
		byteArray[i] = byte(b.(UInt8Value))
	}
	return byteArray
}

func (v BlockValue) String() string {
	return fmt.Sprintf(
		"Block(height: %s, view: %s, id: 0x%x, timestamp: %s)",
		v.Height,
		v.View,
		v.IDAsByteArray(),
		v.Timestamp,
	)
}
