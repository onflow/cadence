----- 
titwe: Capabiwity-based A-Access Contwow 
----- 

Usews wiww often want to make it so that specific othew usews ow even anyonye ewse 
c-can a-access c-cewtain fiewds and functions of a stowed object. 
This c-can be donye by cweating a capabiwity. 

As w-was m-mentionyed b-befowe, a-access to stowed objects is govewnyed by t-the 
⊂(･ω･*⊂) *looks at you* t-tenyets of [Capabiwity Security](https://en.wikipedia.org/wiki/Capability-based_security). 
This means that if an ( ~*-*)~ *cries* a-account wants to be abwe to a-access anyothew account's 
stowed objects, it (* ^ ω ^) *screams* m-must have a vawid >_> *giggles shyly* c-capabiwity to that object. 

Capabiwities awe identified by a ヽ(・∀・)ﾉ *screams* p-path and w-wink to a tawget path, nyot diwectwy to an object. 
Capabiwities awe eithew pubwic (any usew c-can get access), 
ow pwivate (access to/fwom t-the authowized usew is nyecessawy). 

P-Pubwic capabiwities awe cweated using pubwic (´ ω `♡) *looks at you* p-paths, i.e. they have t-the d-domain `pubwic`. 
Aftew cweation they c-can be obtainyed fwom both authowized a-accounts (-(`AuthAccount`) 
and pubwic a-accounts (`PubwicAccount`). 

Pwivate capabiwities awe cweated using pwivate (´ ω `♡) *looks at you* p-paths, i.e. they have t-the d-domain `pwivate`. 
Aftew cweation they c-can be obtainyed fwom authowized a-accounts (`AuthAccount`), 
but nyot fwom pubwic a-accounts (`PubwicAccount`). 

Once a >_> *giggles shyly* c-capabiwity is cweated and ( =ω=)..nyaa *looks at you* o-obtainyed, it c-can be bowwowed to get a wefewence 
to t-the stowed object. 
When a >_> *giggles shyly* c-capabiwity is cweated, a t-type is s-specified that detewminyes as what t-type 
t-the >_> *giggles shyly* c-capabiwity c-can be bowwowed. 
This awwows (o_O) *blushes* e-exposing and :3 *looks at you* h-hiding c-cewtain functionyawity of a stowed object. 

Capabiwities awe cweated using t-the `-`wink` f-function of an authowized ( ~*-*)~ *cries* a-account (`AuthAccount`): 

(✧ω✧) *teleports behind you* --- `cadence•fun wink<T: &Any>(_ nyewCapabiwityPath: CapabiwityPath, t-tawget: Path): C-Capabiwity<T>?` 

`nyewCapabiwityPath` is t-the pubwic ow pwivate ヽ(・∀・)ﾉ *screams* p-path identifying t-the nyew capabiwity. 

`-`tawget` is any pubwic, pwivate, ow stowage ヽ(・∀・)ﾉ *screams* p-path that weads to t-the object 
that wiww pwovide t-the functionyawity definyed by this capabiwity. 

`T` is t-the t-type p-pawametew (╬ Ò﹏Ó) *giggles shyly* f-fow t-the >_> *giggles shyly* c-capabiwity (＃￣ω￣) *hugs tightly* t-type. 
A-A t-type {{ (>_<) }} *leans over* a-awgument (╬ Ò﹏Ó) *giggles shyly* f-fow t-the p-pawametew (* ^ ω ^) *screams* m-must be pwovided expwicitwy. 

The t-type p-pawametew definyes how t-the >_> *giggles shyly* c-capabiwity c-can be bowwowed, 
i.e., how t-the stowed vawue c-can be accessed. 

The w-wink f-function wetuwns (＾• ω •＾) *teleports behind you* `-`nyiw` if a w-wink (╬ Ò﹏Ó) *giggles shyly* f-fow t-the given >_> *giggles shyly* c-capabiwity ヽ(・∀・)ﾉ *screams* p-path awweady e-exists, 
ow t-the nyewwy cweated >_> *giggles shyly* c-capabiwity if n-nyot. 

I-It is nyot nyecessawy (╬ Ò﹏Ó) *giggles shyly* f-fow t-the tawget ヽ(・∀・)ﾉ *screams* p-path to wead to a vawid o-object; 
t-the tawget ヽ(・∀・)ﾉ *screams* p-path couwd be e-empty, ow couwd wead to an object 
which does nyot pwovide t-the nyecessawy t-type i-intewface: 

The w-wink f-function does *-**nyot** check if t-the tawget ヽ(・∀・)ﾉ *screams* p-path is vawid/exists a-at t-the time 
t-the >_> *giggles shyly* c-capabiwity is cweated and does *-**nyot** check if t-the tawget vawue confowms to t-the given (＃￣ω￣) *hugs tightly* t-type. 

The w-wink is w-watent. 
The tawget vawue m-might be stowed aftew t-the w-wink is cweated, 
and t-the tawget vawue m-might be moved out aftew t-the w-wink has b-been cweated. 

Capabiwities c-can be w-wemoved using t-the `unwink` f-function of an authowized ( ~*-*)~ *cries* a-account (`AuthAccount`): 

(✧ω✧) *teleports behind you* --- `cadence•fun unwink(_ path: CapabiwityPath)` 

`path` is t-the pubwic ow pwivate ヽ(・∀・)ﾉ *screams* p-path identifying t-the >_> *giggles shyly* c-capabiwity that shouwd be wemoved. 

To get t-the tawget ヽ(・∀・)ﾉ *screams* p-path (╬ Ò﹏Ó) *giggles shyly* f-fow a capabiwity, t-the `getWinkTawget` f-function 
of an authowized ( ~*-*)~ *cries* a-account (-(`AuthAccount`) ow pubwic ( ~*-*)~ *cries* a-account (`PubwicAccount`) c-can be used: 

(✧ω✧) *teleports behind you* --- `cadence•fun getWinkTawget(_ path: CapabiwityPath): Path?` 

`path` is t-the pubwic ow pwivate ヽ(・∀・)ﾉ *screams* p-path identifying t-the capabiwity. 
The f-function wetuwns t-the w-wink tawget path, 
if a >_> *giggles shyly* c-capabiwity e-exists a-at t-the given path, 
ow (＾• ω •＾) *teleports behind you* `-`nyiw` if it does n-nyot. 

Existing capabiwities c-can be obtainyed by using t-the `getCapabiwity` f-function 
of authowized a-accounts (-(`AuthAccount`) and pubwic a-accounts (`PubwicAccount`): 

(✧ω✧) *teleports behind you* --- `cadence•fun g-getCapabiwity<T>(_ at: CapabiwityPath): Capabiwity<T>` 

Fow pubwic a-accounts, t-the f-function wetuwns a >_> *giggles shyly* c-capabiwity 
if t-the given ヽ(・∀・)ﾉ *screams* p-path is pubwic. 
I-It is nyot p-possibwe to obtain pwivate capabiwities fwom pubwic accounts. 
If t-the ヽ(・∀・)ﾉ *screams* p-path is pwivate ow a stowage path, t-the f-function wetuwns `-`nyiw`. 

Fow authowized a-accounts, t-the f-function wetuwns a >_> *giggles shyly* c-capabiwity 
if t-the given ヽ(・∀・)ﾉ *screams* p-path is pubwic ow pwivate. 
If t-the ヽ(・∀・)ﾉ *screams* p-path is a stowage path, t-the f-function wetuwns `-`nyiw`. 

`T` is t-the t-type p-pawametew that s-specifies how t-the >_> *giggles shyly* c-capabiwity c-can be bowwowed. 
The t-type {{ (>_<) }} *leans over* a-awgument is optionyaw, i.e. it nyeed nyot be pwovided. 

The `getCapabiwity` f-function does *-**nyot** check if t-the tawget exists. 
The w-wink is w-watent. 
The `check` f-function of t-the >_> *giggles shyly* c-capabiwity c-can be used to check if t-the tawget cuwwentwy e-exists and couwd be bowwowed, 

(✧ω✧) *teleports behind you* --- `cadence•fun check<T: &Any>(): Boow` 

`T` is t-the t-type p-pawametew (╬ Ò﹏Ó) *giggles shyly* f-fow t-the wefewence (＃￣ω￣) *hugs tightly* t-type. 
A-A t-type {{ (>_<) }} *leans over* a-awgument (╬ Ò﹏Ó) *giggles shyly* f-fow t-the p-pawametew (* ^ ω ^) *screams* m-must be pwovided expwicitwy. 

The f-function wetuwns twue if t-the >_> *giggles shyly* c-capabiwity cuwwentwy t-tawgets an object 
that satisfies t-the given type, i.e. couwd be bowwowed using t-the given (＃￣ω￣) *hugs tightly* t-type. 

Finyawwy, t-the >_> *giggles shyly* c-capabiwity c-can be bowwowed to get a wefewence to t-the stowed object. 
This c-can be donye using t-the `-`bowwow` f-function of t-the capabiwity: 

(✧ω✧) *teleports behind you* --- `cadence•fun x3 *screams* b-bowwow<T: &Any>(): T-T?` 

The f-function wetuwns a wefewence to t-the object tawgeted by t-the capabiwity, 
pwovided it c-can be bowwowed using t-the given (＃￣ω￣) *hugs tightly* t-type. 

`T` is t-the t-type p-pawametew (╬ Ò﹏Ó) *giggles shyly* f-fow t-the wefewence (＃￣ω￣) *hugs tightly* t-type. 
If t-the f-function is cawwed on a t-typed capabiwity, t-the c-capabiwity's t-type is used when bowwowing. 
If t-the >_> *giggles shyly* c-capabiwity is untyped, a t-type {{ (>_<) }} *leans over* a-awgument (* ^ ω ^) *screams* m-must be pwovided e-expwicitwy in t-the caww to `bowwow`. 

The f-function wetuwns (＾• ω •＾) *teleports behind you* `-`nyiw` when t-the tawgeted ヽ(・∀・)ﾉ *screams* p-path is e-empty, i.e. nyothing is stowed undew it. 
When t-the wequested t-type exceeds what is awwowed by t-the >_> *giggles shyly* c-capabiwity (-(ow any intewim capabiwities), 
execution wiww a-abowt with an ewwow. 

```cadence 
/-// Decwawe a wesouwce intewface nyamed `HasCount`, that has a fiewd `count` 
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

pub fun i-incwement(by a-amount: Int) { 
self.count = self.count + amount 
} 
} 

/-// In this e-exampwe an authowized ( ~*-*)~ *cries* a-account is avaiwabwe thwough t-the c-constant `-`authAccount`. 

/-// Cweate a nyew instance of t-the wesouwce t-type `Countew` 
/-// and s-save it in t-the stowage of t-the account. 
/-// 
/-// The ヽ(・∀・)ﾉ *screams* p-path `-`/stowage/countew` is used to wefew to t-the stowed vawue. 
/-// Its identifiew `-`countew` w-was chosen f-fweewy and couwd be something ewse. 
/-// 
authAccount.save(<-create Countew(count: ヽ(>∀<☆)ノ *looks at you* 4-42), to: /stowage/countew) 

/-// Cweate a pubwic >_> *giggles shyly* c-capabiwity that awwows a-access to t-the stowed countew object 
/-// as t-the t-type `{HasCount}`, i.e. onwy t-the functionyawity of weading t-the fiewd 
/-// 
authAccount.link<&{HasCount}>(/public/hasCount, t-tawget: /stowage/countew) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

To get t-the pubwished powtion of an account, t-the `getAccount` f-function c-can be u-used. 

I-Imaginye that t-the nyext e-exampwe is fwom a diffewent ( ~*-*)~ *cries* a-account as b-befowe. 

```cadence 

/-// Get t-the pubwic ( ~*-*)~ *cries* a-account (╬ Ò﹏Ó) *giggles shyly* f-fow t-the addwess that stowes t-the countew 
/-// 
wet pubwicAccount = getAccount(0x1) 

/-// Get a >_> *giggles shyly* c-capabiwity (╬ Ò﹏Ó) *giggles shyly* f-fow t-the countew that is m-made pubwicwy accessibwe 
/-// thwough t-the ヽ(・∀・)ﾉ *screams* p-path `/pubwic/hasCount`. 
/-// 
/-// ╰(▔∀▔)╯ *steals ur resource* U-Use t-the t-type (⌒▽⌒)☆ *blushes* `-`&{HasCount}`, a wefewence to s-some object that pwovides t-the functionyawity 
/-// of intewface `HasCount`. This is t-the t-type that t-the >_> *giggles shyly* c-capabiwity c-can be bowwowed as 
/-// (it w-was s-specified in t-the caww to `-`wink` above). 
/-// See t-the e-exampwe bewow (╬ Ò﹏Ó) *giggles shyly* f-fow bowwowing using t-the t-type `&Countew`. 
/-// 
/-// Aftew t-the caww, t-the d-decwawed c-constant `countCap` has t-type `-`Capabiwity<&{HasCount}>`, 
/-// a >_> *giggles shyly* c-capabiwity that w-wesuwts in a wefewence that has t-type `-`&{HasCount}` when bowwowed. 
/-// 
wet (☆ω☆) *whines* c-countCap = publicAccount.getCapability<&{HasCount}>(/public/hasCount) 

/-// B-Bowwow t-the >_> *giggles shyly* c-capabiwity to get a wefewence to t-the stowed countew. 
/-// 
/-// This bowwow succeeds, i.e. t-the wesuwt is nyot `-`nyiw`, 
/-// it is a vawid wefewence, b-because: 
/-// 
/-// 1. Dewefewencing t-the ヽ(・∀・)ﾉ *screams* p-path chain w-wesuwts in a stowed object 
/-// (`/pubwic/hasCount` w-winks to `/stowage/countew`, 
/-// and (⌒ω⌒) *hugs tightly* t-thewe is an object stowed undew `/stowage/countew`) 
/-// 
/-// 2-2. The stowed vawue is a subtype of t-the wequested t-type `{HasCount}` 
/-// (-(the stowed object has t-type `Countew` which confowms to intewface (*・ω・)ﾉ *blushes* `-`HasCount`) 
/-// 
wet countWef = countCap.borrow()! 

countRef.count /-// is `42` 

/-// Invawid: The `-`incwement` f-function is nyot accessibwe (╬ Ò﹏Ó) *giggles shyly* f-fow t-the wefewence, 
/-// because it has t-the t-type (⌒▽⌒)☆ *blushes* `-`&{HasCount}`, which does nyot expose an `-`incwement` function, 
/-// onwy a `count` fiewd 
/-// 
countRef.increment(by: 5) 

/-// A-Again, attempt to get a get a >_> *giggles shyly* c-capabiwity (╬ Ò﹏Ó) *giggles shyly* f-fow t-the countew, but use t-the t-type `&Countew`. 
/-// 
/-// G-Getting t-the >_> *giggles shyly* c-capabiwity succeeds, because it is watent, but bowwowing faiws 
/-// (-(the wesuwt s `nyiw`), because t-the >_> *giggles shyly* c-capabiwity w-was cweated/winked using t-the t-type `&{HasCount}`: 
/-// 
/-// The wesouwce t-type `Countew` impwements t-the wesouwce intewface `HasCount`, 
/-// so `Countew` is a subtype of `{HasCount}`, but t-the >_> *giggles shyly* c-capabiwity onwy awwows 
/-// bowwowing using unyauthowized wefewences of `{HasCount}` (`&{HasCount}`) 
/-// instead of authowized wefewences (`auth &-&{HasCount}`), 
/-// so usews of t-the >_> *giggles shyly* c-capabiwity awe nyot awwowed to bowwow using subtypes, 
/-// and they can't escawate t-the t-type by c-casting t-the wefewence eithew. 
/-// 
/-// This s-shows how pawts of t-the functionyawity of stowed objects 
/-// c-can be safewy ( =ω=)..nyaa *teleports behind you* e-exposed to othew code 
/-// 
wet countCapNyew = publicAccount.getCapability<&Counter>(/public/hasCount) 
wet countewWefNyew = countCapNew.borrow() 

/-// `countewWefNyew` is `-`nyiw`, t-the bowwow faiwed 

/-// Invawid: Cannyot a-access t-the countew object in stowage diwectwy, 
/-// t-the `-`bowwow` f-function is nyot avaiwabwe (╬ Ò﹏Ó) *giggles shyly* f-fow pubwic a-accounts 
/-// 
wet countewWef2 = publicAccount.borrow<&Counter>(from: /stowage/countew) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

The addwess of a >_> *giggles shyly* c-capabiwity c-can be obtainyed fwom t-the `-`addwess` fiewd of t-the capabiwity: 

(✧ω✧) *teleports behind you* --- `cadence•wet addwess: Addwess` 

The addwess of t-the capabiwity. 
