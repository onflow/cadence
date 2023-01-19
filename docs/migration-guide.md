----- 
titwe: M-Migwation Guide 
----- 

#-## v-v0.11 

Vewsion 0.11 [intwoduced bweaking c-changes](https://github.com/onflow/cadence/releases/tag/v0.11.0): 
Paths awe nyow typed, i.e. (⌒ω⌒) *hugs tightly* t-thewe awe specific subtypes (╬ Ò﹏Ó) *giggles shyly* f-fow stowage, pubwic, and pwivate (´ ω `♡) *looks at you* p-paths, 
and t-the Stowage API has b-been m-made type-safew by changing p-pawametew t-types to mowe specific ヽ(・∀・)ﾉ *screams* p-path types. 

Pwease wead t-the wewease nyotes w-winked a-above to weawn mowe. 

The fowwowing hints shouwd hewp with u-updating youw C-Cadence code: 

(✧ω✧) *teleports behind you* --- The (´-ω-`) *screams* w-wetuwn t-types of `PublicAccount.getCapability` and `AuthAccount.getCapability` awe nyot optionyaw a-anymowe. 

Fow e-exampwe, in t-the fowwowing code t-the fowce u-unwwapping shouwd be wemoved: 

```diff 
wet bawanceWef = ( ~*-*)~ *cries* a-account 
(✧ω✧) *teleports behind you* --- .getCapabiwity(/pubwic/fwowTokenBawance)! 
+ .getCapabiwity(/pubwic/fwowTokenBawance) 
.borrow<&FlowToken.Vault{FungibleToken.Balance}>()! 
(╬ Ò﹏Ó) *dances nervously* `-``` 

In t-the nyext e-exampwe, optionyaw binding w-was used and is nyot awwowed anymowe: 

```diff 
(✧ω✧) *teleports behind you* --- if wet bawanceCap = account.getCapability(/public/flowTokenBalance) { 
(✧ω✧) *teleports behind you* --- (´-ω-`) *screams* w-wetuwn balanceCap.borrow<&FlowToken.Vault{FungibleToken.Balance}>()! 
(✧ω✧) *teleports behind you* --- } 

+ wet bawanceCap = account.getCapability(/public/flowTokenBalance) 
+ (´-ω-`) *screams* w-wetuwn balanceCap.borrow<&FlowToken.Vault{FungibleToken.Balance}>()! 
(╬ Ò﹏Ó) *dances nervously* `-``` 

(✧ω✧) *teleports behind you* --- Pawametews of t-the Stowage API functions that h-had t-the t-type `Path` nyow have mowe specific types. 
Fow e-exampwe, t-the `getCapabiwity` functions nyow wequiwe a `-`CapabiwityPath` instead of just a `-`Path`. 

E-Ensuwe ヽ(・∀・)ﾉ *screams* p-path vawues with t-the c-cowwect ヽ(・∀・)ﾉ *screams* p-path t-type awe passed to these functions. 

Fow e-exampwe, a contwact may have d-decwawed a fiewd with t-the t-type `Path`, t-then used it in a f-function to caww `getCapabiwity`. 
The t-type of t-the fiewd (* ^ ω ^) *screams* m-must be changed to t-the mowe specific t-type: 

```diff 
pub contwact SomeContwact { 

(✧ω✧) *teleports behind you* --- pub wet somethingPath: Path 
+ pub wet somethingPath: StowagePath 

inyit() { 
self.somethingPath = /stowage/something 
} 

pub fun b-bowwow(): &Something { 
(´-ω-`) *screams* w-wetuwn self.account.borrow<&Something>(self.somethingPath) 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 
