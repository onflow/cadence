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
	_ = x[MemoryKindStorageCapabilityValue-12]
	_ = x[MemoryKindPathLinkValue-13]
	_ = x[MemoryKindAccountLinkValue-14]
	_ = x[MemoryKindStorageReferenceValue-15]
	_ = x[MemoryKindAccountReferenceValue-16]
	_ = x[MemoryKindEphemeralReferenceValue-17]
	_ = x[MemoryKindInterpretedFunctionValue-18]
	_ = x[MemoryKindHostFunctionValue-19]
	_ = x[MemoryKindBoundFunctionValue-20]
	_ = x[MemoryKindBigInt-21]
	_ = x[MemoryKindSimpleCompositeValue-22]
	_ = x[MemoryKindPublishedValue-23]
	_ = x[MemoryKindAtreeArrayDataSlab-24]
	_ = x[MemoryKindAtreeArrayMetaDataSlab-25]
	_ = x[MemoryKindAtreeArrayElementOverhead-26]
	_ = x[MemoryKindAtreeMapDataSlab-27]
	_ = x[MemoryKindAtreeMapMetaDataSlab-28]
	_ = x[MemoryKindAtreeMapElementOverhead-29]
	_ = x[MemoryKindAtreeMapPreAllocatedElement-30]
	_ = x[MemoryKindAtreeEncodedSlab-31]
	_ = x[MemoryKindPrimitiveStaticType-32]
	_ = x[MemoryKindCompositeStaticType-33]
	_ = x[MemoryKindInterfaceStaticType-34]
	_ = x[MemoryKindVariableSizedStaticType-35]
	_ = x[MemoryKindConstantSizedStaticType-36]
	_ = x[MemoryKindDictionaryStaticType-37]
	_ = x[MemoryKindOptionalStaticType-38]
	_ = x[MemoryKindRestrictedStaticType-39]
	_ = x[MemoryKindReferenceStaticType-40]
	_ = x[MemoryKindCapabilityStaticType-41]
	_ = x[MemoryKindFunctionStaticType-42]
	_ = x[MemoryKindCadenceVoidValue-43]
	_ = x[MemoryKindCadenceOptionalValue-44]
	_ = x[MemoryKindCadenceBoolValue-45]
	_ = x[MemoryKindCadenceStringValue-46]
	_ = x[MemoryKindCadenceCharacterValue-47]
	_ = x[MemoryKindCadenceAddressValue-48]
	_ = x[MemoryKindCadenceIntValue-49]
	_ = x[MemoryKindCadenceNumberValue-50]
	_ = x[MemoryKindCadenceArrayValueBase-51]
	_ = x[MemoryKindCadenceArrayValueLength-52]
	_ = x[MemoryKindCadenceDictionaryValue-53]
	_ = x[MemoryKindCadenceKeyValuePair-54]
	_ = x[MemoryKindCadenceStructValueBase-55]
	_ = x[MemoryKindCadenceStructValueSize-56]
	_ = x[MemoryKindCadenceResourceValueBase-57]
	_ = x[MemoryKindCadenceAttachmentValueBase-58]
	_ = x[MemoryKindCadenceResourceValueSize-59]
	_ = x[MemoryKindCadenceAttachmentValueSize-60]
	_ = x[MemoryKindCadenceEventValueBase-61]
	_ = x[MemoryKindCadenceEventValueSize-62]
	_ = x[MemoryKindCadenceContractValueBase-63]
	_ = x[MemoryKindCadenceContractValueSize-64]
	_ = x[MemoryKindCadenceEnumValueBase-65]
	_ = x[MemoryKindCadenceEnumValueSize-66]
	_ = x[MemoryKindCadencePathLinkValue-67]
	_ = x[MemoryKindCadenceAccountLinkValue-68]
	_ = x[MemoryKindCadencePathValue-69]
	_ = x[MemoryKindCadenceTypeValue-70]
	_ = x[MemoryKindCadenceStorageCapabilityValue-71]
	_ = x[MemoryKindCadenceFunctionValue-72]
	_ = x[MemoryKindCadenceOptionalType-73]
	_ = x[MemoryKindCadenceVariableSizedArrayType-74]
	_ = x[MemoryKindCadenceConstantSizedArrayType-75]
	_ = x[MemoryKindCadenceDictionaryType-76]
	_ = x[MemoryKindCadenceField-77]
	_ = x[MemoryKindCadenceParameter-78]
	_ = x[MemoryKindCadenceStructType-79]
	_ = x[MemoryKindCadenceResourceType-80]
	_ = x[MemoryKindCadenceAttachmentType-81]
	_ = x[MemoryKindCadenceEventType-82]
	_ = x[MemoryKindCadenceContractType-83]
	_ = x[MemoryKindCadenceStructInterfaceType-84]
	_ = x[MemoryKindCadenceResourceInterfaceType-85]
	_ = x[MemoryKindCadenceContractInterfaceType-86]
	_ = x[MemoryKindCadenceFunctionType-87]
	_ = x[MemoryKindCadenceReferenceType-88]
	_ = x[MemoryKindCadenceRestrictedType-89]
	_ = x[MemoryKindCadenceCapabilityType-90]
	_ = x[MemoryKindCadenceEnumType-91]
	_ = x[MemoryKindRawString-92]
	_ = x[MemoryKindAddressLocation-93]
	_ = x[MemoryKindBytes-94]
	_ = x[MemoryKindVariable-95]
	_ = x[MemoryKindCompositeTypeInfo-96]
	_ = x[MemoryKindCompositeField-97]
	_ = x[MemoryKindInvocation-98]
	_ = x[MemoryKindStorageMap-99]
	_ = x[MemoryKindStorageKey-100]
	_ = x[MemoryKindTypeToken-101]
	_ = x[MemoryKindErrorToken-102]
	_ = x[MemoryKindSpaceToken-103]
	_ = x[MemoryKindProgram-104]
	_ = x[MemoryKindIdentifier-105]
	_ = x[MemoryKindArgument-106]
	_ = x[MemoryKindBlock-107]
	_ = x[MemoryKindFunctionBlock-108]
	_ = x[MemoryKindParameter-109]
	_ = x[MemoryKindParameterList-110]
	_ = x[MemoryKindTypeParameter-111]
	_ = x[MemoryKindTypeParameterList-112]
	_ = x[MemoryKindTransfer-113]
	_ = x[MemoryKindMembers-114]
	_ = x[MemoryKindTypeAnnotation-115]
	_ = x[MemoryKindDictionaryEntry-116]
	_ = x[MemoryKindFunctionDeclaration-117]
	_ = x[MemoryKindCompositeDeclaration-118]
	_ = x[MemoryKindAttachmentDeclaration-119]
	_ = x[MemoryKindInterfaceDeclaration-120]
	_ = x[MemoryKindEnumCaseDeclaration-121]
	_ = x[MemoryKindFieldDeclaration-122]
	_ = x[MemoryKindTransactionDeclaration-123]
	_ = x[MemoryKindImportDeclaration-124]
	_ = x[MemoryKindVariableDeclaration-125]
	_ = x[MemoryKindSpecialFunctionDeclaration-126]
	_ = x[MemoryKindPragmaDeclaration-127]
	_ = x[MemoryKindAssignmentStatement-128]
	_ = x[MemoryKindBreakStatement-129]
	_ = x[MemoryKindContinueStatement-130]
	_ = x[MemoryKindEmitStatement-131]
	_ = x[MemoryKindExpressionStatement-132]
	_ = x[MemoryKindForStatement-133]
	_ = x[MemoryKindIfStatement-134]
	_ = x[MemoryKindReturnStatement-135]
	_ = x[MemoryKindSwapStatement-136]
	_ = x[MemoryKindSwitchStatement-137]
	_ = x[MemoryKindWhileStatement-138]
	_ = x[MemoryKindRemoveStatement-139]
	_ = x[MemoryKindBooleanExpression-140]
	_ = x[MemoryKindVoidExpression-141]
	_ = x[MemoryKindNilExpression-142]
	_ = x[MemoryKindStringExpression-143]
	_ = x[MemoryKindIntegerExpression-144]
	_ = x[MemoryKindFixedPointExpression-145]
	_ = x[MemoryKindArrayExpression-146]
	_ = x[MemoryKindDictionaryExpression-147]
	_ = x[MemoryKindIdentifierExpression-148]
	_ = x[MemoryKindInvocationExpression-149]
	_ = x[MemoryKindMemberExpression-150]
	_ = x[MemoryKindIndexExpression-151]
	_ = x[MemoryKindConditionalExpression-152]
	_ = x[MemoryKindUnaryExpression-153]
	_ = x[MemoryKindBinaryExpression-154]
	_ = x[MemoryKindFunctionExpression-155]
	_ = x[MemoryKindCastingExpression-156]
	_ = x[MemoryKindCreateExpression-157]
	_ = x[MemoryKindDestroyExpression-158]
	_ = x[MemoryKindReferenceExpression-159]
	_ = x[MemoryKindForceExpression-160]
	_ = x[MemoryKindPathExpression-161]
	_ = x[MemoryKindAttachExpression-162]
	_ = x[MemoryKindConstantSizedType-163]
	_ = x[MemoryKindDictionaryType-164]
	_ = x[MemoryKindFunctionType-165]
	_ = x[MemoryKindInstantiationType-166]
	_ = x[MemoryKindNominalType-167]
	_ = x[MemoryKindOptionalType-168]
	_ = x[MemoryKindReferenceType-169]
	_ = x[MemoryKindRestrictedType-170]
	_ = x[MemoryKindVariableSizedType-171]
	_ = x[MemoryKindPosition-172]
	_ = x[MemoryKindRange-173]
	_ = x[MemoryKindElaboration-174]
	_ = x[MemoryKindActivation-175]
	_ = x[MemoryKindActivationEntries-176]
	_ = x[MemoryKindVariableSizedSemaType-177]
	_ = x[MemoryKindConstantSizedSemaType-178]
	_ = x[MemoryKindDictionarySemaType-179]
	_ = x[MemoryKindOptionalSemaType-180]
	_ = x[MemoryKindRestrictedSemaType-181]
	_ = x[MemoryKindReferenceSemaType-182]
	_ = x[MemoryKindCapabilitySemaType-183]
	_ = x[MemoryKindOrderedMap-184]
	_ = x[MemoryKindOrderedMapEntryList-185]
	_ = x[MemoryKindOrderedMapEntry-186]
	_ = x[MemoryKindLast-187]
}

