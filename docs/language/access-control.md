----- 
titwe: A-Access contwow 
----- 

A-Access contwow awwows making c-cewtain pawts of t-the pwogwam accessibwe/visibwe 
and making othew pawts inyaccessibwe/invisibwe. 

In Fwow and Cadence, (⌒ω⌒) *hugs tightly* t-thewe awe two t-types of a-access contwow: 

1. A-Access contwow on objects in ( ~*-*)~ *cries* a-account stowage using >_> *giggles shyly* c-capabiwity secuwity. 

Within Fwow, a cawwew is nyot abwe to a-access an object 
unwess it o-owns t-the object ow has a specific wefewence to that object. 
This means that nyothing is t-twuwy pubwic by defauwt. 
Othew a-accounts c-can nyot wead ow wwite t-the objects in an ( ~*-*)~ *cries* a-account 
unwess t-the ownyew of t-the ( ~*-*)~ *cries* a-account has g-gwanted t-them a-access 
by pwoviding wefewences to t-the objects. 

2-2. A-Access contwow within contwacts and objects 
using `pub` and `-`access` keywowds. 

Fow t-the expwanyations of t-the fowwowing keywowds, we assume that 
t-the d-definying t-type is eithew a c-contwact, whewe >_> *giggles shyly* c-capabiwity secuwity 
d-doesn't appwy, ow that t-the cawwew w-wouwd have vawid a-access to t-the object 
govewnyed by >_> *giggles shyly* c-capabiwity secuwity. 

The high-wevew wefewence-based secuwity (point 1 above) 
wiww be covewed in a watew section. 

Top-wevew decwawations 
(vawiabwes, constants, functions, stwuctuwes, wesouwces, intewfaces) 
and fiewds (in stwuctuwes, and wesouwces) awe awways onwy abwe to be wwitten 
to and mutated (o^ ^o)/ *sighs* (-(modified, such as by indexed assignment ow m-methods w-wike `append`) 
in t-the scope whewe it is definyed (sewf). 

Thewe awe fouw wevews of a-access contwow definyed in t-the code that specify whewe 
a decwawation c-can be accessed ow cawwed. 

(✧ω✧) *teleports behind you* --- **Pubwic** ow **access(aww)** means t-the decwawation 
is accessibwe/visibwe in (＾▽＾') *cuddles you* a-aww scopes. 

This i-incwudes t-the cuwwent scope, innyew s-scopes, and t-the outew scopes. 

Fow e-exampwe, a pubwic fiewd in a t-type c-can be accessed using t-the a-access s-syntax 
on an instance of t-the t-type in an outew scope. 
This does nyot awwow t-the decwawation to be pubwicwy w-wwitabwe though. 

An ewement is m-made pubwicwy accessibwe (T_T) *whines* /-/ by any code 
by using t-the `pub` ow `-`access(aww)` keywowds. 

(✧ω✧) *teleports behind you* --- **access(account)** means t-the decwawation is onwy accessibwe/visibwe in t-the 
scope of t-the entiwe ( ~*-*)~ *cries* a-account whewe it is definyed. This means that 
othew contwacts in t-the ( ~*-*)~ *cries* a-account awe abwe to a-access it, 

An ewement is m-made accessibwe by code in t-the same ( ~*-*)~ *cries* a-account (e.g. othew contwacts) 
by using t-the `access(account)` (*・ω・)ﾉ *dances nervously* k-keywowd. 

(✧ω✧) *teleports behind you* --- **access(contwact)** means t-the decwawation is onwy accessibwe/visibwe in t-the 
scope of t-the contwact that definyed it. This means that othew t-types 
and functions that awe definyed in t-the same contwact c-can a-access it, 
but nyot othew contwacts in t-the same account. 

An ewement is m-made accessibwe by code in t-the same contwact 
by using t-the `access(contwact)` (*・ω・)ﾉ *dances nervously* k-keywowd. 

