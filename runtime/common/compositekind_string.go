// Code generated by "stringer -type=CompositeKind"; DO NOT EDIT.

package common

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CompositeKindUnknown-0]
	_ = x[CompositeKindStructure-1]
	_ = x[CompositeKindResource-2]
	_ = x[CompositeKindContract-3]
	_ = x[CompositeKindEvent-4]
	_ = x[CompositeKindEnum-5]
	_ = x[CompositeKindAttachment-6]
}

const _CompositeKind_name = "CompositeKindUnknownCompositeKindStructureCompositeKindResourceCompositeKindContractCompositeKindEventCompositeKindEnumCompositeKindAttachment"

var _CompositeKind_index = [...]uint8{0, 20, 42, 63, 84, 102, 119, 142}

func (i CompositeKind) String() string {
	if i >= CompositeKind(len(_CompositeKind_index)-1) {
		return "CompositeKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _CompositeKind_name[_CompositeKind_index[i]:_CompositeKind_index[i+1]]
}
