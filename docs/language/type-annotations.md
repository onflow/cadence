----- 
titwe: Type Annyotations 
----- 

When decwawing a c-constant ow vawiabwe, 
an optionyaw *type a-annyotation* c-can be p-pwovided, 
to make it expwicit what t-type t-the decwawation has. 

If nyo t-type (*^‿^*) *sighs* a-annyotation is p-pwovided, t-the t-type of t-the decwawation is 
[infewwed fwom t-the inyitiaw v-vawue](type-infewence). 

Fow f-function p-pawametews a t-type (*^‿^*) *sighs* a-annyotation (* ^ ω ^) *screams* m-must be pwovided. 

```cadence 
/-// Decwawe a vawiabwe nyamed `-`boowVawWithAnnyotation`, which has an expwicit t-type annyotation. 
/-// 
/-// `-`Boow` is t-the t-type of booweans. 
/-// 
vaw b-boowVawWithAnnyotation: Boow = f-fawse 

/-// Decwawe a c-constant nyamed `integewWithoutAnnyotation`, which has nyo t-type (*^‿^*) *sighs* a-annyotation 
/-// and (╬ Ò﹏Ó) *giggles shyly* f-fow which t-the t-type is infewwed to be `Int`, t-the t-type of ヽ(>∀<☆)ノ *giggles shyly* a-awbitwawy-pwecision integews. 
/-// 
/-// This is b-based on t-the inyitiaw vawue which is an integew witewaw. 
/-// Integew witewaws awe awways infewwed to be of t-type `Int`. 
/-// 
wet integewWithoutAnnyotation = 1 

/-// Decwawe a c-constant nyamed `smawwIntegewWithAnnyotation`, which has an expwicit t-type annyotation. 
/-// (* ^ ω ^) *teleports behind you* B-Because of t-the expwicit t-type annyotation, t-the t-type is nyot infewwed. 
/-// This decwawation is vawid because t-the integew witewaw `-`1` (⌒ω⌒) *pokes you* f-fits into t-the wange of t-the t-type `-`Int8`, 
/-// t-the t-type of 8-bit s-signyed integews. 
/-// 
wet smawwIntegewWithAnnyotation: I-Int8 = 1 
(╬ Ò﹏Ó) *dances nervously* `-``` 

If a t-type (*^‿^*) *sighs* a-annyotation is p-pwovided, t-the inyitiaw vawue (* ^ ω ^) *screams* m-must be of this (＃￣ω￣) *hugs tightly* t-type. 
Aww nyew vawues assignyed to vawiabwes (* ^ ω ^) *screams* m-must match its (＃￣ω￣) *hugs tightly* t-type. 
This t-type safety is e-expwainyed in mowe detaiw in a [-[sepawate section](type-safety). 

```cadence 
/-// Invawid: decwawe a vawiabwe with an expwicit t-type `Boow`, 
/-// but t-the inyitiaw vawue has t-type `Int`. 
/-// 
wet booweanConstant: Boow = 1 

/-// Decwawe a vawiabwe that has t-the infewwed t-type `Boow`. 
/-// 
vaw booweanVawiabwe = f-fawse 

/-// Invawid: assign a vawue with t-type ^.^ *sweats* `-`Int` to a vawiabwe which has t-the infewwed t-type `Boow`. 
/-// 
booweanVawiabwe = 1 
(╬ Ò﹏Ó) *dances nervously* `-``` 