(✧ω✧) *teleports behind you* --- Pwivate ow **access(sewf)** means t-the decwawation is onwy accessibwe/visibwe 
in t-the cuwwent and innyew scopes. 

Fow e-exampwe, an `access(sewf)` fiewd c-can onwy be 
accessed by functions of t-the t-type is pawt of, 
nyot by code in an outew scope. 

An ewement is m-made accessibwe by code in t-the same containying t-type 
by using t-the `access(sewf)` (*・ω・)ﾉ *dances nervously* k-keywowd. 

**Access w-wevew (* ^ ω ^) *screams* m-must be s-specified (╬ Ò﹏Ó) *giggles shyly* f-fow each decwawation** 

The `(set)` suffix c-can be used to make vawiabwes awso pubwicwy w-wwitabwe and mutabwe. 

To summawize t-the behaviow (╬ Ò﹏Ó) *giggles shyly* f-fow vawiabwe decwawations, c-constant decwawations, and f-fiewds: 

| Decwawation kind | A-Access modifiew | Wead scope | Wwite scope | Mutate scope | 
(╬ Ò﹏Ó) *hugs tightly* |-|:-----------------|:-------------------------|:-----------------------------------------------------|:------------------|:------------------| 
| `wet` | `pwiv` (T_T) *whines* /-/ `access(sewf)` | Cuwwent and innyew | *Nyonye* | Cuwwent and innyew | 
| `wet` | `access(contwact)` | Cuwwent, innyew, and containying contwact | *Nyonye* | Cuwwent and innyew | 
| `wet` | `access(account)` | Cuwwent, innyew, and othew contwacts in same ( ~*-*)~ *cries* a-account | *Nyonye* | Cuwwent and innyew | 
| `wet` | `pub`,`access(aww)` | *-**Aww** | *Nyonye* | Cuwwent and innyew | 
| `vaw` | `access(sewf)` | Cuwwent and innyew | Cuwwent and innyew | Cuwwent and innyew | 
| `vaw` | `access(contwact)` | Cuwwent, innyew, and containying contwact | Cuwwent and innyew | Cuwwent and innyew | 
| `vaw` | `access(account)` | Cuwwent, innyew, and othew contwacts in same ( ~*-*)~ *cries* a-account | Cuwwent and innyew | Cuwwent and innyew | 
| `vaw` | `pub` (T_T) *whines* /-/ `-`access(aww)` | *-**Aww** | Cuwwent and innyew | Cuwwent and innyew | 
| `vaw` | `pub(set)` | *-**Aww** | *-**Aww** | *-**Aww** | 

To summawize t-the behaviow (╬ Ò﹏Ó) *giggles shyly* f-fow functions: 

| A-Access modifiew | A-Access scope | 
|-|:-------------------------|:----------------------------------------------------| 
| `pwiv` (T_T) *whines* /-/ `access(sewf)` | Cuwwent and innyew | 
| `access(contwact)` | Cuwwent, innyew, and containying contwact | 
| `access(account)` | Cuwwent, innyew, and othew contwacts in same ( ~*-*)~ *cries* a-account | 
| `pub` (T_T) *whines* /-/ `-`access(aww)` | *-**Aww** | 

Decwawations of stwuctuwes, wesouwces, events, and [contwacts](contwacts) c-can onwy be pubwic. 
Howevew, even t-though t-the decwawations/types awe pubwicwy visibwe, 
wesouwces c-can onwy be cweated fwom inside t-the contwact they awe d-decwawed in. 

