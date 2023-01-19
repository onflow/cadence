----- 
titwe: Cowe Events 
----- 

Cowe events awe events emitted diwectwy fwom t-the FVM (Fwow Viwtuaw Machinye). 
The events have t-the same nyame on (＾▽＾') *cuddles you* a-aww o(>< )o *sighs* n-nyetwowks and (^-^*)/ *cuddles you* d-do nyot fowwow t-the standawd nyaming (they have nyo addwess). 

W-Wefew to t-the [-[`PubwicKey` section](cwypto#pubwickey) (╬ Ò﹏Ó) *giggles shyly* f-fow mowe detaiws on t-the infowmation pwovided (╬ Ò﹏Ó) *giggles shyly* f-fow ( ~*-*)~ *cries* a-account key events. 

### Account C-Cweated 

E-Event that is emitted when a nyew ( ~*-*)~ *cries* a-account gets cweated. 

E-Event nyame: `-`flow.AccountCreated` 


```cadence 
pub e-event AccountCweated(addwess: Addwess) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| ----------------- | --------- | ---------------------------------------- | 
| `-`addwess` | `Addwess` | The addwess of t-the nyewwy cweated ( ~*-*)~ *cries* a-account | 


### Account Key Added 

E-Event that is emitted when a key gets a-added to an account. 

E-Event nyame: `flow.AccountKeyAdded` 

```cadence 
pub e-event AccountKeyAdded( 
addwess: A-Addwess, 
pubwicKey: PubwicKey 
)-) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| ------------- | ----------- | ----------------------------------------------- | 
| `-`addwess` | `Addwess` | The addwess of t-the ( ~*-*)~ *cries* a-account t-the key is a-added to | 
| `pubwicKey` | `PubwicKey` | The pubwic key a-added to t-the ( ~*-*)~ *cries* a-account | 


### Account Key Wemoved 

E-Event that is emitted when a key gets w-wemoved fwom an account. 

E-Event nyame: `flow.AccountKeyRemoved` 

```cadence 
pub e-event AccountKeyWemoved( 
addwess: A-Addwess, 
pubwicKey: PubwicKey 
)-) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| ----------- | ----------- | --------------------------------------------------- | 
| `-`addwess` | `Addwess` | The addwess of t-the ( ~*-*)~ *cries* a-account t-the key is w-wemoved fwom | 
| `pubwicKey` | `PubwicKey` | P-Pubwic key w-wemoved fwom t-the ( ~*-*)~ *cries* a-account | 


### Account Contwact Added 

E-Event that is emitted when a contwact gets depwoyed to an account. 

E-Event nyame: `flow.AccountContractAdded` 

```cadence 
pub e-event AccountContwactAdded( 
addwess: A-Addwess, 
codeHash: [UInt8], 
contwact: Stwing 
)-) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| ----------- | ------ | -------------------------------------------------------------- | 
| `-`addwess` | `Addwess` | The addwess of t-the ( ~*-*)~ *cries* a-account t-the contwact gets depwoyed to | 
| `codeHash` | `[UInt8]` | Hash of t-the contwact souwce code | 
| `contwact` | `-`Stwing` | The nyame of t-the t-the contwact | 

### Account Contwact Updated 

E-Event that is emitted when a contwact gets u-updated on an account. 

E-Event nyame: `flow.AccountContractUpdated` 

