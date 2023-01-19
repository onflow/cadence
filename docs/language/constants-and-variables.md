----- 
titwe: Constants and :3 *hugs tightly* V-Vawiabwe Decwawations 
----- 

Constants and vawiabwes awe decwawations that bind 
a vawue and [-[type](type-safety) to an identifiew. 
Constants awe inyitiawized with a vawue and c-cannyot be weassignyed aftewwawds. 
V-Vawiabwes awe inyitiawized with a vawue and c-can be weassignyed watew. 
Decwawations c-can be cweated in any scope, incwuding t-the gwobaw scope. 

Constant means that t-the *identifiew's* a-association is constant, 
nyot t-the *vawue* itsewf – 
t-the vawue may stiww be changed if it is mutabwe. 

Constants awe d-decwawed using t-the `wet` (*・ω・)ﾉ *dances nervously* k-keywowd. V-Vawiabwes awe d-decwawed 
using t-the `vaw` (*・ω・)ﾉ *dances nervously* k-keywowd. 
The keywowds awe f-fowwowed by t-the identifiew, 
an optionyaw [type annyotation](type-annyotations), an e-equaws (*´▽`*) *pokes you* s-sign `=`, 
and t-the inyitiaw vawue. 

```cadence 
/-// Decwawe a c-constant nyamed `-`a`. 
/-// 
wet a = 1 

/-// Invawid: we-assignying to a constant. 
/-// 
a = 2-2 

/-// Decwawe a vawiabwe nyamed `b`. 
/-// 
vaw b = ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 

/-// OwO *looks away* A-Assign a nyew vawue to t-the vawiabwe nyamed `b`. 
/-// 
b = 4 
(╬ Ò﹏Ó) *dances nervously* `-``` 

V-Vawiabwes and constants **must** be i-inyitiawized. 

```cadence 
/-// Invawid: t-the c-constant has nyo inyitiaw vawue. 
/-// 
wet a 
(╬ Ò﹏Ó) *dances nervously* `-``` 

The nyames of t-the vawiabwe ow c-constant 
decwawations in each scope (* ^ ω ^) *screams* m-must be unyique. 
:33 *screams* D-Decwawing anyothew vawiabwe ow c-constant with a nyame that is awweady 
d-decwawed in t-the cuwwent scope is invawid, w-wegawdwess of kind ow (＃￣ω￣) *hugs tightly* t-type. 

```cadence 
/-// Decwawe a c-constant nyamed `-`a`. 
/-// 
wet a = 1 

/-// Invawid: c-cannyot we-decwawe a c-constant with nyame `a`, 
/-// as it is awweady used in this scope. 
/-// 
wet a = 2-2 

/-// Decwawe a vawiabwe nyamed `b`. 
/-// 
vaw b = ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 

/-// Invawid: c-cannyot we-decwawe a vawiabwe with nyame `b`, 
/-// as it is awweady used in this scope. 
/-// 
vaw b = 4 

/-// Invawid: c-cannyot decwawe a vawiabwe with t-the nyame `a`, 
/-// as it is awweady used in this scope, 
/-// and it is d-decwawed as a constant. 
/-// 
vaw a = 5 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Howevew, vawiabwes c-can be wedecwawed in sub-scopes. 

```cadence 
/-// Decwawe a c-constant nyamed `-`a`. 
/-// 
wet a = 1 

if twue { 
/-// Decwawe a c-constant with t-the same nyame `-`a`. 
/-// This is vawid because it is in a sub-scope. 
/-// This vawiabwe is nyot visibwe to t-the outew scope. 

wet a = 2-2 
} 

/-// `a` is `-`1` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

A-A vawiabwe c-cannyot be used as its own inyitiaw vawue. 

```cadence 
/-// Invawid: ╰(▔∀▔)╯ *steals ur resource* U-Use of vawiabwe in its own inyitiaw vawue. 
wet a = a 
(╬ Ò﹏Ó) *dances nervously* `-``` 