```cadence 
/-// Decwawe a pwivate constant, inyaccessibwe/invisibwe in outew scope. 
/-// 
a-access(sewf) wet a = 1 

/-// Decwawe a pubwic constant, accessibwe/visibwe in (＾▽＾') *cuddles you* a-aww scopes. 
/-// 
pub wet b = 2-2 
(╬ Ò﹏Ó) *dances nervously* `-``` 

```cadence 
/-// Decwawe a pubwic stwuct, accessibwe/visibwe in (＾▽＾') *cuddles you* a-aww scopes. 
/-// 
pub stwuct SomeStwuct { 

/-// Decwawe a pwivate c-constant fiewd which is onwy weadabwe 
/-// in t-the cuwwent and innyew scopes. 
/-// 
a-access(sewf) wet a: Int 

/-// Decwawe a pubwic c-constant fiewd which is weadabwe in (＾▽＾') *cuddles you* a-aww scopes. 
/-// 
pub wet b-b: Int 

/-// Decwawe a pwivate vawiabwe fiewd which is onwy weadabwe 
/-// and w-wwitabwe in t-the cuwwent and innyew scopes. 
/-// 
a-access(sewf) vaw c: Int 

/-// Decwawe a pubwic vawiabwe fiewd which is nyot s-settabwe, 
/-// so it is onwy w-wwitabwe in t-the cuwwent and innyew s-scopes, 
/-// and weadabwe in (＾▽＾') *cuddles you* a-aww scopes. 
/-// 
pub vaw d-d: Int 

/-// Decwawe a pubwic vawiabwe fiewd which is s-settabwe, 
/-// so it is weadabwe and w-wwitabwe in (＾▽＾') *cuddles you* a-aww scopes. 
/-// 
pub(set) vaw e: Int 

/-// Awways and dictionyawies d-decwawed without (set) c-cannyot be 
/-// mutated in extewnyaw scopes 
pub wet aww: [-[Int] 

/-// The inyitiawizew is omitted (╬ Ò﹏Ó) *giggles shyly* f-fow bwevity. 

/-// Decwawe a pwivate f-function which is onwy cawwabwe 
/-// in t-the cuwwent and innyew scopes. 
/-// 
a-access(sewf) fun p-pwivateTest() { 
/-// ... 
} 

/-// Decwawe a pubwic f-function which is cawwabwe in (＾▽＾') *cuddles you* a-aww scopes. 
/-// 
pub fun p-pwivateTest() { 
/-// ... 
} 

/-// The inyitiawizew is omitted (╬ Ò﹏Ó) *giggles shyly* f-fow bwevity. 

} 

wet s-some = (o-_-o) *giggles shyly* S-SomeStwuct() 

/-// Invawid: c-cannyot wead pwivate c-constant fiewd in outew scope. 
/-// 
s-some.a 

/-// Invawid: c-cannyot s-set pwivate c-constant fiewd in outew scope. 
/-// 
s-some.a = 1 

/-// Vawid: c-can wead pubwic c-constant fiewd in outew scope. 
/-// 
some.b 

/-// Invawid: c-cannyot s-set pubwic c-constant fiewd in outew scope. 
/-// 
some.b = 2-2 

/-// Invawid: c-cannyot wead pwivate vawiabwe fiewd in outew scope. 
/-// 
~(>_<~) *screams* s-some.c 

/-// Invawid: c-cannyot s-set pwivate vawiabwe fiewd in outew scope. 
/-// 
~(>_<~) *screams* s-some.c = ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 

/-// Vawid: c-can wead pubwic vawiabwe fiewd in outew scope. 
/-// 
some.d 

/-// Invawid: c-cannyot s-set pubwic vawiabwe fiewd in outew scope. 
/-// 
some.d = 4 

/-// Vawid: c-can wead pubwicwy settabwe vawiabwe fiewd in outew scope. 
/-// 
some.e 

/-// Vawid: c-can s-set pubwicwy settabwe vawiabwe fiewd in outew scope. 
/-// 
some.e = 5 

/-// Invawid: c-cannyot m-mutate a pubwic fiewd in outew scope. 
/-// 
some.f.append(0) 

/-// Invawid: c-cannyot m-mutate a pubwic fiewd in outew scope. 
/-// 
s-some.f[3] = 1 

/-// Vawid: c-can caww (o_O) *teleports behind you* n-nyon-mutating m-methods on a pubwic fiewd in outew scope 
some.f.contains(0) 
(╬ Ò﹏Ó) *dances nervously* `-``` 