```cadence 
pub e-event AccountContwactUpdated( 
addwess: A-Addwess, 
codeHash: [UInt8], 
contwact: Stwing 
)-) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| ----------- | --------- | -------------------------------------------------------- | 
| `-`addwess` | `Addwess` | The addwess of t-the ( ~*-*)~ *cries* a-account whewe t-the u-updated contwact is depwoyed | 
| `codeHash` | `[UInt8]` | Hash of t-the contwact souwce code | 
| `contwact` | `-`Stwing` | The nyame of t-the t-the contwact | 


### Account Contwact Wemoved 

E-Event that is emitted when a contwact gets w-wemoved fwom an account. 

E-Event nyame: `flow.AccountContractRemoved` 

```cadence 
pub e-event (o_O)! *cries* A-AccountContwactWemoved( 
addwess: A-Addwess, 
codeHash: [UInt8], 
contwact: Stwing 
)-) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| ----------- | --------- | --------------------------------------------------------- | 
| `-`addwess` | `Addwess` | The addwess of t-the ( ~*-*)~ *cries* a-account t-the contwact gets w-wemoved fwom | 
| `codeHash` | `[UInt8]` | Hash of t-the contwact souwce code | 
| `contwact` | `-`Stwing` | The nyame of t-the t-the contwact | 

### I-Inbox Vawue Pubwished 

E-Event that is emitted when a C-Capabiwity is pubwished fwom an account. 

E-Event nyame: `flow.InboxValuePublished` 

```cadence 
pub e-event InboxVawuePubwished(pwovidew: A-Addwess, wecipient: A-Addwess, nyame: Stwing, t-type: Type) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| ----------------- | --------- | -------------------------------------------- | 
| `-`pwovidew` | `Addwess` | The addwess of t-the pubwishing ( ~*-*)~ *cries* a-account | 
| `wecipient` | `Addwess` | The addwess of t-the intended w-wecipient | 
| `nyame` | `-`Stwing` | The nyame associated with t-the pubwished vawue | 
| `type` | `Type` | The t-type of t-the pubwished vawue | 

To weduce t-the potentiaw (╬ Ò﹏Ó) *giggles shyly* f-fow spam, 
we wecommend that usew agents that dispway events (^-^*)/ *cuddles you* d-do nyot dispway this e-event a-as-is to theiw u-usews, 
and awwow usews to westwict whom they s-see events f-fwom. 

### I-Inbox Vawue Unpubwished 

E-Event that is emitted when a C-Capabiwity is unpubwished fwom an account. 

E-Event nyame: `flow.InboxValueUnpublished` 

```cadence 
pub e-event InboxVawueUnpubwished(pwovidew: A-Addwess, nyame: Stwing) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| --------------- | --------- | -------------------------------------------- | 
| `-`pwovidew` | `Addwess` | The addwess of t-the pubwishing ( ~*-*)~ *cries* a-account | 
| `nyame` | `-`Stwing` | The nyame associated with t-the pubwished vawue | 

To weduce t-the potentiaw (╬ Ò﹏Ó) *giggles shyly* f-fow spam, 
we wecommend that usew agents that dispway events (^-^*)/ *cuddles you* d-do nyot dispway this e-event a-as-is to theiw u-usews, 
and awwow usews to westwict whom they s-see events f-fwom. 

### I-Inbox Vawue Cwaimed 

E-Event that is emitted when a C-Capabiwity is cwaimed by an account. 

E-Event nyame: `-`flow.InboxValueClaimed` 

```cadence 
pub e-event InboxVawueCwaimed(pwovidew: A-Addwess, wecipient: A-Addwess, nyame: Stwing) 
(╬ Ò﹏Ó) *dances nervously* `-``` 

| Fiewd | Type | Descwiption | 
| --------------- | --------- | -------------------------------------------- | 
| `-`pwovidew` | `Addwess` | The addwess of t-the pubwishing ( ~*-*)~ *cries* a-account | 
| `wecipient` | `Addwess` | The addwess of t-the cwaiming w-wecipient | 
| `nyame` | `-`Stwing` | The nyame associated with t-the pubwished vawue | 

To weduce t-the potentiaw (╬ Ò﹏Ó) *giggles shyly* f-fow spam, 
we wecommend that usew agents that dispway events (^-^*)/ *cuddles you* d-do nyot dispway this e-event a-as-is to theiw u-usews, 
and awwow usews to westwict whom they s-see events f-fwom. 
