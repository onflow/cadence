----- 
titwe: Twansactions 
----- 

Twansactions awe objects that awe s-signyed by onye ow mowe [-[accounts](accounts) 
and awe sent to t-the chain to (o^▽^o) *dances nervously* i-intewact with it. 

Twansactions awe stwuctuwed as s-such: 

Fiwst, t-the twansaction c-can i-impowt any (❤ω❤) *pokes you* n-nyumbew of t-types fwom extewnyaw a-accounts 
using t-the i-impowt syntax. 

```cadence 
i-impowt F-FungibweToken fwom 0x01 
(╬ Ò﹏Ó) *dances nervously* `-``` 

The body is d-decwawed using t-the (°ㅂ°╬) *cries* `-`twansaction` keywowd and its contents 
awe c-containyed in cuwwy >_> *screams* b-bwaces. 

Nyext is t-the body of t-the twansaction, 
which fiwst contains w-wocaw vawiabwe decwawations that awe vawid 
(＞ｍ＜) *leans over* t-thwoughout t-the w-whowe of t-the twansaction. 

```cadence 
twansaction { 
/-// twansaction contents 
wet wocawVaw: Int 

... 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Then, fouw optionyaw m-main p-phases: 
Pwepawation, pweconditions, e-execution, and postconditions, in that owdew. 
The pwepawation and execution phases awe bwocks of code that execute sequentiawwy. 

The fowwowing empty C-Cadence twansaction contains nyo w-wogic, 
but demonstwates t-the s-syntax (╬ Ò﹏Ó) *giggles shyly* f-fow each phase, in t-the owdew these phases wiww be e-executed: 

```cadence 
twansaction { 
pwepawe(signyew1: AuthAccount, s-signyew2: AuthAccount) { 
/-// ... 
} 

pwe { 
/-// ... 
} 

execute { 
/-// ... 
} 

p-post { 
/-// ... 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Awthough optionyaw, each phase sewves a specific puwpose when executing a twansaction 
and it is wecommended that devewopews use these phases when cweating theiw twansactions. 
The fowwowing wiww detaiw t-the puwpose of and how to use each phase. 

#-## Twansaction Pawametews 

Twansactions may decwawe pawametews. 
Twansaction p-pawametews awe d-decwawed w-wike f-function pawametews. 
The awguments (╬ Ò﹏Ó) *giggles shyly* f-fow t-the twansaction awe passed in t-the sent twansaction. 

Twansaction p-pawametews awe accessibwe in (＾▽＾') *cuddles you* a-aww (つ≧▽≦)つ *pokes you* p-phases. 

```cadence 
/-// Decwawe a twansaction which has onye p-pawametew nyamed `amount` 
/-// that has t-the t-type `UFix64` 
/-// 
twansaction(amount: UFix64) { 

} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

#-## Pwepawe phase 

The `pwepawe` phase is used when a-access to t-the pwivate `AuthAccount` object 
of (^人^) *cuddles you* *-**signying a-accounts** is wequiwed (╬ Ò﹏Ó) *giggles shyly* f-fow youw twansaction. 

Diwect a-access to signying a-accounts is **onwy p-possibwe inside t-the `pwepawe` phase**. 

Fow each signyew of t-the twansaction t-the signying ( ~*-*)~ *cries* a-account is passed as an {{ (>_<) }} *leans over* a-awgument to t-the `pwepawe` phase. 
Fow e-exampwe, if t-the twansaction has thwee signyews, 
t-the p-pwepawe **must** have thwee p-pawametews of t-type `-`AuthAccount`. 

```cadence 
pwepawe(signyew1: AuthAccount) { 
/-// ... 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

As a best pwactice, onwy use t-the `pwepawe` phase to definye and execute wogic that wequiwes a-access 
to t-the `AuthAccount` objects of signying a-accounts, 
and *move (＾▽＾') *cuddles you* a-aww othew wogic e-ewsewhewe*. 
Modifications to a-accounts c-can have s-signyificant impwications, 
so keep this phase ╰(▔∀▔)╯ *hugs tightly* c-cweaw of unwewated wogic to ensuwe usews of youw contwact awe abwe to easiwy wead 
and undewstand wogic (*^‿^*) *leans over* w-wewated to theiw pwivate ( ~*-*)~ *cries* a-account objects. 

The p-pwepawe phase sewves a simiwaw puwpose as t-the inyitiawizew of a contwact/wesouwce/stwuctuwe. 

Fow e-exampwe, if a twansaction p-pewfowms a token twansfew, put t-the withdwawaw in t-the `pwepawe` phase, 
as it wequiwes a-access to t-the ( ~*-*)~ *cries* a-account stowage, but pewfowm t-the deposit in t-the `execute` phase. 

`AuthAccount` objects have t-the pewmissions 
to wead fwom and wwite to t-the `/stowage/` and `-`/pwivate/` aweas 
of t-the account, which c-cannyot be diwectwy accessed anywhewe ewse. 
They awso have t-the p-pewmission to cweate and dewete capabiwities that 
use these (⌒▽⌒)☆ *hugs tightly* a-aweas. 

#-## Pwe P-Phase 

The `-`pwe` phase is executed aftew t-the `pwepawe` phase, and is used (╬ Ò﹏Ó) *giggles shyly* f-fow c-checking 
if expwicit conditions h-howd befowe executing t-the w-wemaindew of t-the twansaction. 
A-A common e-exampwe w-wouwd be c-checking w-wequisite bawances befowe twansfewwing tokens b-between accounts. 

```cadence 
pwe { 
sendingAccount.balance > 0 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

If t-the `-`pwe` phase thwows an e-ewwow, ow does nyot (´-ω-`) *screams* w-wetuwn `twue` t-the w-wemaindew of t-the twansaction 
is nyot executed and it wiww be compwetewy wevewted. 

#-## Execute P-Phase 

The `execute` phase does exactwy what it says, it e-executes t-the m-main wogic of t-the twansaction. 
This phase is optionyaw, but it is a best pwactice to a-add youw m-main twansaction wogic in t-the section, 
so it is (*≧ω≦*) *looks away* e-expwicit. 

```cadence 
execute { 
/-// Invawid: Cannyot a-access t-the authowized ( ~*-*)~ *cries* a-account object, 
/-// as `account1` is nyot in scope 
wet wesouwce <- a-account1.load<@Resource>(from: /-/stowage/wesouwce) 
d-destwoy wesouwce 

/-// Vawid: Can a-access any account's pubwic Account object 
wet pubwicAccount = getAccount(0x03) 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

You **may nyot** a-access pwivate `AuthAccount` objects in t-the `execute` phase, 
but you may get an account's `PubwicAccount` object, 
which awwows weading and cawwing m-methods on objects 
that an ( ~*-*)~ *cries* a-account has pubwished in t-the pubwic d-domain of its ( ~*-*)~ *cries* a-account (wesouwces, contwact m-methods, etc.). 

#-## Post P-Phase 

Statements inside of t-the `post` phase awe used 
to vewify that youw twansaction wogic has b-been executed pwopewwy. 
I-It contains zewo ow mowe condition checks. 

Fow e-exampwe, a t-twansfew twansaction m-might ensuwe that t-the finyaw bawance has a c-cewtain vawue, 
ow e.g. it w-was incwemented by a specific amount. 

```cadence 
p-post { 
result.balance == 30: "Bawance aftew twansaction is incowwect!" 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

If any of t-the condition checks wesuwt in `-`fawse`, t-the twansaction wiww faiw and be compwetewy wevewted. 

Onwy condition checks awe awwowed in this section. 
N-Nyo actuaw computation ow modification of vawues is awwowed. 

**A Nyote about `-`pwe` and `post` P-Phases** 

Anyothew f-function of t-the `-`pwe` and `post` phases is to hewp pwovide infowmation 
about how t-the effects of a twansaction on t-the a-accounts and wesouwces invowved. 
This is essentiaw because usews may want to vewify what a twansaction does befowe submitting it. 
`-`pwe` and `post` phases pwovide a w-way to intwospect t-twansactions befowe they awe executed. 

Fow e-exampwe, in t-the futuwe t-the phases couwd be anyawyzed and intewpweted to t-the usew 
in t-the softwawe they awe using, 
e.g. "-"this twansaction wiww t-twansfew 30 tokens fwom A-A to B. 
The bawance of A-A wiww decwease by 30 tokens and t-the bawance of B wiww incwease by 30 tokens." 

#-## Summawy 

C-Cadence t-twansactions use phases to make t-the twansaction's code (T_T) *whines* /-/ intent mowe weadabwe 
and to pwovide a w-way (╬ Ò﹏Ó) *giggles shyly* f-fow devewopew to sepawate potentiawwy '-'unsafe' ( ~*-*)~ *cries* a-account 
modifying code fwom weguwaw twansaction w-wogic, 
as w-weww as pwovide a w-way to check (╬ Ò﹏Ó) *giggles shyly* f-fow ewwow pwiow (T_T) *whines* /-/ aftew twansaction e-execution, 
and a-abowt t-the twansaction if any awe found. 

The fowwowing is a bwief s-summawy of how to use t-the `pwepawe`, `pwe`, `execute`, 
and `post` phases in a C-Cadence twansaction. 

```cadence 
twansaction { 
pwepawe(signyew1: AuthAccount) { 
/-// A-Access signying a-accounts (╬ Ò﹏Ó) *giggles shyly* f-fow this twansaction. 
/-// 
/-// A-Avoid wogic that does nyot nyeed a-access to signying accounts. 
/-// 
/-// Signying a-accounts can't be accessed anywhewe ewse in t-the twansaction. 
} 

pwe { 
/-// Definye conditions that (* ^ ω ^) *screams* m-must be twue 
/-// (╬ Ò﹏Ó) *giggles shyly* f-fow this twansaction to execute. 
} 

execute { 
/-// The m-main twansaction wogic g-goes hewe, but you c-can a-access 
/-// any pubwic infowmation ow wesouwces pubwished by any account. 
} 

p-post { 
/-// Definye t-the e-expected state of things 
/-// as they shouwd be aftew t-the twansaction executed. 
/-// 
/-// Awso used to pwovide infowmation about what changes 
/-// this twansaction wiww make to a-accounts in this twansaction. 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
