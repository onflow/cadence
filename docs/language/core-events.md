---
title: Core Events
---

Core events are events emitted directly from the FVM (Flow virtual machine). The events have the same name on all networks and don't follow the standard naming.

### Account Created

Event that is emitted when a new account gets created.

Event name: `flow.AccountCreated`


```cadence
pub event AccountCreated(address: Address)
```

| Field             | Type   | Description                                                            |
| ----------------- | ------ | ---------------------------------------------------------------------- |
| address       | Address | The address of the newly created account |


### Account Key Added

Event that is emitted when a key gets added to an account.

Event name: `flow.AccountKeyAdded`

```cadence
pub event AccountKeyAdded(address: Address, publicKey: [UInt8])
```

| Field             | Type   | Description                                                            |
| ----------------- | ------ | ---------------------------------------------------------------------- |
| address       | Address | The address of the account the key is added to |
| publicKey       | [UInt8] | Public key added to an account |


### Account Key Removed

Event that is emitted when a key gets removed from an account.

Event name: `flow.AccountKeyRemoved`

```cadence
pub event AccountKeyRemoved(address: Address, publicKey: [UInt8])
```

| Field             | Type   | Description                                                            |
| ----------------- | ------ | ---------------------------------------------------------------------- |
| address       | Address | The address of the account the key is removed from |
| publicKey       | [UInt8] | Public key removed from an account |


### Account Contract Added

Event that is emitted when a contract gets deployed to an account.

Event name: `flow.AccountContractAdded`

```cadence
pub event AccountContractAdded(address: Address, codeHash: [UInt8], contract: String)
```

| Field             | Type   | Description                                                            |
| ----------------- | ------ | ---------------------------------------------------------------------- |
| address       | Address | The address of the account the contract gets deployed to |
| codeHash       | [UInt8] | Hash of the contract source code |
| contract       | String | The name of the the contract |

### Account Contract Updated

Event that is emitted when a contract gets updated on an account.

Event name: `flow.AccountContractUpdated`

```cadence
pub event AccountContractUpdated(address: Address, codeHash: [UInt8], contract: String)
```

| Field             | Type   | Description                                                            |
| ----------------- | ------ | ---------------------------------------------------------------------- |
| address       | Address | The address of the account the contract gets updated on |
| codeHash       | [UInt8] | Hash of the contract source code |
| contract       | String | The name of the the contract |


### Account Contract Removed

Event that is emitted when a contract gets removed from an account.

Event name: `flow.AccountContractRemoved`

```cadence
pub event AccountContractRemoved(address: Address, codeHash: [UInt8], contract: String)
```

| Field             | Type   | Description                                                            |
| ----------------- | ------ | ---------------------------------------------------------------------- |
| address       | Address | The address of the account the contract gets removed from |
| codeHash       | [UInt8] | Hash of the contract source code |
| contract       | String | The name of the the contract |

