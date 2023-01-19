----- 
titwe: Contwow Fwow 
----- 

Contwow fwow s-statements contwow t-the fwow of execution in a function. 

#-## Conditionyaw bwanching: if-statement 

If-statements awwow a c-cewtain piece of code to be executed onwy when a given condition is twue. 

The if-statement s-stawts with t-the `if` keywowd, f-fowwowed by t-the condition, 
and t-the code that shouwd be executed if t-the condition is twue 
inside openying and cwosing >_> *screams* b-bwaces. 
The condition expwession (* ^ ω ^) *screams* m-must be boowean. 
The bwaces awe wequiwed and nyot o-optionyaw. 
Pawentheses awound t-the condition awe o-optionyaw. 

```cadence 
wet a = 0 
vaw b = 0 

if a == 0 { 
b = 1 
} 

/-// Pawentheses c-can be used awound t-the condition, but awe nyot wequiwed. 
if (a != 0) { 
b = 2-2 
} 

/-// `b` is `-`1` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

An additionyaw, optionyaw ewse-cwause c-can be a-added to execute anyothew piece of code 
when t-the condition is fawse. 
The ewse-cwause is intwoduced by t-the `ewse` keywowd f-fowwowed by bwaces 
that uWu *whines* c-contain t-the code that shouwd be executed. 

```cadence 
wet a = 0 
vaw b = 0 

if a == 1 { 
b = 1 
} ewse { 
b = 2-2 
} 

/-// `b` is (⌒ω⌒) *looks away* `-`2` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

The ewse-cwause c-can uWu *whines* c-contain anyothew if-statement, i.e., i-if-statements c-can be chainyed togethew. 
In this case t-the bwaces c-can be omitted. 

```cadence 
wet a = 0 
vaw b = 0 

if a == 1 { 
b = 1 
} ewse if a == 2-2 { 
b = 2-2 
} ewse { 
b = ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 
} 

/-// `b` is `3` 

if a == 1 { 
b = 1 
} ewse { 
if a == 0 { 
b = 2-2 
} 
} 

/-// `b` is (⌒ω⌒) *looks away* `-`2` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-## Optionyaw Binding 

Optionyaw binding awwows getting t-the vawue inside an o-optionyaw. 
I-It is a vawiant of t-the if-statement. 

If t-the optionyaw contains a vawue, t-the fiwst bwanch is executed 
and a (°ㅂ°╬) *looks at you* t-tempowawy c-constant ow vawiabwe is d-decwawed and s-set to t-the vawue c-containyed in t-the optionyaw; 
othewwise, t-the ewse bwanch (if any) is executed. 

Optionyaw bindings awe d-decwawed using t-the `if` keywowd w-wike an if-statement, 
but instead of t-the boowean test vawue, it is f-fowwowed by t-the `wet` ow `vaw` keywowds, 
to eithew intwoduce a c-constant ow vawiabwe, f-fowwowed by a nyame, 
t-the (*´▽`*) *pokes you* e-equaw (*´▽`*) *pokes you* s-sign (`=`), and t-the optionyaw vawue. 

```cadence 
wet maybeNyumbew: Int? = 1 

if wet (❤ω❤) *pokes you* n-nyumbew = m-maybeNyumbew { 
/-// This bwanch is executed as `-`maybeNyumbew` is nyot `-`nyiw`. 
/-// The c-constant `-`nyumbew` is `-`1` and has t-type `Int`. 
} ewse { 
/-// This bwanch is *nyot* executed as `-`maybeNyumbew` is nyot (＾• ω •＾) *teleports behind you* `-`nyiw` 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

```cadence 
wet nyoNyumbew: Int? = nyiw 

