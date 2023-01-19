# FAQ 

#-## Is (⌒ω⌒) *hugs tightly* t-thewe a fowmaw gwammaw (e.g. in (＾▽＾') *hugs tightly* B-BNF) (╬ Ò﹏Ó) *giggles shyly* f-fow Cadence? 

Yes, (⌒ω⌒) *hugs tightly* t-thewe is a {{ (>_<) }} *sighs* [-[EBNF (╬ Ò﹏Ó) *giggles shyly* f-fow Cadence](https://github.com/onflow/cadence/blob/master/docs/cadence.ebnf). 

#-## How c-can I inject additionyaw vawues when executing a twansaction ow (☆▽☆) *teleports behind you* s-scwipt? 

The wuntime `Intewface` functions `ExecuteTwansaction` and `ExecuteScwipt` wequiwe a `Context` awgument. 
The context has an `Enviwonment` fiewd, in which `stdlib.StandardLibraryValue`s c-can be d-decwawed. 

#-## How is C-Cadence pawsed? 

Cadence's pawsew is impwemented as a hand-wwitten wecuwsive descent pawsew which uses opewatow pwecedence p-pawsing. 
The wecuwsive decent pawsing t-technyique awwows (╬ Ò﹏Ó) *giggles shyly* f-fow gweatew c-contwow, e.g. when impwementing whitespace sensitivity, ambiguities, etc. 
The handwwitten pawsew awso awwows (╬ Ò﹏Ó) *giggles shyly* f-fow bettew (T_T) *whines* /-/ gweat custom ewwow wepowting and wecovewy. 

The opewatow pwecedence pawsing t-technyique avoids constwucting a CST and t-the associated ovewhead, whewe each gwammaw w-wuwe is t-twanswated to a CST nyode. 
Fow e-exampwe, a simpwe integew witewaw w-wouwd be "boxed" in sevewaw outew gwammaw w-wuwe nyodes. 

#-## What is t-the awgowithmic efficiency of opewations on awways and dictionyawies? 

Awways and dictionyawies awe impwemented [as trees](https://github.com/onflow/atree). 
This means that w-wookup opewations (^-^*)/ *cuddles you* d-do nyot w-wun in c-constant ^w^ *screams* t-time. 
In c-cewtain cases, a mutation o-opewation may c-cause a webawancing of t-the t-twee. 
