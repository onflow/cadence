// Code generated by "stringer -type=MemoryKind -trimprefix=MemoryKind"; DO NOT EDIT.

package common

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[MemoryKindUnknown-0]
	_ = x[MemoryKindAddressValue-1]
	_ = x[MemoryKindStringValue-2]
	_ = x[MemoryKindCharacterValue-3]
	_ = x[MemoryKindNumberValue-4]
	_ = x[MemoryKindArrayValueBase-5]
	_ = x[MemoryKindDictionaryValueBase-6]
	_ = x[MemoryKindCompositeValueBase-7]
	_ = x[MemoryKindSimpleCompositeValueBase-8]
	_ = x[MemoryKindOptionalValue-9]
	_ = x[MemoryKindTypeValue-10]
	_ = x[MemoryKindPathValue-11]
	_ = x[MemoryKindCapabilityValue-12]
	_ = x[MemoryKindStorageReferenceValue-13]
	_ = x[MemoryKindEphemeralReferenceValue-14]
	_ = x[MemoryKindInterpretedFunctionValue-15]
	_ = x[MemoryKindHostFunctionValue-16]
	_ = x[MemoryKindBoundFunctionValue-17]
	_ = x[MemoryKindBigInt-18]
	_ = x[MemoryKindSimpleCompositeValue-19]
	_ = x[MemoryKindPublishedValue-20]
	_ = x[MemoryKindStorageCapabilityControllerValue-21]
	_ = x[MemoryKindAccountCapabilityControllerValue-22]
	_ = x[MemoryKindAtreeArrayDataSlab-23]
	_ = x[MemoryKindAtreeArrayMetaDataSlab-24]
	_ = x[MemoryKindAtreeArrayElementOverhead-25]
	_ = x[MemoryKindAtreeMapDataSlab-26]
	_ = x[MemoryKindAtreeMapMetaDataSlab-27]
	_ = x[MemoryKindAtreeMapElementOverhead-28]
	_ = x[MemoryKindAtreeMapPreAllocatedElement-29]
	_ = x[MemoryKindAtreeEncodedSlab-30]
	_ = x[MemoryKindPrimitiveStaticType-31]
	_ = x[MemoryKindCompositeStaticType-32]
	_ = x[MemoryKindInterfaceStaticType-33]
	_ = x[MemoryKindVariableSizedStaticType-34]
	_ = x[MemoryKindConstantSizedStaticType-35]
	_ = x[MemoryKindDictionaryStaticType-36]
	_ = x[MemoryKindInclusiveRangeStaticType-37]
	_ = x[MemoryKindOptionalStaticType-38]
	_ = x[MemoryKindIntersectionStaticType-39]
	_ = x[MemoryKindEntitlementSetStaticAccess-40]
	_ = x[MemoryKindEntitlementMapStaticAccess-41]
	_ = x[MemoryKindReferenceStaticType-42]
	_ = x[MemoryKindCapabilityStaticType-43]
	_ = x[MemoryKindFunctionStaticType-44]
	_ = x[MemoryKindCadenceVoidValue-45]
	_ = x[MemoryKindCadenceOptionalValue-46]
	_ = x[MemoryKindCadenceBoolValue-47]
	_ = x[MemoryKindCadenceStringValue-48]
	_ = x[MemoryKindCadenceCharacterValue-49]
	_ = x[MemoryKindCadenceAddressValue-50]
	_ = x[MemoryKindCadenceIntValue-51]
	_ = x[MemoryKindCadenceNumberValue-52]
	_ = x[MemoryKindCadenceArrayValueBase-53]
	_ = x[MemoryKindCadenceArrayValueLength-54]
	_ = x[MemoryKindCadenceDictionaryValue-55]
	_ = x[MemoryKindCadenceInclusiveRangeValue-56]
	_ = x[MemoryKindCadenceKeyValuePair-57]
	_ = x[MemoryKindCadenceStructValueBase-58]
	_ = x[MemoryKindCadenceStructValueSize-59]
	_ = x[MemoryKindCadenceResourceValueBase-60]
	_ = x[MemoryKindCadenceAttachmentValueBase-61]
	_ = x[MemoryKindCadenceResourceValueSize-62]
	_ = x[MemoryKindCadenceAttachmentValueSize-63]
	_ = x[MemoryKindCadenceEventValueBase-64]
	_ = x[MemoryKindCadenceEventValueSize-65]
	_ = x[MemoryKindCadenceContractValueBase-66]
	_ = x[MemoryKindCadenceContractValueSize-67]
	_ = x[MemoryKindCadenceEnumValueBase-68]
	_ = x[MemoryKindCadenceEnumValueSize-69]
	_ = x[MemoryKindCadencePathValue-70]
	_ = x[MemoryKindCadenceTypeValue-71]
	_ = x[MemoryKindCadenceCapabilityValue-72]
	_ = x[MemoryKindCadenceFunctionValue-73]
	_ = x[MemoryKindCadenceOptionalType-74]
	_ = x[MemoryKindCadenceVariableSizedArrayType-75]
	_ = x[MemoryKindCadenceConstantSizedArrayType-76]
	_ = x[MemoryKindCadenceDictionaryType-77]
	_ = x[MemoryKindCadenceInclusiveRangeType-78]
	_ = x[MemoryKindCadenceField-79]
	_ = x[MemoryKindCadenceParameter-80]
	_ = x[MemoryKindCadenceTypeParameter-81]
	_ = x[MemoryKindCadenceStructType-82]
	_ = x[MemoryKindCadenceResourceType-83]
	_ = x[MemoryKindCadenceAttachmentType-84]
	_ = x[MemoryKindCadenceEventType-85]
	_ = x[MemoryKindCadenceContractType-86]
	_ = x[MemoryKindCadenceStructInterfaceType-87]
	_ = x[MemoryKindCadenceResourceInterfaceType-88]
	_ = x[MemoryKindCadenceContractInterfaceType-89]
	_ = x[MemoryKindCadenceFunctionType-90]
	_ = x[MemoryKindCadenceEntitlementSetAccess-91]
	_ = x[MemoryKindCadenceEntitlementMapAccess-92]
	_ = x[MemoryKindCadenceReferenceType-93]
	_ = x[MemoryKindCadenceIntersectionType-94]
	_ = x[MemoryKindCadenceCapabilityType-95]
	_ = x[MemoryKindCadenceEnumType-96]
	_ = x[MemoryKindRawString-97]
	_ = x[MemoryKindAddressLocation-98]
	_ = x[MemoryKindBytes-99]
	_ = x[MemoryKindVariable-100]
	_ = x[MemoryKindCompositeTypeInfo-101]
	_ = x[MemoryKindCompositeField-102]
	_ = x[MemoryKindInvocation-103]
	_ = x[MemoryKindStorageMap-104]
	_ = x[MemoryKindStorageKey-105]
	_ = x[MemoryKindTypeToken-106]
	_ = x[MemoryKindErrorToken-107]
	_ = x[MemoryKindSpaceToken-108]
	_ = x[MemoryKindProgram-109]
	_ = x[MemoryKindIdentifier-110]
	_ = x[MemoryKindArgument-111]
	_ = x[MemoryKindBlock-112]
	_ = x[MemoryKindFunctionBlock-113]
	_ = x[MemoryKindParameter-114]
	_ = x[MemoryKindParameterList-115]
	_ = x[MemoryKindTypeParameter-116]
	_ = x[MemoryKindTypeParameterList-117]
	_ = x[MemoryKindTransfer-118]
	_ = x[MemoryKindMembers-119]
	_ = x[MemoryKindTypeAnnotation-120]
	_ = x[MemoryKindDictionaryEntry-121]
	_ = x[MemoryKindFunctionDeclaration-122]
	_ = x[MemoryKindCompositeDeclaration-123]
	_ = x[MemoryKindAttachmentDeclaration-124]
	_ = x[MemoryKindInterfaceDeclaration-125]
	_ = x[MemoryKindEntitlementDeclaration-126]
	_ = x[MemoryKindEntitlementMappingElement-127]
	_ = x[MemoryKindEntitlementMappingDeclaration-128]
	_ = x[MemoryKindEnumCaseDeclaration-129]
	_ = x[MemoryKindFieldDeclaration-130]
	_ = x[MemoryKindTransactionDeclaration-131]
	_ = x[MemoryKindImportDeclaration-132]
	_ = x[MemoryKindVariableDeclaration-133]
	_ = x[MemoryKindSpecialFunctionDeclaration-134]
	_ = x[MemoryKindPragmaDeclaration-135]
	_ = x[MemoryKindAssignmentStatement-136]
	_ = x[MemoryKindBreakStatement-137]
	_ = x[MemoryKindContinueStatement-138]
	_ = x[MemoryKindEmitStatement-139]
	_ = x[MemoryKindExpressionStatement-140]
	_ = x[MemoryKindForStatement-141]
	_ = x[MemoryKindIfStatement-142]
	_ = x[MemoryKindReturnStatement-143]
	_ = x[MemoryKindSwapStatement-144]
	_ = x[MemoryKindSwitchStatement-145]
	_ = x[MemoryKindWhileStatement-146]
	_ = x[MemoryKindRemoveStatement-147]
	_ = x[MemoryKindBooleanExpression-148]
	_ = x[MemoryKindVoidExpression-149]
	_ = x[MemoryKindNilExpression-150]
	_ = x[MemoryKindStringExpression-151]
	_ = x[MemoryKindIntegerExpression-152]
	_ = x[MemoryKindFixedPointExpression-153]
	_ = x[MemoryKindArrayExpression-154]
	_ = x[MemoryKindDictionaryExpression-155]
	_ = x[MemoryKindIdentifierExpression-156]
	_ = x[MemoryKindInvocationExpression-157]
	_ = x[MemoryKindMemberExpression-158]
	_ = x[MemoryKindIndexExpression-159]
	_ = x[MemoryKindConditionalExpression-160]
	_ = x[MemoryKindUnaryExpression-161]
	_ = x[MemoryKindBinaryExpression-162]
	_ = x[MemoryKindFunctionExpression-163]
	_ = x[MemoryKindCastingExpression-164]
	_ = x[MemoryKindCreateExpression-165]
	_ = x[MemoryKindDestroyExpression-166]
	_ = x[MemoryKindReferenceExpression-167]
	_ = x[MemoryKindForceExpression-168]
	_ = x[MemoryKindPathExpression-169]
	_ = x[MemoryKindAttachExpression-170]
	_ = x[MemoryKindConstantSizedType-171]
	_ = x[MemoryKindDictionaryType-172]
	_ = x[MemoryKindFunctionType-173]
	_ = x[MemoryKindInstantiationType-174]
	_ = x[MemoryKindNominalType-175]
	_ = x[MemoryKindOptionalType-176]
	_ = x[MemoryKindReferenceType-177]
	_ = x[MemoryKindIntersectionType-178]
	_ = x[MemoryKindVariableSizedType-179]
	_ = x[MemoryKindPosition-180]
	_ = x[MemoryKindRange-181]
	_ = x[MemoryKindElaboration-182]
	_ = x[MemoryKindActivation-183]
	_ = x[MemoryKindActivationEntries-184]
	_ = x[MemoryKindVariableSizedSemaType-185]
	_ = x[MemoryKindConstantSizedSemaType-186]
	_ = x[MemoryKindDictionarySemaType-187]
	_ = x[MemoryKindOptionalSemaType-188]
	_ = x[MemoryKindIntersectionSemaType-189]
	_ = x[MemoryKindReferenceSemaType-190]
	_ = x[MemoryKindEntitlementSemaType-191]
	_ = x[MemoryKindEntitlementMapSemaType-192]
	_ = x[MemoryKindEntitlementRelationSemaType-193]
	_ = x[MemoryKindCapabilitySemaType-194]
	_ = x[MemoryKindInclusiveRangeSemaType-195]
	_ = x[MemoryKindOrderedMap-196]
	_ = x[MemoryKindOrderedMapEntryList-197]
	_ = x[MemoryKindOrderedMapEntry-198]
	_ = x[MemoryKindLast-199]
}

