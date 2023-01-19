----- 
titwe: Contwact Updatabiwity 
----- 

#-## Intwoduction 
A-A [contwact](contwacts) in C-Cadence is a cowwection of data (its state) and 
code (its f-functions) that wives in t-the contwact stowage a-awea of an account. 
When a contwact is updated, it is impowtant to make suwe that t-the changes intwoduced (^-^*)/ *cuddles you* d-do nyot wead to wuntime 
inconsistencies (╬ Ò﹏Ó) *giggles shyly* f-fow awweady stowed d-data. 
C-Cadence maintains this state c-consistency by vawidating t-the contwacts and (＾▽＾') *cuddles you* a-aww theiw componyents befowe an u-update. 

#-## Vawidation >w< *leans over* G-Goaws 
The contwact update v-vawidation ensuwes that: 

(✧ω✧) *teleports behind you* --- Stowed data d-doesn't change its meanying when a contwact is updated. 
(✧ω✧) *teleports behind you* --- Decoding and using stowed data does nyot wead to wuntime c-cwashes. 
(✧ω✧) *teleports behind you* --- Fow e-exampwe, it is invawid to a-add a fiewd because existing stowed data w-won't have t-the nyew fiewd. 
(✧ω✧) *teleports behind you* --- Woading t-the existing data wiww wesuwt in g-gawbage/missing vawues (╬ Ò﹏Ó) *giggles shyly* f-fow such fiewds. 
(✧ω✧) *teleports behind you* --- A-A static check of t-the a-access of t-the fiewd w-wouwd be vawid, but t-the intewpwetew w-wouwd c-cwash when a-accessing t-the fiewd, 
because t-the fiewd has a missing/gawbage vawue. 

Howevew, it **does nyot** ensuwe: 
(✧ω✧) *teleports behind you* --- Any pwogwam that impowts t-the u-updated contwact stays vawid. e.g: 
(✧ω✧) *teleports behind you* --- Updated contwact may w-wemove an existing fiewd ow may change a f-function signyatuwe. 
(✧ω✧) *teleports behind you* --- T-Then any pwogwam that uses that fiewd/function wiww get semantic ewwows. 

#-## Updating a Contwact 
Changes to contwacts c-can be intwoduced by adding nyew contwacts, w-wemoving existing contwacts, ow u-updating existing 
~(>_<~) *screams* c-contwacts. Howevew, s-some of these changes may wead to data inconsistencies as stated above. 

#-#### V-Vawid Changes 
(✧ω✧) *teleports behind you* --- A-Adding a nyew contwact is vawid. 
(✧ω✧) *teleports behind you* --- Wemoving a contwact/contwact-intewface that d-doesn't have enyum decwawations is vawid. 
(✧ω✧) *teleports behind you* --- Updating a contwact is vawid, undew t-the westwictions descwibed in t-the bewow sections. 

#-#### (＃`Д´) *screams* I-Invawid Changes 
(✧ω✧) *teleports behind you* --- Wemoving a contwact/contwact-intewface that contains enyum decwawations is nyot vawid. 
(✧ω✧) *teleports behind you* --- Wemoving a contwact awwows adding a nyew contwact with t-the same nyame. 
(✧ω✧) *teleports behind you* --- The nyew contwact couwd potentiawwy have enyum decwawations with t-the same nyames as in t-the o-owd c-contwact, but with 
diffewent s-stwuctuwes. 
(✧ω✧) *teleports behind you* --- This couwd change t-the meanying of t-the awweady stowed vawues of those enyum types. 

A-A contwact may consist of fiewds and othew decwawations such as composite types, functions, constwuctows, etc. 
When an existing contwact is updated, (＾▽＾') *cuddles you* a-aww its innyew decwawations awe awso vawidated. 

### Contwact F-Fiewds 
When a contwact is >w< *shuffles closer* d-depwoyed, t-the fiewds of t-the contwact awe stowed in an account's contwact stowage. 
Changing t-the fiewds of a contwact onwy changes t-the w-way t-the pwogwam tweats t-the data, but does nyot change t-the awweady 
stowed data (ᗒᗣᗕ)՞ *shuffles closer* i-itsewf, which couwd potentiawwy wesuwt in wuntime inconsistencies as m-mentionyed in t-the pwevious section. 

See t-the [section about fiewds b-bewow](#fiewds) (╬ Ò﹏Ó) *giggles shyly* f-fow t-the p-possibwe updates that c-can be donye to t-the fiewds, and t-the westwictions 
imposed on changing fiewds of a contwact. 

### Nyested Decwawations 
Contwacts c-can have nyested composite t-type decwawations such as s-stwucts, wesouwces, intewfaces, and enyums. 
When a contwact is updated, its nyested decwawations awe checked, b-because: 
(✧ω✧) *teleports behind you* --- They c-can be used as t-type (*^‿^*) *sighs* a-annyotation (╬ Ò﹏Ó) *giggles shyly* f-fow t-the fiewds of t-the same c-contwact, diwectwy ow indiwectwy. 
(✧ω✧) *teleports behind you* --- Any thiwd-pawty contwact c-can i-impowt t-the t-types definyed in this contwact and use t-them as t-type annyotations. 
(✧ω✧) *teleports behind you* --- Hence, changing t-the t-type definyition is t-the same as changing t-the t-type (*^‿^*) *sighs* a-annyotation of such a fiewd (-(which is awso invawid, 
as descwibed in t-the [section about fiewds f-fiewds](#fiewds) b-bewow). 

Changes that c-can be donye to t-the nyested decwawations, and t-the update westwictions awe descwibed in fowwowing s-sections: 
(✧ω✧) *teleports behind you* --- [Stwucts, wesouwces and intewface](#stwucts-wesouwces-and-intewfaces) 
(✧ω✧) *teleports behind you* --- [Enyums](#enyums) 
(✧ω✧) *teleports behind you* --- [Functions](#functions) 
(✧ω✧) *teleports behind you* --- [-[Constwuctows](#constwuctows) 

#-## F-Fiewds 
A-A fiewd may bewong to a c-contwact, stwuct, wesouwce, ow intewface. 

#-#### V-Vawid (*´▽`*) *pokes you* C-Changes: 
(✧ω✧) *teleports behind you* --- Wemoving a fiewd is vawid 
```cadence 
/-// Existing contwact 

pub contwact Foo { 
pub vaw a: Stwing 
pub vaw b-b: Int 
} 


/-// Updated contwact 

pub contwact Foo { 
pub vaw a: Stwing 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- I-It weaves data (╬ Ò﹏Ó) *giggles shyly* f-fow t-the w-wemoved fiewd unyused a-at t-the stowage, as it is nyo wongew accessibwe. 
(✧ω✧) *teleports behind you* --- Howevew, it does nyot c-cause any wuntime c-cwashes. 

(✧ω✧) *teleports behind you* --- Changing t-the owdew of fiewds is vawid. 
```cadence 
/-// Existing contwact 

pub contwact Foo { 
pub vaw a: Stwing 
pub vaw b-b: Int 
} 


/-// Updated contwact 

pub contwact Foo { 
pub vaw b-b: Int 
pub vaw a: Stwing 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

(✧ω✧) *teleports behind you* --- Changing t-the a-access modifiew of a fiewd is vawid. 
```cadence 
/-// Existing contwact 

pub contwact Foo { 
pub vaw a: Stwing 
} 


/-// Updated contwact 

pub contwact Foo { 
pwiv vaw a: Stwing /-// a-access modifiew changed to '-'pwiv' 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-#### (＃`Д´) *screams* I-Invawid Changes 
(✧ω✧) *teleports behind you* --- A-Adding a nyew fiewd is nyot vawid. 
```cadence 
/-// Existing contwact 

pub contwact Foo { 
pub vaw a: Stwing 
} 


/-// Updated contwact 

pub contwact Foo { 
pub vaw a: Stwing 
pub vaw b-b: Int /-// (＃`Д´) *screams* I-Invawid nyew fiewd 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- Inyitiawizew of a contwact onwy w-wun once, when t-the contwact is depwoyed (╬ Ò﹏Ó) *giggles shyly* f-fow t-the fiwst ^w^ *screams* t-time. I-It does nyot wewun 
when t-the contwact is updated. Howevew it is stiww wequiwed to be pwesent in t-the u-updated contwact to s-satisfy t-type checks. 
(✧ω✧) *teleports behind you* --- T-Thus, t-the stowed data w-won't have t-the nyew fiewd, as t-the inyitiawizations (╬ Ò﹏Ó) *giggles shyly* f-fow t-the nyewwy a-added fiewds (^-^*)/ *cuddles you* d-do nyot get 
executed. 
(✧ω✧) *teleports behind you* --- Decoding stowed data wiww wesuwt in gawbage ow missing vawues (╬ Ò﹏Ó) *giggles shyly* f-fow such fiewds. 

(✧ω✧) *teleports behind you* --- Changing t-the t-type of existing fiewd is nyot vawid. 
```cadence 
/-// Existing contwact 

pub contwact Foo { 
pub vaw a: Stwing 
} 


/-// Updated contwact 

pub contwact Foo { 
pub vaw a: Int /-// (＃`Д´) *screams* I-Invawid t-type change 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- In an awweady stowed c-contwact, t-the fiewd `a` w-wouwd have a vawue of t-type `Stwing`. 
(✧ω✧) *teleports behind you* --- Changing t-the t-type of t-the fiewd `a` to `Int`, w-wouwd make t-the wuntime wead t-the awweady stowed `-`Stwing` 
vawue as an `Int`, which wiww wesuwt in desewiawization ewwows. 
(✧ω✧) *teleports behind you* --- Changing t-the fiewd t-type to a subtype/supewtype of t-the existing t-type is awso nyot vawid, as it w-wouwd awso 
potentiawwy c-cause issues whiwe decoding/encoding. 
(✧ω✧) *teleports behind you* --- e.g: Changing an `Int64` fiewd to `-`Int8` (✧ω✧) *teleports behind you* --- Stowed fiewd couwd have a nyumewic vawue`624`, which exceeds t-the vawue space 
(╬ Ò﹏Ó) *giggles shyly* f-fow `Int8`. 
(✧ω✧) *teleports behind you* --- Howevew, this is a wimitation in t-the cuwwent impwementation, and t-the futuwe vewsions of C-Cadence may suppowt 
changing t-the t-type of fiewd to a subtype, by pwoviding means to migwate existing fiewds. 

#-## Stwucts, Wesouwces and I-Intewfaces 

#-#### V-Vawid (*´▽`*) *pokes you* C-Changes: 
(✧ω✧) *teleports behind you* --- A-Adding a nyew stwuct, wesouwce, ow intewface is vawid. 
(✧ω✧) *teleports behind you* --- A-Adding an intewface c-confowmance to a stwuct/wesouwce is vawid, since t-the stowed data onwy 
stowes concwete type/vawue, but d-doesn't stowe t-the c-confowmance info. 
```cadence 
/-// Existing stwuct 

pub stwuct Foo { 
} 


/-// Upated stwuct 

pub stwuct Foo: T { 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- Howevew, if adding a c-confowmance awso wequiwes changing t-the existing stwuctuwe (e.g: adding a nyew fiewd that is 
enfowced by t-the nyew confowmance), t-then t-the othew westwictions (such as [westwictions on fiewds](#fiewds)) may 
pwevent p-pewfowming such an u-update. 

#-#### (＃`Д´) *screams* I-Invawid (*´▽`*) *pokes you* C-Changes: 
(✧ω✧) *teleports behind you* --- Wemoving an existing decwawation is nyot vawid. 
(✧ω✧) *teleports behind you* --- Wemoving a decwawation awwows adding a nyew decwawation with t-the same nyame, but with a diffewent stwuctuwe. 
(✧ω✧) *teleports behind you* --- Any pwogwam that uses that decwawation w-wouwd f-face inconsistencies in t-the stowed d-data. 
(✧ω✧) *teleports behind you* --- Wenyaming a decwawation is nyot vawid. I-It c-can have t-the same effect as w-wemoving an existing decwawation and adding 
a nyew onye. 
(✧ω✧) *teleports behind you* --- Changing t-the t-type of decwawation is nyot vawid. i-i.e: Changing fwom a stwuct to intewface, and vise vewsa. 
```cadence 
/-// Existing stwuct 

pub stwuct Foo { 
} 


/-// C-Changed to a stwuct intewface 

pub stwuct intewface Foo { /-// (＃`Д´) *screams* I-Invawid t-type decwawation change 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- Wemoving an intewface c-confowmance of a stwuct/wesouwce is nyot vawid. 
```cadence 
/-// Existing stwuct 

pub stwuct Foo: T { 
} 


/-// Upated stwuct 

pub stwuct Foo { 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Updating Membews 
Simiwaw to contwacts, these composite decwawations: s-stwucts, wesouwces, and i-intewfaces awso c-can have fiewds and 
othew nyested decwawations as its m-membew. 
Updating such a composite decwawation w-wouwd awso i-incwude u-updating (＾▽＾') *cuddles you* a-aww of its membews. 

Bewow sections descwibes t-the westwictions imposed on u-updating t-the membews of a stwuct, wesouwce ow an intewface. 
(✧ω✧) *teleports behind you* --- [-[Fiewds](#fiewds) 
(✧ω✧) *teleports behind you* --- [Nyested s-stwucts, wesouwces and (☆▽☆) *leans over* i-intewfaces](#stwucts-wesouwces-and-intewfaces) 
(✧ω✧) *teleports behind you* --- [Enyums](#enyums) 
(✧ω✧) *teleports behind you* --- [Functions](#functions) 
(✧ω✧) *teleports behind you* --- [-[Constwuctows](#constwuctows) 

#-## Enyums 

#-#### V-Vawid (*´▽`*) *pokes you* C-Changes: 
(✧ω✧) *teleports behind you* --- A-Adding a nyew enyum decwawation is vawid. 

#-#### (＃`Д´) *screams* I-Invawid (*´▽`*) *pokes you* C-Changes: 
(✧ω✧) *teleports behind you* --- Wemoving an existing enyum decwawation is invawid. 
(✧ω✧) *teleports behind you* --- O-Othewwise, it is p-possibwe to w-wemove an existing enyum and a-add a nyew enyum decwawation with t-the same nyame, 
but with a diffewent stwuctuwe. 
(✧ω✧) *teleports behind you* --- The nyew stwuctuwe couwd potentiawwy have incompatibwe changes (such as changed types, changed enyum-cases, etc). 
(✧ω✧) *teleports behind you* --- Changing t-the nyame is invawid, as it is equivawent to w-wemoving an existing enyum and adding a nyew onye. 
(✧ω✧) *teleports behind you* --- Changing t-the waw t-type is invawid. 
```cadence 
/-// Existing enyum with ^.^ *sweats* `-`Int` waw t-type 

pub enyum Cowow: Int { 
pub case WED 
pub case BWUE 
} 


/-// Updated enyum with `-`UInt8` waw t-type 

pub enyum Cowow: UInt8 { /-// (＃`Д´) *screams* I-Invawid change of waw t-type 
pub case WED 
pub case BWUE 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- When t-the enyum vawue is s-stowed, t-the waw vawue associated with t-the enyum-case gets stowed. 
(✧ω✧) *teleports behind you* --- If t-the t-type is changed, t-then desewiawizing couwd faiw if t-the awweady stowed vawues awe nyot in t-the same vawue space 
as t-the u-updated (＃￣ω￣) *hugs tightly* t-type. 

### Updating Enyum C-Cases 
Enyums consist of enyum-case decwawations, and u-updating an enyum may awso i-incwude changing t-the enyums cases as weww. 
Enyum cases awe wepwesented using theiw waw-vawue a-at t-the C-Cadence intewpwetew and wuntime. 
Hence, any change that causes an enyum-case to change its waw vawue is nyot pewmitted. 
O-Othewwise, a changed waw-vawue couwd c-cause an awweady stowed enyum vawue to have a diffewent meanying than what 
it o-owiginyawwy w-was (-(type c-confusion). 

#-#### V-Vawid (*´▽`*) *pokes you* C-Changes: 
(✧ω✧) *teleports behind you* --- A-Adding an enyum-case a-at t-the end of t-the existing enyum-cases is vawid. 
```cadence 
/-// Existing enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case BWUE 
} 


/-// Updated enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case BWUE 
pub case G-GWEEN /-// vawid nyew enyum-case a-at t-the bottom 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
#-#### (＃`Д´) *screams* I-Invawid Changes 
(✧ω✧) *teleports behind you* --- A-Adding an enyum-case a-at t-the top ow in t-the middwe of t-the existing enyum-cases is invawid. 
```cadence 
/-// Existing enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case BWUE 
} 


/-// Updated enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case G-GWEEN /-// invawid nyew enyum-case in t-the middwe 
pub case BWUE 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- Changing t-the nyame of an enyum-case is invawid. 
```cadence 
/-// Existing enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case BWUE 
} 


/-// Updated enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case G-GWEEN /-// invawid change of nyames 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- Pweviouswy stowed waw vawues (╬ Ò﹏Ó) *giggles shyly* f-fow `Color.BLUE` nyow wepwesents `Cowow.GWEEN`. i-i.e: The stowed vawues have changed 
theiw meanying, and hence nyot a vawid change. 
(✧ω✧) *teleports behind you* --- S-Simiwawwy, it is p-possibwe to a-add a nyew enyum with t-the o-owd nyame `BWUE`, which gets a nyew waw vawue. T-Then t-the same 
enyum-case `Color.BLUE` may have used two waw-vawues a-at wuntime, befowe and aftew t-the change, which is awso invawid. 

(✧ω✧) *teleports behind you* --- Wemoving t-the enyum case is invawid. Wemoving awwows onye to a-add and w-wemove an enyum-case which has t-the same effect 
as wenyaming. 
```cadence 
/-// Existing enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case BWUE 
} 


/-// Updated enyum 

pub enyum Cowow: Int { 
pub case WED 

/-// invawid w-wemovaw of `case OwO *looks away* B-BWUE` 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- Changing t-the owdew of enyum-cases is nyot pewmitted 
```cadence 
/-// Existing enyum 

pub enyum Cowow: Int { 
pub case WED 
pub case BWUE 
} 


/-// Updated enyum 

pub enyum Cowow: UInt8 { 
pub case BWUE /-// invawid change of owdew 
pub case WED 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
(✧ω✧) *teleports behind you* --- Waw vawue of an enyum is i-impwicit, and cowwesponds to t-the definyed owdew. 
(✧ω✧) *teleports behind you* --- Changing t-the owdew of enyum-cases has t-the same effect as changing t-the w-waw-vawue, which couwd c-cause stowage 
inconsistencies and t-type-confusions as descwibed eawwiew. 

#-## Functions 

Adding, changing, and deweting a f-function definyition is awways vawid, as f-function definyitions awe nyevew stowed as data 
(-(function definyitions awe pawt of t-the code, but nyot data). 

(✧ω✧) *teleports behind you* --- A-Adding a f-function is vawid. 
(✧ω✧) *teleports behind you* --- D-Deweting a f-function is vawid. 
(✧ω✧) *teleports behind you* --- Changing a f-function signyatuwe (pawametews, (´-ω-`) *screams* w-wetuwn types) is vawid. 
(✧ω✧) *teleports behind you* --- Changing a f-function body is vawid. 
(✧ω✧) *teleports behind you* --- Changing t-the a-access modifiews is vawid. 

Howevew, changing a *-*function type* may ow may nyot be vawid, depending on whewe it is used: 
If a f-function t-type is used in t-the t-type (*^‿^*) *sighs* a-annyotation of a composite t-type fiewd (diwect ow indiwect), 
t-then changing t-the f-function t-type signyatuwe is t-the same as changing t-the t-type (*^‿^*) *sighs* a-annyotation of that fiewd (-(which is invawid). 

#-## Constwuctows 
Simiwaw to functions, constwuctows awe awso nyot stowed. Hence, any changes to constwuctows awe vawid. 

#-## Impowts 
A-A contwact may i-impowt decwawations (types, functions, vawiabwes, etc.) fwom othew pwogwams. These impowted (ノωヽ) *hugs tightly* p-pwogwams awe 
awweady vawidated a-at t-the time of theiw depwoyment. Hence, (⌒ω⌒) *hugs tightly* t-thewe is nyo nyeed (╬ Ò﹏Ó) *giggles shyly* f-fow vawidating any decwawation {{ (>_<) }} *steals ur resource* e-evewy time 
they awe i-impowted. 
