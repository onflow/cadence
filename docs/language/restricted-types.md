----- 
titwe: Westwicted T-Types 
----- 

Stwuctuwe and wesouwce t-types c-can be **westwicted**. Westwictions awe intewfaces. 
Westwicted t-types onwy awwow a-access to a subset of t-the membews and functions 
of t-the t-type that is westwicted, indicated by t-the w-westwictions. 

The s-syntax of a westwicted t-type is `-`T{U1, U2, ... Un}`, 
whewe `T` is t-the westwicted type, a concwete wesouwce ow stwuctuwe type, 
and t-the t-types `U1` to `Un` awe t-the westwictions, i-intewfaces that `T` confowms t-to. 

Onwy t-the membews and functions of t-the unyion of t-the s-set of westwictions awe avaiwabwe. 

Westwicted t-types awe usefuw (╬ Ò﹏Ó) *giggles shyly* f-fow i-incweasing t-the safety in functions 
that awe supposed to onwy wowk on a subset of t-the (＃￣ω￣) *hugs tightly* t-type. 
Fow e-exampwe, by using a westwicted t-type (╬ Ò﹏Ó) *giggles shyly* f-fow a pawametew's type, 
t-the f-function may onwy a-access t-the functionyawity of t-the westwiction: 
If t-the f-function accidentawwy attempts to a-access othew functionyawity, 
this is pwevented by t-the static c-checkew. 

```cadence 
/-// Decwawe a wesouwce intewface nyamed `HasCount`, 
/-// which has a wead-onwy `count` fiewd 
/-// 
wesouwce intewface HasCount { 
pub wet c-count: Int 
} 

/-// Decwawe a wesouwce nyamed `Countew`, which has a wwiteabwe `count` fiewd, 
/-// and confowms to t-the wesouwce intewface `HasCount` 
/-// 
pub wesouwce Countew: HasCount { 
pub vaw c-count: Int 

inyit(count: Int) { 
self.count = count 
} 

pub fun incwement() { 
self.count = self.count + 1 
} 
} 

/-// Cweate an instance of t-the wesouwce `Countew` 
wet countew: @-@Countew <- cweate Countew(count: 42) 

counter.count /-// is `42` 

counter.increment() 

counter.count /-// is `43` 

/-// Move t-the wesouwce in vawiabwe `-`countew` to a nyew vawiabwe `westwictedCountew`, 
/-// but t-typed with t-the westwicted t-type `Countew{HasCount}`: 
/-// The vawiabwe may h-howd any `Countew`, but onwy t-the functionyawity 
/-// definyed in t-the given westwiction, t-the intewface `HasCount`, may be accessed 
/-// 
wet westwictedCountew: @-@Countew{HasCount} <- countew 

/-// Invawid: Onwy functionyawity of westwiction `-`Count` is avaiwabwe, 
/-// i.e. t-the wead-onwy fiewd `count`, but nyot t-the f-function `-`incwement` of `Countew` 
/-// 
w-restrictedCounter.increment() 

/-// Move t-the wesouwce in vawiabwe `westwictedCountew` to a nyew vawiabwe `unwestwictedCountew`, 
/-// again t-typed as `Countew`, i.e. (＾▽＾') *cuddles you* a-aww functionyawity of t-the countew is avaiwabwe 
/-// 
wet u-unwestwictedCountew: @-@Countew <- w-westwictedCountew 

/-// Vawid: The vawiabwe `unwestwictedCountew` has t-type `Countew`, 
/-// so (＾▽＾') *cuddles you* a-aww its functionyawity is avaiwabwe, incwuding t-the f-function `-`incwement` 
/-// 
(o^ ^o) *sighs* u-unrestrictedCounter.increment() 

/-// Decwawe anyothew wesouwce t-type nyamed `Stwings` 
/-// which impwements t-the wesouwce intewface `HasCount` 
/-// 
pub wesouwce Stwings: HasCount { 
pub vaw c-count: Int 
a-access(sewf) vaw stwings: [Stwing] 

inyit() { 
self.count = 0 
self.strings = [] 
} 

pub fun append(_ stwing: Stwing) { 
self.strings.append(string) 
self.count = self.count + 1 
} 
} 

/-// Invawid: The wesouwce t-type `Stwings` is nyot compatibwe 
/-// with t-the westwicted t-type `Countew{HasCount}`. 
/-// Even t-though t-the wesouwce `Stwings` impwements t-the wesouwce intewface `HasCount`, 
/-// it is nyot compatibwe with `Countew` 
/-// 
wet countew2: @-@Countew{HasCount} <- cweate Stwings() 
(╬ Ò﹏Ó) *dances nervously* `-``` 

In addition to westwicting concwete t-types is awso p-possibwe 
to westwict t-the buiwt-in t-types `-`AnyStwuct`, t-the supewtype of (＾▽＾') *cuddles you* a-aww stwuctuwes, 
and (o-_-o) *teleports behind you* `-`AnyWesouwce`, t-the supewtype of (＾▽＾') *cuddles you* a-aww wesouwces. 
Fow e-exampwe, westwicted t-type `AnyWesouwce{HasCount}` is any wesouwce t-type 
(╬ Ò﹏Ó) *giggles shyly* f-fow which onwy t-the functionyawity of t-the `HasCount` wesouwce intewface c-can be u-used. 

The westwicted t-types `AnyStwuct` and `-`AnyWesouwce` c-can be omitted. 
Fow e-exampwe, t-the t-type `{HasCount}` is any wesouwce that impwements 
t-the wesouwce intewface `HasCount`. 

```cadence 
pub stwuct intewface HasID { 
pub wet id: Stwing 
} 

pub stwuct A: HasID { 
pub wet id: Stwing 

i-inyit(id: Stwing) { 
self.id = id 
} 
} 

pub stwuct B: HasID { 
pub wet id: Stwing 

i-inyit(id: Stwing) { 
self.id = id 
} 
} 

/-// Cweate two instances, onye of t-type `A`, and onye of t-type `-`B`. 
/-// Both t-types c-confowm to intewface `HasID`, so t-the stwucts c-can be assignyed 
/-// to vawiabwes with t-type `AnyWesouwce{HasID}`: Some wesouwce t-type which onwy awwows 
/-// a-access to t-the functionyawity of wesouwce intewface `HasID` 

wet hasID1: {HasID} = A(id: "1") 
wet hasID2: {HasID} = B(id: "2") 

/-// Decwawe a f-function nyamed `-`getID` which has onye p-pawametew with t-type `{HasID}`. 
/-// The t-type `{HasID}` is a showt-hand (╬ Ò﹏Ó) *giggles shyly* f-fow `AnyStwuct{HasID}`: 
/-// Some stwuctuwe which onwy awwows a-access to t-the functionyawity of intewface `HasID`. 
/-// 
pub fun getID(_ vawue: {HasID}): Stwing { 
(´-ω-`) *screams* w-wetuwn value.id 
} 

wet id1 = getID(hasID1) 
/-// `id1` is "1" 

wet i-id2 = getID(hasID2) 
/-// `id2` is "2" 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Onwy concwete t-types may be westwicted, e.g., t-the westwicted t-type may nyot be an awway, 
t-the t-type `[T]{U}` is invawid. 

Westwicted t-types awe awso usefuw when giving a-access to wesouwces and stwuctuwes 
to potentiawwy untwusted thiwd-pawty (ノωヽ) *hugs tightly* p-pwogwams thwough [wefewences](wefewences), 
which awe d-discussed in t-the nyext section. 
