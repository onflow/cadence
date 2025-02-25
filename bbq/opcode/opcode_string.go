// Code generated by "stringer -type=Opcode"; DO NOT EDIT.

package opcode

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Unknown-0]
	_ = x[Return-1]
	_ = x[ReturnValue-2]
	_ = x[Jump-3]
	_ = x[JumpIfFalse-4]
	_ = x[JumpIfNil-5]
	_ = x[Add-11]
	_ = x[Subtract-12]
	_ = x[Multiply-13]
	_ = x[Divide-14]
	_ = x[Mod-15]
	_ = x[Less-20]
	_ = x[Greater-21]
	_ = x[LessOrEqual-22]
	_ = x[GreaterOrEqual-23]
	_ = x[Equal-31]
	_ = x[NotEqual-32]
	_ = x[Not-33]
	_ = x[Unwrap-37]
	_ = x[Destroy-38]
	_ = x[Transfer-39]
	_ = x[SimpleCast-40]
	_ = x[FailableCast-41]
	_ = x[ForceCast-42]
	_ = x[True-50]
	_ = x[False-51]
	_ = x[New-52]
	_ = x[Path-53]
	_ = x[Nil-54]
	_ = x[NewArray-55]
	_ = x[NewDictionary-56]
	_ = x[NewRef-57]
	_ = x[GetConstant-70]
	_ = x[GetLocal-71]
	_ = x[SetLocal-72]
	_ = x[GetGlobal-73]
	_ = x[SetGlobal-74]
	_ = x[GetField-75]
	_ = x[SetField-76]
	_ = x[SetIndex-77]
	_ = x[GetIndex-78]
	_ = x[Invoke-90]
	_ = x[InvokeDynamic-91]
	_ = x[Drop-100]
	_ = x[Dup-101]
	_ = x[Iterator-108]
	_ = x[IteratorHasNext-109]
	_ = x[IteratorNext-110]
	_ = x[EmitEvent-111]
}

const (
	_Opcode_name_0 = "UnknownReturnReturnValueJumpJumpIfFalseJumpIfNil"
	_Opcode_name_1 = "AddSubtractMultiplyDivideMod"
	_Opcode_name_2 = "LessGreaterLessOrEqualGreaterOrEqual"
	_Opcode_name_3 = "EqualNotEqualNot"
	_Opcode_name_4 = "UnwrapDestroyTransferSimpleCastFailableCastForceCast"
	_Opcode_name_5 = "TrueFalseNewPathNilNewArrayNewDictionaryNewRef"
	_Opcode_name_6 = "GetConstantGetLocalSetLocalGetGlobalSetGlobalGetFieldSetFieldSetIndexGetIndex"
	_Opcode_name_7 = "InvokeInvokeDynamic"
	_Opcode_name_8 = "DropDup"
	_Opcode_name_9 = "IteratorIteratorHasNextIteratorNextEmitEvent"
)

var (
	_Opcode_index_0 = [...]uint8{0, 7, 13, 24, 28, 39, 48}
	_Opcode_index_1 = [...]uint8{0, 3, 11, 19, 25, 28}
	_Opcode_index_2 = [...]uint8{0, 4, 11, 22, 36}
	_Opcode_index_3 = [...]uint8{0, 5, 13, 16}
	_Opcode_index_4 = [...]uint8{0, 6, 13, 21, 31, 43, 52}
	_Opcode_index_5 = [...]uint8{0, 4, 9, 12, 16, 19, 27, 40, 46}
	_Opcode_index_6 = [...]uint8{0, 11, 19, 27, 36, 45, 53, 61, 69, 77}
	_Opcode_index_7 = [...]uint8{0, 6, 19}
	_Opcode_index_8 = [...]uint8{0, 4, 7}
	_Opcode_index_9 = [...]uint8{0, 8, 23, 35, 44}
)

func (i Opcode) String() string {
	switch {
	case i <= 5:
		return _Opcode_name_0[_Opcode_index_0[i]:_Opcode_index_0[i+1]]
	case 11 <= i && i <= 15:
		i -= 11
		return _Opcode_name_1[_Opcode_index_1[i]:_Opcode_index_1[i+1]]
	case 20 <= i && i <= 23:
		i -= 20
		return _Opcode_name_2[_Opcode_index_2[i]:_Opcode_index_2[i+1]]
	case 31 <= i && i <= 33:
		i -= 31
		return _Opcode_name_3[_Opcode_index_3[i]:_Opcode_index_3[i+1]]
	case 37 <= i && i <= 42:
		i -= 37
		return _Opcode_name_4[_Opcode_index_4[i]:_Opcode_index_4[i+1]]
	case 50 <= i && i <= 57:
		i -= 50
		return _Opcode_name_5[_Opcode_index_5[i]:_Opcode_index_5[i+1]]
	case 70 <= i && i <= 78:
		i -= 70
		return _Opcode_name_6[_Opcode_index_6[i]:_Opcode_index_6[i+1]]
	case 90 <= i && i <= 91:
		i -= 90
		return _Opcode_name_7[_Opcode_index_7[i]:_Opcode_index_7[i+1]]
	case 100 <= i && i <= 101:
		i -= 100
		return _Opcode_name_8[_Opcode_index_8[i]:_Opcode_index_8[i+1]]
	case 108 <= i && i <= 111:
		i -= 108
		return _Opcode_name_9[_Opcode_index_9[i]:_Opcode_index_9[i+1]]
	default:
		return "Opcode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