const _MemoryKind_name = "UnknownAddressValueStringValueCharacterValueNumberValueArrayValueBaseDictionaryValueBaseCompositeValueBaseSimpleCompositeValueBaseOptionalValueTypeValuePathValueStorageCapabilityValuePathLinkValueAccountLinkValueStorageReferenceValueAccountReferenceValueEphemeralReferenceValueInterpretedFunctionValueHostFunctionValueBoundFunctionValueBigIntSimpleCompositeValuePublishedValueAtreeArrayDataSlabAtreeArrayMetaDataSlabAtreeArrayElementOverheadAtreeMapDataSlabAtreeMapMetaDataSlabAtreeMapElementOverheadAtreeMapPreAllocatedElementAtreeEncodedSlabPrimitiveStaticTypeCompositeStaticTypeInterfaceStaticTypeVariableSizedStaticTypeConstantSizedStaticTypeDictionaryStaticTypeOptionalStaticTypeRestrictedStaticTypeReferenceStaticTypeCapabilityStaticTypeFunctionStaticTypeCadenceVoidValueCadenceOptionalValueCadenceBoolValueCadenceStringValueCadenceCharacterValueCadenceAddressValueCadenceIntValueCadenceNumberValueCadenceArrayValueBaseCadenceArrayValueLengthCadenceDictionaryValueCadenceKeyValuePairCadenceStructValueBaseCadenceStructValueSizeCadenceResourceValueBaseCadenceAttachmentValueBaseCadenceResourceValueSizeCadenceAttachmentValueSizeCadenceEventValueBaseCadenceEventValueSizeCadenceContractValueBaseCadenceContractValueSizeCadenceEnumValueBaseCadenceEnumValueSizeCadencePathLinkValueCadenceAccountLinkValueCadencePathValueCadenceTypeValueCadenceStorageCapabilityValueCadenceFunctionValueCadenceOptionalTypeCadenceVariableSizedArrayTypeCadenceConstantSizedArrayTypeCadenceDictionaryTypeCadenceFieldCadenceParameterCadenceStructTypeCadenceResourceTypeCadenceAttachmentTypeCadenceEventTypeCadenceContractTypeCadenceStructInterfaceTypeCadenceResourceInterfaceTypeCadenceContractInterfaceTypeCadenceFunctionTypeCadenceReferenceTypeCadenceRestrictedTypeCadenceCapabilityTypeCadenceEnumTypeRawStringAddressLocationBytesVariableCompositeTypeInfoCompositeFieldInvocationStorageMapStorageKeyTypeTokenErrorTokenSpaceTokenProgramIdentifierArgumentBlockFunctionBlockParameterParameterListTypeParameterTypeParameterListTransferMembersTypeAnnotationDictionaryEntryFunctionDeclarationCompositeDeclarationAttachmentDeclarationInterfaceDeclarationEnumCaseDeclarationFieldDeclarationTransactionDeclarationImportDeclarationVariableDeclarationSpecialFunctionDeclarationPragmaDeclarationAssignmentStatementBreakStatementContinueStatementEmitStatementExpressionStatementForStatementIfStatementReturnStatementSwapStatementSwitchStatementWhileStatementRemoveStatementBooleanExpressionVoidExpressionNilExpressionStringExpressionIntegerExpressionFixedPointExpressionArrayExpressionDictionaryExpressionIdentifierExpressionInvocationExpressionMemberExpressionIndexExpressionConditionalExpressionUnaryExpressionBinaryExpressionFunctionExpressionCastingExpressionCreateExpressionDestroyExpressionReferenceExpressionForceExpressionPathExpressionAttachExpressionConstantSizedTypeDictionaryTypeFunctionTypeInstantiationTypeNominalTypeOptionalTypeReferenceTypeRestrictedTypeVariableSizedTypePositionRangeElaborationActivationActivationEntriesVariableSizedSemaTypeConstantSizedSemaTypeDictionarySemaTypeOptionalSemaTypeRestrictedSemaTypeReferenceSemaTypeCapabilitySemaTypeOrderedMapOrderedMapEntryListOrderedMapEntryLast"