const _MemoryKind_name = "UnknownAddressValueStringValueCharacterValueNumberValueArrayValueBaseDictionaryValueBaseCompositeValueBaseSimpleCompositeValueBaseOptionalValueTypeValuePathValueCapabilityValueStorageReferenceValueEphemeralReferenceValueInterpretedFunctionValueHostFunctionValueBoundFunctionValueBigIntSimpleCompositeValuePublishedValueStorageCapabilityControllerValueAccountCapabilityControllerValueAtreeArrayDataSlabAtreeArrayMetaDataSlabAtreeArrayElementOverheadAtreeMapDataSlabAtreeMapMetaDataSlabAtreeMapElementOverheadAtreeMapPreAllocatedElementAtreeEncodedSlabPrimitiveStaticTypeCompositeStaticTypeInterfaceStaticTypeVariableSizedStaticTypeConstantSizedStaticTypeDictionaryStaticTypeInclusiveRangeStaticTypeOptionalStaticTypeIntersectionStaticTypeEntitlementSetStaticAccessEntitlementMapStaticAccessReferenceStaticTypeCapabilityStaticTypeFunctionStaticTypeCadenceVoidValueCadenceOptionalValueCadenceBoolValueCadenceStringValueCadenceCharacterValueCadenceAddressValueCadenceIntValueCadenceNumberValueCadenceArrayValueBaseCadenceArrayValueLengthCadenceDictionaryValueCadenceInclusiveRangeValueCadenceKeyValuePairCadenceStructValueBaseCadenceStructValueSizeCadenceResourceValueBaseCadenceAttachmentValueBaseCadenceResourceValueSizeCadenceAttachmentValueSizeCadenceEventValueBaseCadenceEventValueSizeCadenceContractValueBaseCadenceContractValueSizeCadenceEnumValueBaseCadenceEnumValueSizeCadencePathValueCadenceTypeValueCadenceCapabilityValueCadenceFunctionValueCadenceOptionalTypeCadenceVariableSizedArrayTypeCadenceConstantSizedArrayTypeCadenceDictionaryTypeCadenceInclusiveRangeTypeCadenceFieldCadenceParameterCadenceTypeParameterCadenceStructTypeCadenceResourceTypeCadenceAttachmentTypeCadenceEventTypeCadenceContractTypeCadenceStructInterfaceTypeCadenceResourceInterfaceTypeCadenceContractInterfaceTypeCadenceFunctionTypeCadenceEntitlementSetAccessCadenceEntitlementMapAccessCadenceReferenceTypeCadenceIntersectionTypeCadenceCapabilityTypeCadenceEnumTypeRawStringAddressLocationBytesVariableCompositeTypeInfoCompositeFieldInvocationStorageMapStorageKeyTypeTokenErrorTokenSpaceTokenProgramIdentifierArgumentBlockFunctionBlockParameterParameterListTypeParameterTypeParameterListTransferMembersTypeAnnotationDictionaryEntryFunctionDeclarationCompositeDeclarationAttachmentDeclarationInterfaceDeclarationEntitlementDeclarationEntitlementMappingElementEntitlementMappingDeclarationEnumCaseDeclarationFieldDeclarationTransactionDeclarationImportDeclarationVariableDeclarationSpecialFunctionDeclarationPragmaDeclarationAssignmentStatementBreakStatementContinueStatementEmitStatementExpressionStatementForStatementIfStatementReturnStatementSwapStatementSwitchStatementWhileStatementRemoveStatementBooleanExpressionVoidExpressionNilExpressionStringExpressionIntegerExpressionFixedPointExpressionArrayExpressionDictionaryExpressionIdentifierExpressionInvocationExpressionMemberExpressionIndexExpressionConditionalExpressionUnaryExpressionBinaryExpressionFunctionExpressionCastingExpressionCreateExpressionDestroyExpressionReferenceExpressionForceExpressionPathExpressionAttachExpressionConstantSizedTypeDictionaryTypeFunctionTypeInstantiationTypeNominalTypeOptionalTypeReferenceTypeIntersectionTypeVariableSizedTypePositionRangeElaborationActivationActivationEntriesVariableSizedSemaTypeConstantSizedSemaTypeDictionarySemaTypeOptionalSemaTypeIntersectionSemaTypeReferenceSemaTypeEntitlementSemaTypeEntitlementMapSemaTypeEntitlementRelationSemaTypeCapabilitySemaTypeInclusiveRangeSemaTypeOrderedMapOrderedMapEntryListOrderedMapEntryLast"

