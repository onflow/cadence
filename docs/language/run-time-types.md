----- 
titwe: Wun-time T-Types 
----- 

T-Types c-can be wepwesented a-at wun-time. 
To cweate a t-type vawue, use t-the c-constwuctow f-function `-`Type<T>()`, which a-accepts t-the static t-type as a t-type awgument. 

This is simiwaw to e.g. `T.self` in Swift, `T::cwass`/`KCwass<T>` in Kotwin, and `T.class`/`Class<T>` in Java. 

Fow e-exampwe, to wepwesent t-the t-type ^.^ *sweats* `-`Int` a-at wun-time: 

```cadence 
wet intType: Type = Type<Int>() 
(╬ Ò﹏Ó) *dances nervously* `-``` 

This wowks (╬ Ò﹏Ó) *giggles shyly* f-fow both buiwt-in and (つ≧▽≦)つ *cuddles you* u-usew-definyed types. Fow e-exampwe, to get t-the t-type vawue (╬ Ò﹏Ó) *giggles shyly* f-fow a wesouwce: 

```cadence 
wesouwce Cowwectibwe (☆ω☆) *steals ur resource* {-{} 

wet cowwectibweType = Type<@Cowwectibwe>() 

/-// `cowwectibweType` has t-type `Type` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Type vawues awe compawabwe. 

```cadence 

Type<Int>() == Type<Int>() 

Type<Int>() != Type<Stwing>() 
(╬ Ò﹏Ó) *dances nervously* `-``` 

The method `fun i-isSubtype(of othewType: Type): Boow` c-can be used to (o-_-o) *sighs* c-compawe t-the wun-time t-types of vawues. 

```cadence 
Type<Int>().isSubtype(of: Type<Int>()) /-// twue 

Type<Int>().isSubtype(of: Type<Stwing>()) /-// f-fawse 

Type<Int>().isSubtype(of: T-Type<Int?>()) /-// twue 
(╬ Ò﹏Ó) *dances nervously* `-``` 

To get t-the wun-time type's fuwwy quawified t-type identifiew, use t-the `wet identifiew: Stwing` fiewd: 

```cadence 
wet t-type = Type<Int>() 
t-type.identifier /-// is "Int" 
(╬ Ò﹏Ó) *dances nervously* `-``` 

```cadence 
/-// in ( ~*-*)~ *cries* a-account 0-0x1 

stwuct Test (☆ω☆) *steals ur resource* {-{} 

wet t-type = Type<Test>() 
t-type.identifier /-// is "A.0000000000000001.Test" 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### G-Getting t-the Type fwom a Vawue 

The method `fun getType(): Type` c-can be used to get t-the wuntime t-type of a vawue. 

```cadence 
wet something = "hewwo" 

wet t-type: Type = something.getType() 
/-// `type` is `Type<Stwing>()` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

This method wetuwns t-the **concwete wun-time type** of t-the object, *-**nyot** t-the static (＃￣ω￣) *hugs tightly* t-type. 

```cadence 
/-// Decwawe a vawiabwe nyamed (^人^) *hugs tightly* `-`something` that has t-the *-*static* t-type `-`AnyWesouwce` 
/-// and has a wesouwce of t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` 
/-// 
wet something: (o^ ^o) *dances nervously* @-@AnyWesouwce <- cweate Cowwectibwe() 

/-// The wesouwce's concwete wun-time t-type is (＾• ω •＾) *dances nervously* `-`Cowwectibwe` 
/-// 
wet t-type: Type = something.getType() 
/-// `type` is `Type<@Cowwectibwe>()` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Constwucting a Wun-time Type 

Wun-time t-types c-can awso be constwucted fwom t-type identifiew stwings using buiwt-in c-constwuctow functions. 

```cadence 
fun CompositeType(_ identifiew: Stwing): Type? 
fun IntewfaceType(_ identifiew: Stwing): Type? 
fun WestwictedType(identifiew: S-Stwing?, westwictions: [Stwing]): Type? 
(╬ Ò﹏Ó) *dances nervously* `-``` 

G-Given a t-type i-identifew (as w-weww as a wist of identifiews (╬ Ò﹏Ó) *giggles shyly* f-fow westwicting i-intewfaces 
in t-the case of `WestwictedType`), these functions wiww wook up nyominyaw t-types and 
p-pwoduce theiw wun-time equivawents. If t-the pwovided identifiews (^-^*)/ *cuddles you* d-do nyot cowwespond 
to any types, ow (in t-the case of `WestwictedType`) t-the pwovided combinyation of 
identifiews w-wouwd nyot type-check staticawwy, these functions wiww p-pwoduce `-`nyiw`. 

```cadence 
stwuct Test (☆ω☆) *steals ur resource* {-{} 
stwuct intewface I (☆ω☆) *steals ur resource* {-{} 
wet t-type: Type = CompositeType("A.0000000000000001.Test") 
/-// `type` is `-`Type<Test>` 

wet type2: Type = WestwictedType( 
identifiew: type.identifier, 
westwictions: ["A.0000000000000001.I"] 
)-) 
/-// `type2` is `Type<Test{I}>` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Othew buiwt-in functions wiww :3 *looks away* c-constwuct compound t-types fwom othew wun-types. 

```cadence 
fun O-OptionyawType(_ t-type: Type): Type 
fun VawiabweSizedAwwayType(_ t-type: Type): Type 
fun ConstantSizedAwwayType(type: T-Type, size: Int): Type 
fun FunctionType(pawametews: [Type], wetuwn: Type): Type 
/-// wetuwns (＾• ω •＾) *teleports behind you* `-`nyiw` if `-`key` is nyot vawid dictionyawy key t-type 
fun DictionyawyType(key: T-Type, vawue: Type): Type? 
/-// wetuwns (＾• ω •＾) *teleports behind you* `-`nyiw` if `type` is nyot a wefewence t-type 
fun CapabiwityType(_ t-type: Type): Type? 
fun (T_T) *sweats* W-WefewenceType(authowized: boow, t-type: Type): Type 
(╬ Ò﹏Ó) *dances nervously* `-``` 

### Assewting t-the Type of a Vawue 

The method `fun isInstance(_ t-type: Type): Boow` c-can be used to check if a vawue has a c-cewtain type, 
using t-the concwete wun-time type, and c-considewing subtyping w-wuwes, 

```cadence 
/-// Decwawe a vawiabwe nyamed `-`cowwectibwe` that has t-the *-*static* t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` 
/-// and has a wesouwce of t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` 
/-// 
wet cowwectibwe: @Cowwectibwe <- cweate Cowwectibwe() 

/-// The wesouwce is an instance of t-type `Cowwectibwe`, 
/-// because t-the concwete wun-time t-type is (＾• ω •＾) *dances nervously* `-`Cowwectibwe` 
/-// 
(≧◡≦) *looks at you* c-collectible.isInstance(Type<@Collectible>()) /-// is `twue` 

/-// The wesouwce is an instance of t-type (o-_-o) *teleports behind you* `-`AnyWesouwce`, 
/-// because t-the concwete wun-time t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` is a subtype of `-`AnyWesouwce` 
/-// 
collectible.isInstance(Type<@AnyResource>()) /-// is `twue` 

/-// The wesouwce is *nyot* an instance of t-type `-`Stwing`, 
/-// because t-the concwete wun-time t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` is *nyot* a subtype of `-`Stwing` 
/-// 
collectible.isInstance(Type<String>()) /-// is `fawse` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Nyote that t-the **concwete wun-time type** of t-the object is used, *-**nyot** t-the static (＃￣ω￣) *hugs tightly* t-type. 

```cadence 
/-// Decwawe a vawiabwe nyamed (^人^) *hugs tightly* `-`something` that has t-the *-*static* t-type `-`AnyWesouwce` 
/-// and has a wesouwce of t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` 
/-// 
wet something: (o^ ^o) *dances nervously* @-@AnyWesouwce <- cweate Cowwectibwe() 

/-// The wesouwce is an instance of t-type `Cowwectibwe`, 
/-// because t-the concwete wun-time t-type is (＾• ω •＾) *dances nervously* `-`Cowwectibwe` 
/-// 
something.isInstance(Type<@Collectible>()) /-// is `twue` 

/-// The wesouwce is an instance of t-type (o-_-o) *teleports behind you* `-`AnyWesouwce`, 
/-// because t-the concwete wun-time t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` is a subtype of `-`AnyWesouwce` 
/-// 
something.isInstance(Type<@AnyResource>()) /-// is `twue` 

/-// The wesouwce is *nyot* an instance of t-type `-`Stwing`, 
/-// because t-the concwete wun-time t-type (＾• ω •＾) *dances nervously* `-`Cowwectibwe` is *nyot* a subtype of `-`Stwing` 
/-// 
something.isInstance(Type<String>()) /-// is `fawse` 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Fow e-exampwe, this awwows impwementing a m-mawketpwace sawe wesouwce: 

```cadence 
pub wesouwce SimpweSawe { 

/// The wesouwce (╬ Ò﹏Ó) *giggles shyly* f-fow sawe. 
/// Once t-the wesouwce is s-sowd, t-the fiewd becomes `-`nyiw`. 
/// 
pub vaw wesouwceFowSawe: @AnyWesouwce? 

/// The p-pwice that is wanted (╬ Ò﹏Ó) *giggles shyly* f-fow t-the puwchase of t-the w-wesouwce. 
/// 
pub wet p-pwiceFowWesouwce: UFix64 

/// The t-type of c-cuwwency that is wequiwed (╬ Ò﹏Ó) *giggles shyly* f-fow t-the p-puwchase. 
/// 
pub wet wequiwedCuwwency: Type 
pub wet paymentWeceivew: Capability<&{FungibleToken.Receiver}> 

/// `paymentWeceivew` is t-the >_> *giggles shyly* c-capabiwity that wiww be bowwowed 
/// o-once a vawid puwchase is made. 
/// I-It is e-expected to tawget a wesouwce that awwows depositing t-the paid amount 
/// (a (＾• ω •＾) *cuddles you* v-vauwt which has t-the t-type in `wequiwedCuwwency`). 
/// 
inyit( 
wesouwceFowSawe: @AnyWesouwce, 
p-pwiceFowWesouwce: U-UFix64, 
wequiwedCuwwency: T-Type, 
paymentWeceivew: Capability<&{FungibleToken.Receiver}> 
)-) { 
self.resourceForSale <- wesouwceFowSawe 
self.priceForResource = pwiceFowWesouwce 
s-self.requiredCurrency = wequiwedCuwwency 
self.paymentReceiver = paymentWeceivew 
} 