var _MemoryKind_index = [...]uint16{0, 7, 19, 30, 44, 55, 69, 88, 106, 130, 143, 152, 161, 183, 196, 212, 233, 254, 277, 301, 318, 336, 342, 362, 376, 394, 416, 441, 457, 477, 500, 527, 543, 562, 581, 600, 623, 646, 666, 684, 704, 723, 743, 761, 777, 797, 813, 831, 852, 871, 886, 904, 925, 948, 970, 989, 1011, 1033, 1057, 1083, 1107, 1133, 1154, 1175, 1199, 1223, 1243, 1263, 1283, 1306, 1322, 1338, 1367, 1387, 1406, 1435, 1464, 1485, 1497, 1513, 1530, 1549, 1570, 1586, 1605, 1631, 1659, 1687, 1706, 1726, 1747, 1768, 1783, 1792, 1807, 1812, 1820, 1837, 1851, 1861, 1871, 1881, 1890, 1900, 1910, 1917, 1927, 1935, 1940, 1953, 1962, 1975, 1988, 2005, 2013, 2020, 2034, 2049, 2068, 2088, 2109, 2129, 2148, 2164, 2186, 2203, 2222, 2248, 2265, 2284, 2298, 2315, 2328, 2347, 2359, 2370, 2385, 2398, 2413, 2427, 2442, 2459, 2473, 2486, 2502, 2519, 2539, 2554, 2574, 2594, 2614, 2630, 2645, 2666, 2681, 2697, 2715, 2732, 2748, 2765, 2784, 2799, 2813, 2829, 2846, 2860, 2872, 2889, 2900, 2912, 2925, 2939, 2956, 2964, 2969, 2980, 2990, 3007, 3028, 3049, 3067, 3083, 3101, 3118, 3136, 3146, 3165, 3180, 3184}

func (i MemoryKind) String() string {
	if i >= MemoryKind(len(_MemoryKind_index)-1) {
		return "MemoryKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _MemoryKind_name[_MemoryKind_index[i]:_MemoryKind_index[i+1]]
}