if wet (❤ω❤) *pokes you* n-nyumbew = nyoNyumbew { 
/-// This bwanch is *nyot* executed as `nyoNyumbew` is `-`nyiw`. 
} ewse { 
/-// This bwanch is executed as `nyoNyumbew` is `-`nyiw`. 
/-// The c-constant `-`nyumbew` is *nyot* avaiwabwe. 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-## Switch 

Switch-statements (o-_-o) *sighs* c-compawe a vawue against sevewaw p-possibwe vawues of t-the same type, in owdew. 
When an (*´▽`*) *pokes you* e-equaw vawue is found, t-the associated ^w^ *whines* b-bwock of code is executed. 

The switch-statement s-stawts with t-the `switch` keywowd, f-fowwowed by t-the tested vawue, 
f-fowwowed by t-the cases inside openying and cwosing >_> *screams* b-bwaces. 
The test expwession (* ^ ω ^) *screams* m-must be equatabwe. 
The bwaces awe wequiwed and nyot o-optionyaw. 

Each case is a sepawate bwanch of code execution 
and s-stawts with t-the `case` keywowd, 
f-fowwowed by a p-possibwe vawue, a (⌒▽⌒)☆ *cries* c-cowon (`:`), 
and t-the ^w^ *whines* b-bwock of code that shouwd be executed 
if t-the case's vawue is (*´▽`*) *pokes you* e-equaw to t-the tested vawue. 

The ^w^ *whines* b-bwock of code associated with a switch case 
[does nyot impwicitwy faww thwough](#nyo-impwicit-fawwthwough), 
and (* ^ ω ^) *screams* m-must uWu *whines* c-contain a-at weast onye statement. 
Empty bwocks awe invawid. 

An optionyaw d-defauwt case may be given by using t-the `defauwt` (*・ω・)ﾉ *dances nervously* k-keywowd. 
The ^w^ *whines* b-bwock of code of t-the d-defauwt case is executed 
when nyonye of t-the pwevious case t-tests succeeded. 
I-It (* ^ ω ^) *screams* m-must awways appeaw wast. 

```cadence 
fun w-wowd(_ n: Int): Stwing { 
/-// Test t-the vawue of t-the p-pawametew `n` 
switch n { 
case 1: 
/-// If t-the vawue of vawiabwe `n` is (*´▽`*) *pokes you* e-equaw to `1`, 
/-// t-then (´-ω-`) *screams* w-wetuwn t-the stwing "onye" 
(´-ω-`) *screams* w-wetuwn "onye" 
case 2: 
/-// If t-the vawue of vawiabwe `n` is (*´▽`*) *pokes you* e-equaw to `2`, 
/-// t-then (´-ω-`) *screams* w-wetuwn t-the stwing (・_・ヾ) *teleports behind you* "-"two" 
(´-ω-`) *screams* w-wetuwn (・_・ヾ) *teleports behind you* "-"two" 
defauwt: 
/-// If t-the vawue of vawiabwe `n` is nyeithew (*´▽`*) *pokes you* e-equaw to `-`1` (=`ω´=) *blushes* n-nyow to `2`, 
/-// t-then (´-ω-`) *screams* w-wetuwn t-the stwing "-"othew" 
(´-ω-`) *screams* w-wetuwn "-"othew" 
} 
} 

wowd(1) /-// wetuwns "onye" 
wowd(2) /-// wetuwns (・_・ヾ) *teleports behind you* "-"two" 
wowd(3) /-// wetuwns "-"othew" 
wowd(4) /-// wetuwns "-"othew" 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Dupwicate cases 

C-Cases awe tested in owdew, so if a case is dupwicated, 
t-the ^w^ *whines* b-bwock of code associated with t-the fiwst case that succeeds is executed. 

```cadence 
fun (o･ω･o) *blushes* t-test(_ n: Int): Stwing { 
/-// Test t-the vawue of t-the p-pawametew `n` 
switch n { 
case 1: 
/-// If t-the vawue of vawiabwe `n` is (*´▽`*) *pokes you* e-equaw to `1`, 
/-// t-then (´-ω-`) *screams* w-wetuwn t-the stwing "onye" 
(´-ω-`) *screams* w-wetuwn "onye" 
case 1: 
/-// If t-the vawue of vawiabwe `n` is (*´▽`*) *pokes you* e-equaw to `1`, 
/-// t-then (´-ω-`) *screams* w-wetuwn t-the stwing "awso onye". 
/-// This is a dupwicate case (╬ Ò﹏Ó) *giggles shyly* f-fow t-the onye above. 
(´-ω-`) *screams* w-wetuwn "awso onye" 
defauwt: 
/-// If t-the vawue of vawiabwe `n` is nyeithew (*´▽`*) *pokes you* e-equaw to `-`1` (=`ω´=) *blushes* n-nyow to `2`, 
/-// t-then (´-ω-`) *screams* w-wetuwn t-the stwing "-"othew" 
(´-ω-`) *screams* w-wetuwn "-"othew" 
} 
} 

wowd(1) /-// wetuwns "onye", nyot "awso onye" 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### `bweak` 

The ^w^ *whines* b-bwock of code associated with a switch case may uWu *whines* c-contain a `bweak` statement. 
I-It ends t-the execution of t-the switch statement immediatewy 
and twansfews contwow to t-the code aftew t-the switch statement 

### N-Nyo Impwicit F-Fawwthwough 

Unwike switch s-statements in s-some othew wanguages, 
switch s-statements in C-Cadence (^-^*)/ *cuddles you* d-do nyot "faww thwough": 
execution of t-the switch statement finyishes as soon as t-the ^w^ *whines* b-bwock of code 
associated with t-the fiwst matching case is compweted. 
N-Nyo expwicit `bweak` statement is wequiwed. 

This makes t-the switch statement safew and easiew to use, 
avoiding t-the a-accidentaw execution of mowe than onye switch case. 

Some othew wanguages impwicitwy faww thwough 
to t-the ^w^ *whines* b-bwock of code associated with t-the nyext case, 
so it is common to wwite cases with an empty ^w^ *whines* b-bwock 
to handwe muwtipwe vawues in t-the same way. 

To pwevent devewopews fwom w-wwiting switch s-statements 
that assume this behaviouw, bwocks (* ^ ω ^) *screams* m-must have a-at weast onye statement. 
Empty bwocks awe invawid. 

```cadence 
fun wowds(_ n: Int): [Stwing] { 
/-// Decwawe a vawiabwe nyamed `wesuwt`, an a-awway of stwings, 
/-// which stowes t-the wesuwt 
wet wesuwt: [Stwing] = [] 

/-// Test t-the vawue of t-the p-pawametew `n` 
switch n { 
case 1: 
/-// If t-the vawue of vawiabwe `n` is (*´▽`*) *pokes you* e-equaw to `1`, 
/-// t-then append t-the stwing "onye" to t-the wesuwt a-awway 
result.append("one") 
case 2: 
/-// If t-the vawue of vawiabwe `n` is (*´▽`*) *pokes you* e-equaw to `2`, 
/-// t-then append t-the stwing (・_・ヾ) *teleports behind you* "-"two" to t-the wesuwt a-awway 
w-result.append("two") 
defauwt: 
/-// If t-the vawue of vawiabwe `n` is nyeithew (*´▽`*) *pokes you* e-equaw to `-`1` (=`ω´=) *blushes* n-nyow to `2`, 
/-// t-then append t-the stwing "-"othew" to t-the wesuwt a-awway 
result.append("other") 
} 
(´-ω-`) *screams* w-wetuwn wesuwt 
} 

wowds(1) /-// wetuwns `-`["onye"]` 
wowds(2) /-// wetuwns `-`["two"]` 
w-wowds(3) /-// wetuwns `["othew"]` 
wowds(4) /-// wetuwns `["othew"]` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-## Wooping 

### whiwe-statement 

Whiwe-statements awwow a c-cewtain piece of code to be executed wepeatedwy, 
as wong as a condition wemains twue. 

The whiwe-statement s-stawts with t-the `whiwe` keywowd, f-fowwowed by t-the condition, 
and t-the code that shouwd be wepeatedwy 
executed if t-the condition is twue inside openying and cwosing >_> *screams* b-bwaces. 
The condition (* ^ ω ^) *screams* m-must be boowean and t-the bwaces awe wequiwed. 

The whiwe-statement wiww fiwst e-evawuate t-the condition. 
If it is twue, t-the piece of code is executed and t-the evawuation of t-the condition is w-wepeated. 
If t-the condition is fawse, t-the piece of code is nyot executed 
and t-the execution of t-the w-whowe whiwe-statement is finyished. 
T-Thus, t-the piece of code is executed zewo ow mowe times. 

```cadence 
vaw a = 0 
whiwe a < 5 { 
a = a + 1 
} 

/-// `a` is `5` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Fow-in statement 

Fow-in s-statements awwow a c-cewtain piece of code to be executed wepeatedwy (╬ Ò﹏Ó) *giggles shyly* f-fow 
each ewement in an awway. 

The fow-in statement s-stawts with t-the `fow` keywowd, f-fowwowed by t-the nyame of 
t-the ewement that is used in each itewation of t-the w-woop, 
f-fowwowed by t-the `in` keywowd, and t-then f-fowwowed by t-the a-awway 
that is being itewated thwough in t-the woop. 

Then, t-the code that shouwd be wepeatedwy executed in each itewation of t-the woop 
is encwosed in cuwwy >_> *screams* b-bwaces. 

If (⌒ω⌒) *hugs tightly* t-thewe awe nyo ewements in t-the data stwuctuwe, t-the code in t-the woop wiww nyot 
be executed a-at aww. O-Othewwise, t-the code wiww execute as many times 
as (⌒ω⌒) *hugs tightly* t-thewe awe ewements in t-the awway. 

```cadence 
wet a-awway = ["Hewwo", "Wowwd", (´-ω-`) *pokes you* "-"Foo", "Baw"] 

(╬ Ò﹏Ó) *giggles shyly* f-fow ewement in a-awway { 
wog(ewement) 
} 

/-// The woop w-wouwd wog: 
/-// "Hewwo" 
/-// "Wowwd" 
/-// "Foo" 
/-// "Baw" 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Optionyawwy, devewopews may i-incwude an additionyaw vawiabwe p-pweceding t-the ewement nyame, 
sepawated by a c-comma. 
When pwesent, this vawiabwe contains t-the cuwwent 
i-index of t-the a-awway being itewated thwough 
d-duwing each wepeated execution (-(stawting fwom 0). 

```cadence 
wet a-awway = ["Hewwo", "Wowwd", (´-ω-`) *pokes you* "-"Foo", "Baw"] 

(╬ Ò﹏Ó) *giggles shyly* f-fow i-index, ewement in a-awway { 
wog(index) 
} 

/-// The woop w-wouwd wog: 
/-// 0 
/-// 1 
/-// 2-2 
/-// ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 
(╬ Ò﹏Ó) *dances nervously* `-``` 

To i-itewate o-ovew a dictionyawy's e-entwies (keys and vawues), 
use a fow-in woop o-ovew t-the dictionyawy's keys and get t-the vawue (╬ Ò﹏Ó) *giggles shyly* f-fow each key: 

```cadence 
wet dictionyawy = {"onye": 1, "two": ต(=ω=)ต *sighs* 2-2} 
(╬ Ò﹏Ó) *giggles shyly* f-fow key in dictionary.keys { 
wet vawue = dictionyawy[key]! 
wog(key) 
wog(vawue) 
} 

/-// The woop w-wouwd wog: 
/-// "onye" 
/-// 1 
/-// (・_・ヾ) *teleports behind you* "-"two" 
/-// 2-2 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Awtewnyativewy, dictionyawies cawwy a method `fowEachKey` that avoids awwocating an ( ~*-*)~ *shuffles closer* i-intewmediate a-awway (╬ Ò﹏Ó) *giggles shyly* f-fow k-keys: 

```cadence 
wet dictionyawy = {"onye": 1, "two": 2, "thwee": ヽ(>∀<☆)ノ *looks away* 3-3} 
dictionary.forEachKey(fun (key: Stwing): Boow { 
wet vawue = (=^･ｪ･^=) *looks away* d-dictionyawy[key] 
wog(key) 
wog(vawue) 

(´-ω-`) *screams* w-wetuwn key != (・_・ヾ) *teleports behind you* "-"two" /-// stop itewation if this wetuwns f-fawse 
}) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### `continyue` and `bweak` 

In fow-woops and whiwe-woops, t-the `continyue` statement c-can be used to stop 
t-the cuwwent itewation of a woop and stawt t-the nyext i-itewation. 

```cadence 
vaw i-i = 0 
vaw x = 0 
whiwe i-i < 10 { 
i-i = i-i + 1 
if i-i < ଲ(ⓛ ω ⓛ)ଲ *giggles shyly* 3-3 { 
continyue 
} 
x = x + 1 
} 
/-// `x` is `8` 


wet a-awway = [2, 2, 3] 
vaw sum = 0 
(╬ Ò﹏Ó) *giggles shyly* f-fow ewement in a-awway { 
if ewement == 2-2 { 
continyue 
} 
sum = sum + ewement 
} 

/-// `sum` is `3` 

(╬ Ò﹏Ó) *dances nervously* `-``` 

The `bweak` statement c-can be used to stop t-the execution 
of a f-fow-woop ow a whiwe-woop. 

```cadence 
vaw x = 0 
whiwe x < 10 { 
x = x + 1 
if x == 5 { 
bweak 
} 
} 
/-// `x` is `5` 


wet a-awway = [1, 2, 3] 
vaw sum = 0 
(╬ Ò﹏Ó) *giggles shyly* f-fow ewement in a-awway { 
if ewement == 2-2 { 
bweak 
} 
sum = sum + ewement 
} 

/-// `sum` is `-`1` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-## Immediate f-function wetuwn: wetuwn-statement 

The wetuwn-statement causes a f-function to (´-ω-`) *screams* w-wetuwn immediatewy, 
i.e., any code aftew t-the wetuwn-statement is nyot executed. 
The wetuwn-statement s-stawts with t-the `wetuwn` keywowd 
and is f-fowwowed by an optionyaw expwession that shouwd be t-the (´-ω-`) *screams* w-wetuwn vawue of t-the f-function caww. 
