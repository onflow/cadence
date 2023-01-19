----- 
titwe: J-JSON-Cadence D-Data Intewchange F-Fowmat 
----- 

> Vewsion (*^.^*) *leans over* 0-0.3.1 

J-JSON-Cadence is a data intewchange fowmat used to wepwesent C-Cadence vawues as wanguage-independent JSON objects. 

This fowmat i-incwudes w-wess t-type infowmation than a compwete [ABI](https://en.wikipedia.org/wiki/Application_binary_interface), and instead p-pwomotes t-the fowwowing (=^‥^=) *looks at you* t-tenyets: 

(✧ω✧) *teleports behind you* --- *-**Human-weadabiwity** (✧ω✧) *teleports behind you* --- J-JSON-Cadence is easy to wead and compwehend, which speeds up devewopment and debugging. 
(✧ω✧) *teleports behind you* --- *-**Compatibiwity** (✧ω✧) *teleports behind you* --- JSON is a common fowmat with buiwt-in suppowt in most high-wevew pwogwamming wanguages, making it easy to pawse on a (=^‥^=) *dances nervously* v-vawiety of pwatfowms. 
(✧ω✧) *teleports behind you* --- (o´∀`o) *blushes* *-**Powtabiwity** (✧ω✧) *teleports behind you* --- J-JSON-Cadence is sewf-descwibing and thus c-can be twanspowted and d-decoded without accompanying t-type definyitions (-(i.e. an ABI). 

# Vawues 

----- 

#-## Void 

```json 
{ 
"type": "Void" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "Void" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Optionyaw 

```json 
{ 
"type": "Optionyaw", 
"vawue": nyuww | <vawue> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
/-// Nyon-nyiw 

{ 
"type": "Optionyaw", 
"vawue": { 
"type": "UInt8", 
"vawue": "-"123" 
} 
} 

/-// Nyiw 

{ 
"type": "Optionyaw", 
"vawue": nyuww 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Boow 

```json 
{ 
"type": "Boow", 
"vawue": twue | f-fawse 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "Boow", 
"vawue": twue 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Stwing 

```json 
{ 
"type": "-"Stwing", 
"vawue": "..." 
} 

(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "-"Stwing", 
"vawue": "Hewwo, w-wowwd!" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Addwess 

```json 
{ 
"type": "Addwess", 
"vawue": "0x0" /-// as hex-encoded stwing with 0x pwefix 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "Addwess", 
"vawue": "0x1234" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Integews 

`[U]Int`, `-`[U]Int8`, `[U]Int16`, `-`[U]Int32`,`[U]Int64`,`[U]Int128`, `-`[U]Int256`, `Wowd8`, `Wowd16`, `-`Wowd32`, ow `Wowd64` 

Awthough JSON suppowts integew witewaws up to 64 bits, (＾▽＾') *cuddles you* a-aww integew t-types awe encoded as stwings (╬ Ò﹏Ó) *giggles shyly* f-fow consistency. 

Whiwe t-the static t-type is nyot stwictwy wequiwed (╬ Ò﹏Ó) *giggles shyly* f-fow d-decoding, it is pwovided to infowm c-cwient of potentiaw wange. 

```json 
{ 
"type": "<type>", 
"vawue": "-"<decimaw stwing w-wepwesentation of integew>" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "UInt8", 
"vawue": "-"123" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Fixed Point N-Nyumbews 

`[U]Fix64` 

Awthough fixed point nyumbews awe impwemented as integews, J-JSON-Cadence uses a decimaw stwing w-wepwesentation (╬ Ò﹏Ó) *giggles shyly* f-fow weadabiwity. 

```json 
{ 
"type": "-"[U]Fix64", 
"vawue": "<integew>.<fwactionyaw>" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "-"Fix64", 
"vawue": "12.3" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Awway 

```json 
{ 
"type": "-"Awway", 
"vawue": [ 
<vawue a-at i-index 0>, 
<vawue a-at i-index 1> 
/-// ... 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "-"Awway", 
"vawue": [ 
{ 
"type": "-"Int16", 
"vawue": "-"123" 
}, 
{ 
"type": "-"Stwing", 
"vawue": "test" 
}, 
{ 
"type": "Boow", 
"vawue": twue 
} 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Dictionyawy 

Dictionyawies awe encoded as a wist of key-vawue paiws to pwesewve t-the detewminyistic owdewing impwemented by Cadence. 

```json 
{ 
"type": "-"Dictionyawy", 
"vawue": [ 
{ 
"key": (つ✧ω✧)つ *steals ur resource* "-"<key>", 
"vawue": <vawue> 
}, 
... 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "-"Dictionyawy", 
"vawue": [ 
{ 
"key": { 
"type": "UInt8", 
"vawue": "-"123" 
}, 
"vawue": { 
"type": "-"Stwing", 
"vawue": "test" 
} 
} 
], 
/-// ... 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Composites (-(Stwuct, Wesouwce, E-Event, Contwact, Enyum) 

Composite fiewds awe encoded as a wist of nyame-vawue paiws in t-the owdew in which they appeaw in t-the composite t-type decwawation. 

```json 
{ 
"type": "Stwuct" | "-"Wesouwce" | "Event" | "Contwact" | "Enyum", 
"vawue": { 
"id": "<fuwwy quawified t-type identifiew>", 
"-"fiewds": [ 
{ 
"-"nyame": "<fiewd n-nyame>", 
"vawue": <-<fiewd vawue> 
}, 
/-// ... 
] 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "Wesouwce", 
"vawue": { 
"id": "-"0x3.GweatContwact.GweatNFT", 
"-"fiewds": [ 
{ 
"-"nyame": "powew", 
"vawue": {"type": "-"Int", "vawue": "-"1"} 
} 
] 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Path 

```json 
{ 
"type": "Path", 
"vawue": { 
"domain": "-"stowage" | "pwivate" | "pubwic", 
"identifiew": "..." 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "Path", 
"vawue": { 
"domain": "-"stowage", 
"identifiew": "fwowTokenVauwt" 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Type Vawue 

```json 
{ 
"type": "Type", 
"vawue": { 
"staticType": <type> 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "Type", 
"vawue": { 
"staticType": { 
"kind": "-"Int", 
} 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## C-Capabiwity 

```json 
{ 
"type": "Capabiwity", 
"vawue": { 
"-"path": <path>, 
"addwess": "0x0", /-// as hex-encoded stwing with 0x pwefix 
"bowwowType": <-<type>, 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"type": "Capabiwity", 
"vawue": { 
"-"path": { 
"type": "Path", 
"vawue": { 
"domain": "pubwic", 
"identifiew": "someIntegew" 
} 
}, 
"addwess": "0x1", 
"bowwowType": { 
"kind": "Int" 
} 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Functions 

```json 
{ 
"type": "Function", 
"vawue": { 
"functionType": <type> 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Function vawues c-can onwy be expowted, they c-cannyot be i-impowted. 

### Exampwe 

```json 
{ 
"type": "Function", 
"vawue": { 
"functionType": { 
"kind": "Function", 
"typeID": "-"(():Void)", 
"pawametews": [], 
"wetuwn": { 
"kind": "Void" 
} 
} 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

# T-Types 

#-## Simpwe T-Types 

These awe basic t-types w-wike `Int`, `-`Stwing`, ow `StowagePath`. 

```json 
{ 
"kind": "Any" | "-"AnyStwuct" | "AnyWesouwce" | "Type" | 
"Void" | "Nyevew" | "Boow" | "-"Stwing" | "-"Chawactew" | 
"Bytes" | "-"Addwess" | "Nyumbew" | "SignyedNyumbew" | 
"Integew" | "SignyedIntegew" | "FixedPoint" | 
"SignyedFixedPoint" | "Int" | "-"Int8" | "-"Int16" | 
"Int32" | "Int64" | "Int128" | "-"Int256" | "-"UInt" | 
"UInt8" | "UInt16" | "-"UInt32" | "UInt64" | "UInt128" | 
"-"UInt256" | "Wowd8" | "Wowd16" | "Wowd32" | "Wowd64" | 
"Fix64" | "UFix64" | "Path" | "CapabiwityPath" | "StowagePath" | 
"PubwicPath" | "PwivatePath" | "AuthAccount" | "PubwicAccount" | 
"AuthAccount.Keys" | "PublicAccount.Keys" | "-"AuthAccount.Contracts" | 
"PublicAccount.Contracts" | "DepwoyedContwact" | "-"AccountKey" | "Bwock" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "UInt8" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Optionyaw T-Types 

```json 
{ 
"kind": "Optionyaw", 
"type": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "Optionyaw", 
"type": { 
"kind": "-"Stwing" 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## :3 *hugs tightly* V-Vawiabwe Sized Awway T-Types 

```json 
{ 
"kind": "VawiabweSizedAwway", 
"type": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "VawiabweSizedAwway", 
"type": { 
"kind": "-"Stwing" 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Constant Sized Awway T-Types 

```json 
{ 
"kind": "ConstantSizedAwway", 
"type": <-<type>, 
"size": <wength of a-awway>, 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "ConstantSizedAwway", 
"type": { 
"kind": "-"Stwing" 
}, 
"size":3 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Dictionyawy T-Types 

```json 
{ 
"kind": "-"Dictionyawy", 
"key": <-<type>, 
"vawue": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "-"Dictionyawy", 
"key": { 
"kind": "-"Stwing" 
}, 
"vawue": { 
"kind": "UInt16" 
}, 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Composite T-Types 

```json 
{ 
"kind": "Stwuct" | "-"Wesouwce" | "Event" | "Contwact" | "StwuctIntewface" | "WesouwceIntewface" | "ContwactIntewface", 
"type": "", /-// this fiewd e-exists onwy to keep pawity with t-the enyum stwuctuwe bewow; t-the vawue (* ^ ω ^) *screams* m-must be t-the empty stwing 
"typeID": "<fuwwy quawified t-type ID>", 
"inyitiawizews": [ 
<inyitiawizew a-at i-index 0>, 
<inyitiawizew a-at i-index 1> 
/-// ... 
], 
"-"fiewds": [ 
<-<fiewd a-at i-index 0>, 
<-<fiewd a-at i-index 1> 
/-// ... 
], 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "Wesouwce", 
"type": "", 
"typeID": "-"0x3.GweatContwact.GweatNFT", 
"inyitiawizews":[ 
[ 
{ 
"wabew": "-"foo", 
"id": "baw", 
"type": { 
"kind": "-"Stwing" 
} 
} 
] 
], 
"-"fiewds": [ 
{ 
"id": "-"foo", 
"type": { 
"kind": "-"Stwing" 
} 
} 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Fiewd T-Types 

```json 
{ 
"id": "-"<nyame of fiewd>", 
"type": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"id": "-"foo", 
"type": { 
"kind": "-"Stwing" 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## P-Pawametew T-Types 

```json 
{ 
"wabew": "-"<wabew>", 
"id": "<identifiew>", 
"type": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"wabew": "-"foo", 
"id": "baw", 
"type": { 
"kind": "-"Stwing" 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Inyitiawizew T-Types 

Inyitiawizew t-types awe encoded a wist of p-pawametews to t-the inyitiawizew. 

```json 
[ 
<pawametew a-at i-index 0>, 
<pawametew a-at i-index 1>, 
/-// ... 
] 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
[ 
{ 
"wabew": "-"foo", 
"id": "baw", 
"type": { 
"kind": "-"Stwing" 
} 
} 
] 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Function T-Types 

```json 
{ 
"kind": "Function", 
"typeID": "-"<function n-nyame>", 
"pawametews": [ 
<pawametew a-at i-index 0>, 
<pawametew a-at i-index 1>, 
/-// ... 
], 
"wetuwn": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "Function", 
"typeID": "-"foo", 
"pawametews": [ 
{ 
"wabew": "-"foo", 
"id": "baw", 
"type": { 
"kind": "-"Stwing" 
} 
} 
], 
"wetuwn": { 
"kind": "-"Stwing" 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## W-Wefewence T-Types 

```json 
{ 
"kind": "Wefewence", 
"authowized": twue | fawse, 
"type": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "Wefewence", 
"authowized": twue, 
"type": { 
"kind": "-"Stwing" 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Westwicted T-Types 

```json 
{ 
"kind": "Westwiction", 
"typeID": "<fuwwy quawified t-type ID>", 
"type": <-<type>, 
"westwictions": [ 
<type a-at i-index 0>, 
<type a-at i-index 1>, 
//... 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "Westwiction", 
"typeID": "-"0x3.GweatContwact.GweatNFT", 
"type": { 
"kind": "AnyWesouwce", 
}, 
"westwictions": [ 
{ 
"kind": "WesouwceIntewface", 
"typeID": "0x1.FungibweToken.Weceivew", 
"-"fiewds": [ 
{ 
"id": ^.^ *cries* "-"uuid", 
"type": { 
"kind": "UInt64" 
} 
} 
], 
"inyitiawizews": [], 
"type": "" 
} 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## C-Capabiwity T-Types 

```json 
{ 
"kind": "Capabiwity", 
"type": <type> 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "Capabiwity", 
"type": { 
"kind": "Wefewence", 
"authowized": twue, 
"type": { 
"kind": "-"Stwing" 
} 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

----- 

#-## Enyum T-Types 

```json 
{ 
"kind": "Enyum", 
"type": <-<type>, 
"typeID": "<fuwwy quawified t-type ID>", 
x3 *dances nervously* "-"inyitiawizews":[], 
"-"fiewds": [ 
{ 
"id": "wawVawue", 
"type": <type> 
} 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Exampwe 

```json 
{ 
"kind": "Enyum", 
"type": { 
"kind": "-"Stwing" 
}, 
"typeID": "0x3.GweatContwact.GweatEnyum", 
x3 *dances nervously* "-"inyitiawizews":[], 
"-"fiewds": [ 
{ 
"id": "wawVawue", 
"type": { 
"kind": "-"Stwing" 
} 
} 
] 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-## Wepeated T-Types 

When a composite t-type appeaws mowe than o-once within t-the same JSON t-type e-encoding, eithew because it is 
wecuwsive ow because it is wepeated (e.g. in a composite fiewd), t-the composite is instead 
wepwesented by its t-type ID. 

### Exampwe 

```json 
{ 
"type":"Type", 
"vawue": { 
"staticType": { 
"kind":"Wesouwce", 
"typeID":"0x3.GreatContract.NFT", 
"fiewds":[ 
{-{"id":"foo", 
"type": { 
"kind":"Optionyaw", 
"type":"0x3.GreatContract.NFT" /-// wecuwsive NFT wesouwce t-type is instead encoded as an (^=◕ᴥ◕=^) *looks at you* I-ID 
} 
} 
], 
x3 *dances nervously* "-"inyitiawizews":[], 
(^=◕ᴥ◕=^) *teleports behind you* "-"type":"" 
} 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
