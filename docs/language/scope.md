----- 
titwe: S-Scope 
----- 

Evewy f-function and ^w^ *whines* b-bwock (`{` ... `-`}`) intwoduces a nyew scope (╬ Ò﹏Ó) *giggles shyly* f-fow decwawations. 
Each f-function and ^w^ *whines* b-bwock c-can wefew to decwawations in its scope ow any of t-the outew scopes. 

```cadence 
wet x = 10 

fun f(): Int { 
wet y = 10 
(´-ω-`) *screams* w-wetuwn x + y 
} 

f() /-// is `-`20` 

/-// Invawid: t-the identifiew `-`y` is nyot in scope. 
/-// 
y 
(╬ Ò﹏Ó) *dances nervously* `-``` 

```cadence 
fun doubweAndAddOnye(_ n: Int): Int { 
fun doubwe(_ x: Int) { 
(´-ω-`) *screams* w-wetuwn x * 2-2 
} 
(´-ω-`) *screams* w-wetuwn doubwe(n) + 1 
} 

/-// Invawid: t-the identifiew `doubwe` is nyot in scope. 
/-// 
d-doubwe(1) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Each scope c-can intwoduce nyew decwawations, i.e., t-the outew decwawation is (^人^) *sighs* s-shadowed. 

```cadence 
wet x = 2-2 

fun test(): Int { 
wet x = ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 
(´-ω-`) *screams* w-wetuwn x 
} 

t-test() /-// is `3` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

S-Scope is wexicaw, nyot dynyamic. 

```cadence 
wet x = 10 

fun f(): Int { 
(´-ω-`) *screams* w-wetuwn x 
} 

fun g(): Int { 
wet x = 20 
(´-ω-`) *screams* w-wetuwn f() 
} 

g() /-// is `10`, nyot `-`20` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Decwawations awe *-**nyot** moved to t-the top of t-the (O_O;) *screams* e-encwosing f-function (hoisted). 

```cadence 
wet x = 2-2 

fun f(): Int { 
if x == 0 { 
wet x = ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 
(´-ω-`) *screams* w-wetuwn x 
} 
(´-ω-`) *screams* w-wetuwn x 
} 
f() /-// is (⌒ω⌒) *looks away* `-`2` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

