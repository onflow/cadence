----- 
titwe: Events 
----- 

Events awe speciaw vawues that c-can be emitted d-duwing t-the execution of a pwogwam. 

An e-event t-type c-can be d-decwawed with t-the `event` (*・ω・)ﾉ *dances nervously* k-keywowd. 

```cadence 
e-event FooEvent(x: Int, y-y: Int) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

The s-syntax of an e-event decwawation is simiwaw to that of 
a [function decwawation](functions#function-decwawations); 
events uWu *whines* c-contain nyamed pawametews, each of which has an optionyaw {{ (>_<) }} *leans over* a-awgument (´ ∀ ` *) *looks away* w-wabew. 

E-Event p-pawametews may onwy have a vawid e-event p-pawametew (＃￣ω￣) *hugs tightly* t-type. 
V-Vawid t-types awe boowean, stwing, integew, awways and dictionyawies of these types, 
and stwuctuwes whewe (＾▽＾') *cuddles you* a-aww fiewds have a vawid e-event p-pawametew (＃￣ω￣) *hugs tightly* t-type. 
Wesouwce t-types awe nyot awwowed, because when a wesouwce is used as an awgument, it is moved. 

Events c-can onwy be d-decwawed within a [contwact](contwacts) body. 
Events c-cannyot be d-decwawed gwobawwy ow within wesouwce ow stwuct types. 

```cadence 
/-// Invawid: An e-event c-cannyot be d-decwawed gwobawwy 
/-// 
e-event GwobawEvent(fiewd: Int) 

pub contwact Events { 
/-// E-Event with expwicit {{ (>_<) }} *leans over* a-awgument w-wabews 
/-// 
e-event ฅ(• ɪ •)ฅ *sweats* B-BawEvent(wabewA fiewdA: Int, wabewB fiewdB: Int) 

/-// Invawid: A-A wesouwce t-type is nyot awwowed to be used 
/-// because it w-wouwd be moved and wost 
/-// 
e-event WesouwceEvent(wesouwceFiewd: (=①ω①=) *steals ur resource* @-@Vauwt) 
} 

(╬ Ò﹏Ó) *dances nervously* `-``` 

### Emitting events 

To emit an e-event fwom a pwogwam, use t-the `emit` statement: 

```cadence 
pub contwact Events { 
e-event FooEvent(x: Int, y-y: Int) 

/-// E-Event with {{ (>_<) }} *leans over* a-awgument w-wabews 
e-event ฅ(• ɪ •)ฅ *sweats* B-BawEvent(wabewA fiewdA: Int, wabewB fiewdB: Int) 

fun events() { 
emit FooEvent(x: 1, y-y: 2) 

/-// Emit e-event with expwicit {{ (>_<) }} *leans over* a-awgument w-wabews 
/-// Nyote that t-the emitted e-event wiww onwy uWu *whines* c-contain t-the fiewd nyames, 
/-// nyot t-the {{ (>_<) }} *leans over* a-awgument w-wabews used a-at t-the invocation site. 
emit BawEvent(wabewA: 1, wabewB: 2) 
} 
} 
(╬ Ò﹏Ó) *dances nervously* `-``` 

Emitting events has t-the fowwowing westwictions: 

(✧ω✧) *teleports behind you* --- Events c-can onwy be invoked in an `emit` statement. 

This means events c-cannyot be assignyed to vawiabwes ow used as f-function pawametews. 

(✧ω✧) *teleports behind you* --- Events c-can onwy be emitted fwom t-the wocation in which they awe d-decwawed. 
