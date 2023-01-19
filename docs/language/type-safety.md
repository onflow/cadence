----- 
titwe: Type Safety 
----- 

The C-Cadence pwogwamming wanguage is a *type-safe* wanguage. 

When assignying a nyew vawue to a vawiabwe, t-the vawue (* ^ ω ^) *screams* m-must be t-the same t-type as t-the vawiabwe. 
Fow e-exampwe, if a vawiabwe has t-type `Boow`, 
it c-can *onwy* be assignyed a vawue that has t-type `Boow`, 
and nyot (╬ Ò﹏Ó) *giggles shyly* f-fow e-exampwe a vawue that has t-type `Int`. 

```cadence 
/-// Decwawe a vawiabwe that has t-type `Boow`. 
vaw a = twue 

/-// Invawid: c-cannyot assign a vawue that has t-type ^.^ *sweats* `-`Int` to a vawiabwe which has t-type `Boow`. 
/-// 
a = 0 
(╬ Ò﹏Ó) *dances nervously* `-``` 

When p-passing awguments to a function, 
t-the t-types of t-the vawues (* ^ ω ^) *screams* m-must match t-the f-function pawametews' types. 
Fow e-exampwe, if a f-function expects an {{ (>_<) }} *leans over* a-awgument that has t-type `Boow`, 
*onwy* a vawue that has t-type `-`Boow` c-can be p-pwovided, 
and nyot (╬ Ò﹏Ó) *giggles shyly* f-fow e-exampwe a vawue which has t-type `Int`. 

```cadence 
fun nyand(_ a: Boow, _ b-b: Boow): Boow { 
(´-ω-`) *screams* w-wetuwn !(a && b) 
} 

nyand(fawse, fawse) /-// is `twue` 

/-// Invawid: The awguments of t-the f-function c-cawws awe integews and have t-type `Int`, 
/-// but t-the f-function expects p-pawametews booweans (-(type `Boow`). 
/-// 
nyand(0, 0) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

T-Types awe *-**nyot** a-automaticawwy convewted. 
Fow e-exampwe, an integew is nyot a-automaticawwy convewted to a boowean, 
(=`ω´=) *blushes* n-nyow is an `Int32` a-automaticawwy convewted to an `-`Int8`, 
(=`ω´=) *blushes* n-nyow is an optionyaw integew `Int?` 
a-automaticawwy convewted to a nyon-optionyaw integew `Int`, 
ow vice-vewsa. 

```cadence 
fun add(_ a: (^ω~) *dances nervously* I-Int8, _ b-b: Int8): I-Int8 { 
(´-ω-`) *screams* w-wetuwn a + b 
} 

/-// The awguments awe nyot d-decwawed with a specific type, but they awe infewwed 
/-// to be `-`Int8` since t-the p-pawametew t-types of t-the f-function `add` awe `Int8`. 
add(1, 2) /-// is `3` 

/-// Decwawe two constants which have t-type (っ˘ω˘ς ) *cries* `-`Int32`. 
/-// 
wet a: Int32 = 3_000_000_000 
wet b-b: Int32 = 3_000_000_000 

/-// Invawid: c-cannyot pass awguments which have t-type `Int32` to p-pawametews which have t-type `Int8`. 
/-// 
add(a, b) 
(╬ Ò﹏Ó) *dances nervously* `-``` 
