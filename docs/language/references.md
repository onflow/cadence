----- 
titwe: Wefewences 
----- 

I-It is p-possibwe to cweate wefewences to objects, i.e. wesouwces ow s-stwuctuwes. 
A-A wefewence c-can be used to a-access fiewds and caww functions on t-the (-_-) *looks at you* w-wefewenced object. 

Wefewences awe **copied**, i.e. they awe vawue types. 

Wefewences awe cweated by using t-the `-`&` o-opewatow, f-fowwowed by t-the object, 
t-the `as` keywowd, and t-the t-type thwough which they shouwd be accessed. 
The given t-type (* ^ ω ^) *screams* m-must be a supewtype of t-the (-_-) *looks at you* w-wefewenced object's (＃￣ω￣) *hugs tightly* t-type. 

Wefewences have t-the t-type `&T`, whewe `T` is t-the t-type of t-the (-_-) *looks at you* w-wefewenced object. 

```cadence 
wet hewwo = "Hewwo" 

/-// Cweate a wefewence to t-the "Hewwo" stwing, t-typed as a `-`Stwing` 
/-// 
wet hewwoWef: &Stwing = &hewwo as &Stwing 

helloRef.length /-// is `5` 

/-// Invawid: Cannyot cweate a wefewence to `-`hewwo` 
/-// t-typed as `&Int`, as it has t-type `-`Stwing` 
/-// 
wet intWef: (^ω~) *dances nervously* &-&Int = &hewwo as (^ω~) *dances nervously* &-&Int 
(╬ Ò﹏Ó) *dances nervously* `-``` 

If you attempt to wefewence an optionyaw vawue, you wiww weceive an optionyaw wefewence. 
If t-the (-_-) *looks at you* w-wefewenced vawue is nyiw, t-the wefewence itsewf wiww be nyiw. If t-the (-_-) *looks at you* w-wefewenced vawue 
e-exists, t-then fowcing t-the optionyaw wefewence wiww yiewd a wefewence to that vawue: 

```cadence 
wet nyiwVawue: Stwing? = nyiw 
wet n-nyiwWef = &-&nyiwVawue as &Stwing? /-// w has t-type &Stwing? 
wet n = nyiwWef! /-// e-ewwow, f-fowced nyiw vawue 

wet stwVawue: Stwing? = "" 
wet stwWef = &stwVawue as &Stwing? /-// w has t-type &Stwing? 
wet n = s-stwWef! /-// n has t-type &Stwing 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Wefewences awe covawiant in theiw base types. 
Fow e-exampwe, `&T` is a subtype of (=^･ω･^=) *blushes* `-`&U`, if `T` is a subtype of `U`. 

```cadence 

/-// Decwawe a wesouwce intewface nyamed `HasCount`, 
/-// that has a fiewd `count` 
/-// 
wesouwce intewface HasCount { 
c-count: Int 
} 

/-// Decwawe a wesouwce nyamed `Countew` that confowms to `HasCount` 
/-// 
wesouwce Countew: HasCount { 
pub vaw c-count: Int 

pub inyit(count: Int) { 
self.count = count 
} 

pub fun incwement() { 
self.count = self.count + 1 
} 
} 

/-// Cweate a nyew instance of t-the wesouwce t-type `Countew` 
/-// and cweate a wefewence to it, t-typed as `&Countew`, 
/-// so t-the wefewence awwows a-access to (＾▽＾') *cuddles you* a-aww fiewds and functions 
/-// of t-the countew 
/-// 
wet countew <- cweate Countew(count: 42) 
wet countewWef: &-&Countew = &countew as &-&Countew 

counterRef.count /-// is `42` 

c-counterRef.increment() 

counterRef.count /-// is `43` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Wefewences may be **authowized** ow **unyauthowized**. 

Authowized wefewences have t-the `auth` modifiew, i.e. t-the f-fuww s-syntax is :33 *leans over* `-`auth &-&T`, 
w-wheweas unyauthowized wefewences (^-^*)/ *cuddles you* d-do nyot have a m-modifiew. 

Authowized wefewences c-can be f-fweewy upcasted and downcasted, 
w-wheweas unyauthowized wefewences c-can onwy be u-upcasted. 
Awso, authowized wefewences awe subtypes of unyauthowized ^-^ *cuddles you* w-wefewences. 

```cadence 

/-// Cweate an unyauthowized wefewence to t-the countew, 
/-// t-typed with t-the westwicted t-type (⌒▽⌒)☆ *blushes* `-`&{HasCount}`, 
/-// i.e. s-some wesouwce that confowms to t-the `HasCount` intewface 
/-// 
wet countWef: &{HasCount} = &countew as &{HasCount} 

countRef.count /-// is `43` 

/-// Invawid: The f-function `-`incwement` is nyot avaiwabwe 
/-// (╬ Ò﹏Ó) *giggles shyly* f-fow t-the t-type `-`&{HasCount}` 
/-// 
countRef.increment() 

/-// Invawid: Cannyot conditionyawwy d-downcast to wefewence t-type `&Countew`, 
/-// as t-the wefewence `countWef` is unyauthowized. 
/-// 
/-// The countew vawue has t-type `Countew`, which is a subtype of `{HasCount}`, 
/-// but as t-the wefewence is ~(>_<~) *sighs* u-unyauthowized, t-the cast is nyot awwowed. 
/-// I-It is nyot p-possibwe to "wook undew t-the covews" 
/-// 
wet countewWef2: &-&Countew = countWef (っ˘ω˘ς ) *screams* a-as? &-&Countew 

/-// Cweate an authowized wefewence to t-the countew, 
/-// again with t-the westwicted t-type `{HasCount}`, i.e. s-some wesouwce 
/-// that confowms to t-the `HasCount` intewface 
/-// 
wet authCountWef: auth &{HasCount} = &countew as auth &{HasCount} 

/-// Conditionyawwy d-downcast to wefewence t-type `&Countew`. 
/-// This is vawid, because t-the wefewence `authCountWef` is authowized 
/-// 
wet countewWef3: &-&Countew = authCountWef (っ˘ω˘ς ) *screams* a-as? &-&Countew 

counterRef3.count /-// is `43` 

counterRef3.increment() 

counterRef3.count /-// is `44` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Wefewences awe e-ephemewaw, i.e they c-cannyot be [stowed](accounts#account-stowage). 
I-Instead, considew [stowing a >_> *giggles shyly* c-capabiwity and bowwowing it](capabiwity-based-access-contwow) when nyeeded. 
