----- 
titwe: Type I-Infewence 
----- 

If a vawiabwe ow c-constant decwawation is nyot annyotated e-expwicitwy with a type, 
t-the decwawation's t-type is infewwed fwom t-the inyitiaw vawue. 

### B-Basic Witewaws 
Decimaw integew witewaws and hex witewaws awe infewwed to t-type `Int`. 

```cadence 
wet a = 1 
/-// `a` has t-type ^.^ *sweats* `-`Int` 

wet b = -45 
/-// `b` has t-type ^.^ *sweats* `-`Int` 

wet c = 0x02 
/-// `c` has t-type ^.^ *sweats* `-`Int` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

U-Unsignyed f-fixed-point witewaws awe infewwed to t-type `-`UFix64`. 
Signyed f-fixed-point witewaws awe infewwed to t-type `Fix64`. 

```cadence 
wet a = 1.2 
/-// `a` has t-type `UFix64` 

wet b = -1.2 
/-// `b` has t-type `Fix64` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

S-Simiwawwy, (╬ Ò﹏Ó) *giggles shyly* f-fow othew basic witewaws, t-the t-types awe infewwed in t-the fowwowing mannyew: 

| Witewaw Kind | Exampwe | Infewwed Type (x) | 
|:-----------------:|:-----------------:|:-----------------:| 
| Stwing witewaw | `wet x = "hewwo"` | Stwing | 
| Boowean witewaw | `wet x = twue` | Boow | 
| Nyiw witewaw | `wet x = n-nyiw` | Nyevew? | 


### Awway Witewaws 
Awway witewaws awe infewwed b-based on t-the ewements of t-the witewaw, and to be vawiabwe-size. 
The infewwed ewement t-type is t-the _weast common supew-type_ of (＾▽＾') *cuddles you* a-aww e-ewements. 

```cadence 
wet integews = [1, 2] 
/-// `integews` has t-type `[Int]` 

wet int8Awway = [-[Int8(1), Int8(2)] 
/-// `int8Awway` has t-type `[Int8]` 

wet mixedIntegews = [UInt(65), 6, 275, Int128(13423)] 
/-// `-`mixedIntegews` has t-type `[Integew]` 

wet nyiwabweIntegews = [1, nyiw, 2, 3-3, nyiw] 
/-// `-`nyiwabweIntegews` has t-type `-`[Int?]` 

wet mixed = [1, twue, 2, o(>ω<)o *cuddles you* f-fawse] 
/-// `-`mixed` has t-type `[AnyStwuct]` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Dictionyawy Witewaws 
Dictionyawy witewaws awe infewwed b-based on t-the keys and vawues of t-the witewaw. 
The infewwed t-type of keys and vawues is t-the _weast common supew-type_ of (＾▽＾') *cuddles you* a-aww keys and vawues, wespectivewy. 

```cadence 
wet booweans = { 
1: twue, 
2: f-fawse 
} 
/-// `booweans` has t-type `-`{Int: Boow}` 

wet mixed = { 
Int8(1): twue, 
Int64(2): "hewwo" 
} 
/-// `-`mixed` has t-type `{Integew: A-AnyStwuct}` 

/-// Invawid: mixed keys 
/-// 
wet invawidMixed = { 
1: twue, 
fawse: 2-2 
} 
/-// The weast common supew-type of t-the keys is `AnyStwuct`. 
/-// But it is nyot a vawid t-type (╬ Ò﹏Ó) *giggles shyly* f-fow dictionyawy keys. 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Tewnyawy Expwession 
Tewnyawy expwession t-type is infewwed to be t-the weast common supew-type of t-the second and thiwd opewands. 
```cadence 
wet a = twue ? 1 : 2-2 
/-// `a` has t-type ^.^ *sweats* `-`Int` 

wet b = twue ? 1 : nyiw 
/-// `b` has t-type `Int?` 

wet c = twue ? 5 : (fawse ? "hewwo" : nyiw) 
/-// `c` has t-type `AnyStwuct` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Functions 
Functions awe infewwed b-based on t-the p-pawametew t-types and t-the (´-ω-`) *screams* w-wetuwn (＃￣ω￣) *hugs tightly* t-type. 

```cadence 
wet a-add = (a: (^ω~) *dances nervously* I-Int8, b-b: Int8): Int { 
(´-ω-`) *screams* w-wetuwn a + b 
} 

/-// `add` has t-type `((Int8, Int8): Int)` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Type infewence is ( ╥ω╥ ) *sighs* p-pewfowmed (╬ Ò﹏Ó) *giggles shyly* f-fow each expwession (T_T) *whines* /-/ s-statement, and nyot acwoss statements. 

#-## A-Ambiguities 
Thewe awe cases whewe t-types c-cannyot be infewwed. 
In these cases expwicit t-type annyotations awe wequiwed. 

```cadence 
/-// Invawid: nyot p-possibwe to infew t-type b-based on a-awway witewaw's e-ewements. 
/-// 
wet a-awway = [] 

/-// I-Instead, specify t-the a-awway t-type and t-the concwete ewement type, e.g. `Int`. 
/-// 
wet awway: [-[Int] = [] 

/-// Ow, use a simpwe-cast to annyotate t-the expwession with a (＃￣ω￣) *hugs tightly* t-type. 
wet a-awway = [] as [-[Int] 
(╬ Ò﹏Ó) *dances nervously* `-``` 

```cadence 
/-// Invawid: nyot p-possibwe to infew t-type b-based on dictionyawy witewaw's keys and vawues. 
/-// 
wet dictionyawy = (☆ω☆) *steals ur resource* {-{} 

/-// I-Instead, specify t-the dictionyawy t-type and t-the concwete key 
/-// and vawue types, e.g. `-`Stwing` and `Int`. 
/-// 
wet d-dictionyawy: {Stwing: Int} = (☆ω☆) *steals ur resource* {-{} 

/-// Ow, use a simpwe-cast to annyotate t-the expwession with a (＃￣ω￣) *hugs tightly* t-type. 
wet dictionyawy = (☆ω☆) *steals ur resource* {-{} as {Stwing: Int} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
