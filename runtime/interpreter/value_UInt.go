package interpreter

import (
	"math/big"
	"unsafe"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// UIntValue

type UIntValue struct {
	BigInt *big.Int
}

const uint64Size = int(unsafe.Sizeof(uint64(0)))

var uint64BigIntMemoryUsage = common.NewBigIntMemoryUsage(uint64Size)

func NewUIntValueFromUint64(memoryGauge common.MemoryGauge, value uint64) UIntValue {
	return NewUIntValueFromBigInt(
		memoryGauge,
		uint64BigIntMemoryUsage,
		func() *big.Int {
			return new(big.Int).SetUint64(value)
		},
	)
}

func NewUnmeteredUIntValueFromUint64(value uint64) UIntValue {
	return NewUnmeteredUIntValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUIntValueFromBigInt(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) UIntValue {
	common.UseMemory(memoryGauge, memoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredUIntValueFromBigInt(value)
}

func NewUnmeteredUIntValueFromBigInt(value *big.Int) UIntValue {
	return UIntValue{
		BigInt: value,
	}
}

func ConvertUInt(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UIntValue {
	switch value := value.(type) {
	case BigNumberValue:
		return NewUIntValueFromBigInt(
			memoryGauge,
			common.NewBigIntMemoryUsage(value.ByteLength()),
			func() *big.Int {
				v := value.ToBigInt(memoryGauge)
				if v.Sign() < 0 {
					panic(UnderflowError{
						LocationRange: locationRange,
					})
				}
				return v
			},
		)

	case NumberValue:
		v := value.ToInt(locationRange)
		if v < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return NewUIntValueFromUint64(
			memoryGauge,
			uint64(v),
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

var _ Value = UIntValue{}
var _ atree.Storable = UIntValue{}
var _ NumberValue = UIntValue{}
var _ IntegerValue = UIntValue{}
var _ EquatableValue = UIntValue{}
var _ ComparableValue = UIntValue{}
var _ HashableValue = UIntValue{}
var _ MemberAccessibleValue = UIntValue{}

func (UIntValue) isValue() {}

func (v UIntValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUIntValue(interpreter, v)
}

func (UIntValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UIntValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt)
}

func (v UIntValue) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UIntValue) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v UIntValue) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v UIntValue) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v UIntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v UIntValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UIntValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UIntValue) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UIntValue) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewPlusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Add(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Plus(interpreter, other, locationRange)
}

func (v UIntValue) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewMinusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			res.Sub(v.BigInt, o.BigInt)
			// INT30-C
			if res.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return res
		},
	)
}

func (v UIntValue) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewMinusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			res.Sub(v.BigInt, o.BigInt)
			// INT30-C
			if res.Sign() < 0 {
				return sema.UIntTypeMin
			}
			return res
		},
	)
}

func (v UIntValue) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewModBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			// INT33-C
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			res.Rem(v.BigInt, o.BigInt)
			return res
		},
	)
}

func (v UIntValue) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewMulBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Mul(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Mul(interpreter, other, locationRange)
}

func (v UIntValue) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewDivBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			// INT33-C
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UIntValue) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v UIntValue) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v UIntValue) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v UIntValue) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v UIntValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt, ok := other.(UIntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherUInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt (1 byte)
// - big int value encoded in big-endian (n bytes)
func (v UIntValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeUInt)
	copy(buffer[1:], b)
	return buffer
}

func (v UIntValue) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseOrBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseXorBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseAndBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseLeftShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))

		},
	)
}

func (v UIntValue) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseRightShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v UIntValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UIntType, locationRange)
}

func (UIntValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UIntValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UIntValue) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UIntValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v UIntValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (UIntValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UIntValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UIntValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v UIntValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredUIntValueFromBigInt(v.BigInt)
}

func (UIntValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UIntValue) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v UIntValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UIntValue) ChildStorables() []atree.Storable {
	return nil
}
