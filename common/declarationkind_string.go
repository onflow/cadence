// Code generated by "stringer -type=DeclarationKind"; DO NOT EDIT.

package common

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[DeclarationKindUnknown-0]
	_ = x[DeclarationKindValue-1]
	_ = x[DeclarationKindFunction-2]
	_ = x[DeclarationKindVariable-3]
	_ = x[DeclarationKindConstant-4]
	_ = x[DeclarationKindType-5]
	_ = x[DeclarationKindParameter-6]
	_ = x[DeclarationKindArgumentLabel-7]
	_ = x[DeclarationKindStructure-8]
	_ = x[DeclarationKindResource-9]
	_ = x[DeclarationKindContract-10]
	_ = x[DeclarationKindEvent-11]
	_ = x[DeclarationKindField-12]
	_ = x[DeclarationKindInitializer-13]
	_ = x[DeclarationKindDestructorLegacy-14]
	_ = x[DeclarationKindStructureInterface-15]
	_ = x[DeclarationKindResourceInterface-16]
	_ = x[DeclarationKindContractInterface-17]
	_ = x[DeclarationKindEntitlement-18]
	_ = x[DeclarationKindEntitlementMapping-19]
	_ = x[DeclarationKindImport-20]
	_ = x[DeclarationKindSelf-21]
	_ = x[DeclarationKindBase-22]
	_ = x[DeclarationKindTransaction-23]
	_ = x[DeclarationKindPrepare-24]
	_ = x[DeclarationKindExecute-25]
	_ = x[DeclarationKindTypeParameter-26]
	_ = x[DeclarationKindPragma-27]
	_ = x[DeclarationKindEnum-28]
	_ = x[DeclarationKindEnumCase-29]
	_ = x[DeclarationKindAttachment-30]
}

const _DeclarationKind_name = "DeclarationKindUnknownDeclarationKindValueDeclarationKindFunctionDeclarationKindVariableDeclarationKindConstantDeclarationKindTypeDeclarationKindParameterDeclarationKindArgumentLabelDeclarationKindStructureDeclarationKindResourceDeclarationKindContractDeclarationKindEventDeclarationKindFieldDeclarationKindInitializerDeclarationKindDestructorLegacyDeclarationKindStructureInterfaceDeclarationKindResourceInterfaceDeclarationKindContractInterfaceDeclarationKindEntitlementDeclarationKindEntitlementMappingDeclarationKindImportDeclarationKindSelfDeclarationKindBaseDeclarationKindTransactionDeclarationKindPrepareDeclarationKindExecuteDeclarationKindTypeParameterDeclarationKindPragmaDeclarationKindEnumDeclarationKindEnumCaseDeclarationKindAttachment"

var _DeclarationKind_index = [...]uint16{0, 22, 42, 65, 88, 111, 130, 154, 182, 206, 229, 252, 272, 292, 318, 349, 382, 414, 446, 472, 505, 526, 545, 564, 590, 612, 634, 662, 683, 702, 725, 750}

func (i DeclarationKind) String() string {
	if i >= DeclarationKind(len(_DeclarationKind_index)-1) {
		return "DeclarationKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _DeclarationKind_name[_DeclarationKind_index[i]:_DeclarationKind_index[i+1]]
}