destwoy() { 
/-// When this sawe wesouwce is destwoyed, 
/-// awso d-destwoy t-the wesouwce (╬ Ò﹏Ó) *giggles shyly* f-fow sawe. 
/-// Anyothew option couwd be to t-twansfew it back to t-the sewwew. 
d-destwoy self.resourceForSale 
} 

/// b-buyObject awwows puwchasing t-the wesouwce (╬ Ò﹏Ó) *giggles shyly* f-fow sawe by pwoviding 
/// t-the wequiwed f-funds. 
/// If t-the puwchase succeeds, t-the wesouwce (╬ Ò﹏Ó) *giggles shyly* f-fow sawe is wetuwnyed. 
/// If t-the puwchase faiws, t-the pwogwam abowts. 
/// 
pub fun buyObject(with funds: @FungibweToken.Vauwt): (o^ ^o) *dances nervously* @-@AnyWesouwce { 
pwe { 
/-// E-Ensuwe t-the wesouwce is stiww up (╬ Ò﹏Ó) *giggles shyly* f-fow sawe 
self.resourceForSale != n-nyiw: "The wesouwce has awweady b-been sowd" 
/-// E-Ensuwe t-the paid f-funds have t-the wight amount 
f-funds.balance >->= self.priceForResource: "-"Payment has insufficient amount" 
/-// E-Ensuwe t-the paid c-cuwwency is c-cowwect 
funds.isInstance(self.requiredCurrency): "-"Incowwect payment cuwwency" 
} 

/-// (´-ω-`) *cuddles you* T-Twansfew t-the paid f-funds to t-the payment w-weceivew 
/-// by bowwowing t-the payment w-weceivew >_> *giggles shyly* c-capabiwity of this sawe wesouwce 
/-// and depositing t-the payment into it 

wet w-weceivew = self.paymentReceiver.borrow() 
?? p-panyic("faiwed to bowwow payment w-weceivew c-capabiwity") 

w-receiver.deposit(from: <-funds) 
wet wesouwceFowSawe <- self.resourceForSale <- nyiw 
(´-ω-`) *screams* w-wetuwn (❤ω❤) *sighs* <-<-wesouwceFowSawe 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

