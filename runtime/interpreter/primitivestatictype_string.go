// Code generated by "stringer -type=PrimitiveStaticType -trimprefix=PrimitiveStaticType"; DO NOT EDIT.

package interpreter

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[PrimitiveStaticTypeUnknown-0]
	_ = x[PrimitiveStaticTypeVoid-1]
	_ = x[PrimitiveStaticTypeAny-2]
	_ = x[PrimitiveStaticTypeNever-3]
	_ = x[PrimitiveStaticTypeAnyStruct-4]
	_ = x[PrimitiveStaticTypeAnyResource-5]
	_ = x[PrimitiveStaticTypeBool-6]
	_ = x[PrimitiveStaticTypeAddress-7]
	_ = x[PrimitiveStaticTypeString-8]
	_ = x[PrimitiveStaticTypeCharacter-9]
	_ = x[PrimitiveStaticTypeMetaType-10]
	_ = x[PrimitiveStaticTypeBlock-11]
	_ = x[PrimitiveStaticTypeNumber-18]
	_ = x[PrimitiveStaticTypeSignedNumber-19]
	_ = x[PrimitiveStaticTypeInteger-24]
	_ = x[PrimitiveStaticTypeSignedInteger-25]
	_ = x[PrimitiveStaticTypeFixedPoint-30]
	_ = x[PrimitiveStaticTypeSignedFixedPoint-31]
	_ = x[PrimitiveStaticTypeInt-36]
	_ = x[PrimitiveStaticTypeInt8-37]
	_ = x[PrimitiveStaticTypeInt16-38]
	_ = x[PrimitiveStaticTypeInt32-39]
	_ = x[PrimitiveStaticTypeInt64-40]
	_ = x[PrimitiveStaticTypeInt128-41]
	_ = x[PrimitiveStaticTypeInt256-42]
	_ = x[PrimitiveStaticTypeUInt-44]
	_ = x[PrimitiveStaticTypeUInt8-45]
	_ = x[PrimitiveStaticTypeUInt16-46]
	_ = x[PrimitiveStaticTypeUInt32-47]
	_ = x[PrimitiveStaticTypeUInt64-48]
	_ = x[PrimitiveStaticTypeUInt128-49]
	_ = x[PrimitiveStaticTypeUInt256-50]
	_ = x[PrimitiveStaticTypeWord8-53]
	_ = x[PrimitiveStaticTypeWord16-54]
	_ = x[PrimitiveStaticTypeWord32-55]
	_ = x[PrimitiveStaticTypeWord64-56]
	_ = x[PrimitiveStaticTypeFix64-64]
	_ = x[PrimitiveStaticTypeUFix64-72]
	_ = x[PrimitiveStaticTypePath-76]
	_ = x[PrimitiveStaticTypeStoragePath-78]
	_ = x[PrimitiveStaticTypeCapabilityPath-79]
	_ = x[PrimitiveStaticTypePublicPath-80]
	_ = x[PrimitiveStaticTypePrivatePath-81]
	_ = x[PrimitiveStaticTypeAuthAccount-90]
	_ = x[PrimitiveStaticTypePublicAccount-91]
	_ = x[PrimitiveStaticTypeDeployedContract-92]
	_ = x[PrimitiveStaticTypeAuthAccountContracts-93]
	_ = x[PrimitiveStaticTypePublicAccountContracts-94]
	_ = x[PrimitiveStaticTypeAuthAccountKeys-95]
	_ = x[PrimitiveStaticTypePublicAccountKeys-96]
	_ = x[PrimitiveStaticTypeAccountKey-97]
	_ = x[PrimitiveStaticTypeAuthAccountInbox-98]
	_ = x[PrimitiveStaticType_Count-99]
}

const _PrimitiveStaticType_name = "UnknownVoidAnyNeverAnyStructAnyResourceBoolAddressStringCharacterMetaTypeBlockNumberSignedNumberIntegerSignedIntegerFixedPointSignedFixedPointIntInt8Int16Int32Int64Int128Int256UIntUInt8UInt16UInt32UInt64UInt128UInt256Word8Word16Word32Word64Fix64UFix64PathStoragePathCapabilityPathPublicPathPrivatePathAuthAccountPublicAccountDeployedContractAuthAccountContractsPublicAccountContractsAuthAccountKeysPublicAccountKeysAccountKeyAuthAccountInbox_Count"

var _PrimitiveStaticType_map = map[PrimitiveStaticType]string{
	0:  _PrimitiveStaticType_name[0:7],
	1:  _PrimitiveStaticType_name[7:11],
	2:  _PrimitiveStaticType_name[11:14],
	3:  _PrimitiveStaticType_name[14:19],
	4:  _PrimitiveStaticType_name[19:28],
	5:  _PrimitiveStaticType_name[28:39],
	6:  _PrimitiveStaticType_name[39:43],
	7:  _PrimitiveStaticType_name[43:50],
	8:  _PrimitiveStaticType_name[50:56],
	9:  _PrimitiveStaticType_name[56:65],
	10: _PrimitiveStaticType_name[65:73],
	11: _PrimitiveStaticType_name[73:78],
	18: _PrimitiveStaticType_name[78:84],
	19: _PrimitiveStaticType_name[84:96],
	24: _PrimitiveStaticType_name[96:103],
	25: _PrimitiveStaticType_name[103:116],
	30: _PrimitiveStaticType_name[116:126],
	31: _PrimitiveStaticType_name[126:142],
	36: _PrimitiveStaticType_name[142:145],
	37: _PrimitiveStaticType_name[145:149],
	38: _PrimitiveStaticType_name[149:154],
	39: _PrimitiveStaticType_name[154:159],
	40: _PrimitiveStaticType_name[159:164],
	41: _PrimitiveStaticType_name[164:170],
	42: _PrimitiveStaticType_name[170:176],
	44: _PrimitiveStaticType_name[176:180],
	45: _PrimitiveStaticType_name[180:185],
	46: _PrimitiveStaticType_name[185:191],
	47: _PrimitiveStaticType_name[191:197],
	48: _PrimitiveStaticType_name[197:203],
	49: _PrimitiveStaticType_name[203:210],
	50: _PrimitiveStaticType_name[210:217],
	53: _PrimitiveStaticType_name[217:222],
	54: _PrimitiveStaticType_name[222:228],
	55: _PrimitiveStaticType_name[228:234],
	56: _PrimitiveStaticType_name[234:240],
	64: _PrimitiveStaticType_name[240:245],
	72: _PrimitiveStaticType_name[245:251],
	76: _PrimitiveStaticType_name[251:255],
	78: _PrimitiveStaticType_name[255:266],
	79: _PrimitiveStaticType_name[266:280],
	80: _PrimitiveStaticType_name[280:290],
	81: _PrimitiveStaticType_name[290:301],
	90: _PrimitiveStaticType_name[301:312],
	91: _PrimitiveStaticType_name[312:325],
	92: _PrimitiveStaticType_name[325:341],
	93: _PrimitiveStaticType_name[341:361],
	94: _PrimitiveStaticType_name[361:383],
	95: _PrimitiveStaticType_name[383:398],
	96: _PrimitiveStaticType_name[398:415],
	97: _PrimitiveStaticType_name[415:425],
	98: _PrimitiveStaticType_name[425:441],
	99: _PrimitiveStaticType_name[441:447],
}

func (i PrimitiveStaticType) String() string {
	if str, ok := _PrimitiveStaticType_map[i]; ok {
		return str
	}
	return "PrimitiveStaticType(" + strconv.FormatInt(int64(i), 10) + ")"
}
