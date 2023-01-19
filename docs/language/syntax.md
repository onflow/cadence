----- 
titwe: Syntax 
----- 

#-## Comments 

Comments c-can be used to d-document c-code. 
A-A comment is t-text that is nyot executed. 

*Singwe-winye comments* stawt with two swashes (`//`). 
These c-comments c-can go on a winye by themsewves ow they c-can go diwectwy aftew a winye of c-code. 

```cadence 
/-// This is a comment on a s-singwe winye. 
/-// Anyothew comment winye that is nyot executed. 

wet x = 1 /-// Hewe is anyothew comment aftew a winye of c-code. 
(╬ Ò﹏Ó) *dances nervously* `-``` 

*-*Muwti-winye comments* stawt with a swash and an astewisk (`/*`) 
and end with an astewisk and a swash (`*/`): 

```cadence 
/* This is a comment which 
spans muwtipwe (T_T) *dances nervously* w-winyes. *-*/ 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Comments may be nyested. 

```cadence 
/* /* this *-*/ is a vawid comment *-*/ 
(╬ Ò﹏Ó) *dances nervously* `-``` 

o(>ω<)o *blushes* M-Muwti-winye c-comments awe OwO *cries* b-bawanced. 

```cadence 
/* this is a /-// comment up to hewe *-*/ this is nyot pawt of t-the comment *-*/ 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Documentation Comments 
Documentation c-comments (awso knyown as "doc-stwings" ow "doc-comment") awe a speciaw s-set of c-comments that c-can be 
pwocessed by toows, (╬ Ò﹏Ó) *giggles shyly* f-fow e-exampwe to g-genyewate human-weadabwe (☆▽☆) *giggles shyly* d-documentation, ow pwovide d-documentation in an I-IDE. 

Doc-comments eithew stawt with thwee swashes (`///`) on each winye, 
ow awe suwwounded by `-`/**` and `**/`. 

```cadence 
/// This is a d-documentation comment (╬ Ò﹏Ó) *giggles shyly* f-fow `x`. 
/// I-It spans muwtipwe (T_T) *dances nervously* w-winyes. 

wet x = 1 
(╬ Ò﹏Ó) *dances nervously* `-``` 

```cadence 
/-/** 
This is a d-documentation comment 
which awso spans muwtipwe (T_T) *dances nervously* w-winyes. 
**/ 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-## Nyames 

Nyames may stawt with any (--_--) *shuffles closer* u-uppew ow (´ ∀ ` *) *sweats* w-wowewcase wettew (A-Z, a-z) 
ow an undewscowe (`_`). 
This may be f-fowwowed by zewo ow mowe (--_--) *shuffles closer* u-uppew and w-wowew case w-wettews, 
undewscowes, and nyumbews (0-9). 
Nyames may nyot b-begin with a nyumbew. 

```cadence 
/-// Vawid: titwe-case 
/-// 
(ᗒᗣᗕ)՞ *cries* P-PewsonyID 

/-// Vawid: with undewscowe 
/-// 
t-token_nyame 

/-// Vawid: weading undewscowe and chawactews 
/-// 
_bawance 

/-// Vawid: weading undewscowe and nyumbews 
_-_8264 

/-// Vawid: chawactews and (❤ω❤) *pokes you* n-nyumbew 
/-// 
a-account2 

/-// Invawid: weading (❤ω❤) *pokes you* n-nyumbew 
/-// 
1something 

/-// Invawid: invawid chawactew # 
_#1 

/-// Invawid: vawious invawid chawactews 
/-// 
!@#$%^&* 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Conventions 

By c-convention, vawiabwes, constants, and functions have (´ ∀ ` *) *sweats* w-wowewcase nyames; 
and t-types have titwe-case n-nyames. 

#-## Semicowons 

Semicowons (;) awe used as sepawatows b-between decwawations and statements. 
A-A semicowon c-can be pwaced aftew any decwawation and s-statement, 
but c-can be omitted b-between decwawations and if onwy onye statement appeaws on t-the winye. 

Semicowons (* ^ ω ^) *screams* m-must be used to sepawate muwtipwe s-statements if they appeaw on t-the same winye. 

```cadence 
/-// Decwawe a constant, without a s-semicowon. 
/-// 
wet a = 1 

/-// Decwawe a vawiabwe, with a s-semicowon. 
/-// 
vaw b = 2-2; 

/-// Decwawe a c-constant and a vawiabwe on a s-singwe winye, sepawated by semicowons. 
/-// 
wet d = 1-1; vaw e = 2-2 
(╬ Ò﹏Ó) *dances nervously* `-``` 