var _MemoryKind_index = [...]uint16{0, 7, 19, 30, 44, 55, 69, 88, 106, 130, 143, 152, 161, 176, 197, 220, 244, 261, 279, 285, 305, 319, 351, 383, 401, 423, 448, 464, 484, 507, 534, 550, 569, 588, 607, 630, 653, 673, 697, 715, 737, 763, 789, 808, 828, 846, 862, 882, 898, 916, 937, 956, 971, 989, 1010, 1033, 1055, 1081, 1100, 1122, 1144, 1168, 1194, 1218, 1244, 1265, 1286, 1310, 1334, 1354, 1374, 1390, 1406, 1428, 1448, 1467, 1496, 1525, 1546, 1571, 1583, 1599, 1619, 1636, 1655, 1676, 1692, 1711, 1737, 1765, 1793, 1812, 1839, 1866, 1886, 1909, 1930, 1945, 1954, 1969, 1974, 1982, 1999, 2013, 2023, 2033, 2043, 2052, 2062, 2072, 2079, 2089, 2097, 2102, 2115, 2124, 2137, 2150, 2167, 2175, 2182, 2196, 2211, 2230, 2250, 2271, 2291, 2313, 2338, 2367, 2386, 2402, 2424, 2441, 2460, 2486, 2503, 2522, 2536, 2553, 2566, 2585, 2597, 2608, 2623, 2636, 2651, 2665, 2680, 2697, 2711, 2724, 2740, 2757, 2777, 2792, 2812, 2832, 2852, 2868, 2883, 2904, 2919, 2935, 2953, 2970, 2986, 3003, 3022, 3037, 3051, 3067, 3084, 3098, 3110, 3127, 3138, 3150, 3163, 3179, 3196, 3204, 3209, 3220, 3230, 3247, 3268, 3289, 3307, 3323, 3343, 3360, 3379, 3401, 3428, 3446, 3468, 3478, 3497, 3512, 3516}

func (i MemoryKind) String() string {
	if i >= MemoryKind(len(_MemoryKind_index)-1) {
		return "MemoryKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _MemoryKind_name[_MemoryKind_index[i]:_MemoryKind_index[i+1]]
}
