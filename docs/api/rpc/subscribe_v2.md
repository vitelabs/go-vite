---
sidebarDepth: 4
---

# Subscribe
:::tip Maintainer
[viteLiz](https://github.com/viteLiz)
:::

Two subscription schemes, **polling** and **callback**, are provided in Vite.

* **Polling API** asks the client program to create a filter, then polls `subscribe_getChangesByFilterId` method for new events. 
Filter will expire if it has not been used in 5 minutes, in this case you should create a new filter for further usage. 
You can also manually stop subscription by calling `subscribe_uninstallFilter` method.

`subscribe_createSnapshotBlockFilter`, `subscribe_createAccountBlockFilter`, `subscribe_createAccountBlockFilterByAddress`, `subscribe_createUnreceivedBlockFilterByAddress`, `subscribe_createVmlogFilter`, `subscribe_uninstallFilter` and `subscribe_getChangesByFilterId` are polling APIs.

* **Callback API** registers new subscription through WebSocket. Once listening starts, subscribed events will be returned in callback when generated. 
This kind of subscription will close automatically when the WebSocket connection is broken.

`subscribe_createSnapshotBlockSubscription`, `subscribe_createAccountBlockSubscription`, `subscribe_createAccountBlockSubscriptionByAddress`, `subscribe_createUnreceivedBlockSubscriptionByAddress` and `subscribe_createVmlogSubscription` are callback APIs.

At the time being 5 kinds of events are supported: new snapshot, new transaction, new transaction on certain account, new unreceived transaction on certain account and new log. 
All events support rollback. If rollback takes place, `removed` field of the event is set to true.

:::tip Note
Add `"subscribe"` into `"PublicModules"` and set `"SubscribeEnabled":true` in node_config.json to enable subscription API
:::

## Example

### Subscribe via Callback 

First, register a new subscription on address `vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f` by calling `subscribe_subscribe` method with parameter `createVmlogSubscription`.
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": [
    "createVmlogSubscription",
    {
      "addressHeightRange":{
        "vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f":{
          "fromHeight":"0",
          "toHeight":"0"
        }
      }
    }
  ]
}
```
This will return a new subscription id `0x4b97e0674a5ebef942dbb07709c4a608`
```json
{
  "jsonrpc":"2.0",
  "id":1,
  "result":"0x4b97e0674a5ebef942dbb07709c4a608"
}
```
New logs will be sent to subscriber in callback's `result` field when generated
```json
{
  "jsonrpc":"2.0",
  "method":"subscribe_subscription",
  "params":{
    "subscription":"0x4b97e0674a5ebef942dbb07709c4a608",
    "result":[
      {
        "vmlog":{
          "topics":[
            "aa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb",
            "000000000000000000000000bb6ad02107a4422d6a324fd2e3707ad53cfed935"
          ],
          "data":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAo="
        },
        "accountBlockHash":"23ea04b0dea4b9d0aa4d1f84b246b298a30faba753fa48303ad2deb29cd27f40",
        "accountBlockHeight": "10",
        "address":"vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f",
        "removed":false
      }
    ]
  }
}
```
Callback subscription does not need to be cancelled explicitly. The subscription will stop once the connection is closed.

### Subscribe via Polling

Create a filter on address `vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f`
```json
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_createVmlogFilter",
	"params": [{
      "addressHeightRange":{
        "vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f":{"fromHeight":"0","toHeight":"0"}
      }
		}]
}
```
This will return a new filter id `0x61d780619649fb0872e1f94a40cec713`
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "0x61d780619649fb0872e1f94a40cec713"
}
```
Poll for new events periodically by filter id
```json
{
	"jsonrpc": "2.0",
	"id": 2,
	"method": "subscribe_getChangesByFilterId",
	"params": ["0x61d780619649fb0872e1f94a40cec713"]
}
```
New logs after last poll are returned
```json
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "result": [
            {
                "vmlog": {
                    "topics": [
                        "96a65b1cd08da045d0318cafda7b8c8436092851d5a4b7e75054c005a296e3fb",
                        "0000000000000000000000ab24ef68b84e642c0ddca06beec81c9acb1977bb00"
                    ],
                    "data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHsAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF"
                },
                "accountBlockHash": "802b82821ec52bdadb8b79a53363bf2f90645caef95a83c34af20c640a6c320b",
                "accountBlockHeight": "10",
                "address": "vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f",
                "removed": false
            }
        ],
        "subscription": "0x61d780619649fb0872e1f94a40cec713"
    }
}
```
You should cancel polling subscription by `subscribe_uninstallFilter` explicitly
```json
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_uninstallFilter",
	"params": ["0x61d780619649fb0872e1f94a40cec713"]
}
```

### Log Completion

Accidental connection close may cause subscription stopped. In this situation, historical snapshot blocks, account blocks or unreceived blocks can be fetched by `ledger_getSnapshotBlocks`, `ledger_getBlocksByHeight` or `ledger_getUnreceivedBlocksByAddress`, while historical logs can be retrieved by `ledger_getVmlogsByFilter` method.

For example, in order to re-subscribe to new snapshot event, you should call `subscribe_createSnapshotBlockFilter` or `subscribe_newSnapshotBlocks` to resume subscription first, then obtain latest snapshot block height by `ledger_getLatestSnapshotBlock`, finally call `ledger_getSnapshotBlocks` to get all missing snapshot blocks.

## subscribe_createSnapshotBlockFilter
Create a filter for polling for new snapshot blocks by passing into `subscribe_getChangesByFilterId` as parameter
  
- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_createSnapshotBlockFilter",
	"params": []
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "0xf90906914486a9c22d620e50022b38d5"
}
```
:::

## subscribe_createAccountBlockFilter
Create a filter for polling for new transactions on all accounts by passing into `subscribe_getChangesByFilterId` as parameter

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_createAccountBlockFilter",
	"params": []
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "0xf90906914486a9c22d620e50022b38d5"
}
```
:::

## subscribe_createAccountBlockFilterByAddress
Create a filter for polling for new transactions on specified account by passing into `subscribe_getChangesByFilterId` as parameter

- **Parameters**:
  * `string address`: Address of account

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_createAccountBlockFilterByAddress",
	"params": ["vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "0x4f18a08c6e6801aeb7a8cfbad0ca90af"
}
```
:::

## subscribe_createUnreceivedBlockFilterByAddress
Create a filter for polling for unreceived transactions on specified account by passing into `subscribe_getChangesByFilterId` as parameter. 
The events include new unreceived transaction generated, received and rolled back.

- **Parameters**:
  * `Address`: Address of account

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_createUnreceivedBlockFilterByAddress",
	"params": ["vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "0x64e1eb3d26517a0d736b3d85ae9ce299"
}
```
:::

## subscribe_createVmlogFilter
Create a filter for polling for new logs by passing into `subscribe_getChangesByFilterId` as parameter

- **Parameters**:
  * `FilterParam`
    * `addressHeightRange`: `map[Address]Range` Query logs of the specified account address with given range. Address and height range should be filled in the map. At least one address must be specified.
      * `fromHeight`: `uint64` Start height. `0` means starting from the latest block
      * `toHeight`: `uint64` End height. `0` means no specific ending block
    * `topics`: `[][]Hash` Prefix of topics. See below example for reference

```
Topic examples：
 {} matches all logs
 {{A}} matches the logs having "A" as the first element
 {{},{B}} matches the logs having "B" as the second element
 {{A},{B}} matches the logs having "A" as the first element and "B" as the second element
 {{A,B},{C,D}} matches the logs having "A" or "B" as the first element, and "C" or "D" as the second element
```

- **Returns**:  
	- `string` filterId
	
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_createVmlogFilter",
	"params": [{
		"addressHeightRange":{
			"vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792":{
				"fromHeight":"0",
				"toHeight":"0"
			}
		}
	}]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "0x8f34ddeb22b87fdfd2acb6c9f5a2b50d"
}
```
:::

## subscribe_uninstallFilter
Cancel subscription by removing specified filter 

- **Parameters**:
  * `string`: filterId

- **Returns**:  
	- `bool` If `true`, the subscription is cancelled

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_uninstallFilter",
	"params": ["0x8f34ddeb22b87fdfd2acb6c9f5a2b50d"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": true
}
```
:::

## subscribe_getChangesByFilterId
Poll for new events. The content of return value depends on various subscription. If no new event is generated, empty array will be returned.

- **Parameters**:
  * `string`: filterId

- **Returns (in case of `subscribe_createSnapshotBlockFilter`)**: 
  * `SnapshotBlocks`
    * `subscription`: `string` filterId
    * `result`: `Array<SnapshotBlockMessage>`
      * `hash`: `string hash` Hash of snapshot block
      * `height`: `string uint64` Height of snapshot block
      * `removed`: `bool` If `true`, the snapshot block was rolled back. For new snapshot block, `false` is filled

- **Returns (in case of `subscribe_createAccountBlockFilter`)**: 
  * `AccountBlocks`
    * `subscription`: `string` filterId
    * `result`: `Array<AccountBlockMessage>`
      * `hash`: `string hash` Hash of account block
      * `removed`: `bool` If `true`, the account block was rolled back. For new account block, `false` is filled

- **Returns (in case of `subscribe_createAccountBlockFilterByAddress`)**: 
  * `AccountBlocksWithHeight`
    * `subscription`: `string` filterId
    * `result`: `Array<AccountBlockWithHeightMessage>`
      * `hash`: `string hash` Hash of account block
      * `height`: `string height` Height of account block
      * `removed`: `bool` If `true`, the account block was rolled back. For new account block, `false` is filled

- **Returns (in case of `subscribe_createUnreceivedBlockFilterByAddress`)**: 
  * `UnreceivedBlocks`
    * `subscription`: `string` filterId
    * `result`: `Array<UnreceivedBlockMessage>`
      * `hash`: `string hash` Hash of account block
      * `received`: `bool` If `true`, the transaction was received
      * `removed`: `bool` If `true`, the unreceived transaction was rolled back. In the case of `false`, if `received` is `false`, this is new unreceived block, otherwise (`received` is `true`) the transaction has been received. 
  
- **Returns (in case of `subscribe_createVmlogFilter`)**:
  * `Vmlogs` 
    * `subscription`: `string` filterId
    * `result`: `Array<VmlogMessage>`
      * `accountBlockHash`: `Hash` Hash of account block
      * `accountBlockHeight`: `uint64` Height of account block
      * `address`: `Address` Address of account
      * `vmlog`: `VmLog` Event log of smart contract
        * `topics`: `Array<string hash>` Event signature and indexed field. The signature can be generated from ABI
        * `data`: `string base64` Non-indexed field of event, can be decoded based on ABI
      * `removed`: `bool` If `true`, the log has been rolled back

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_getChangesByFilterId",
	"params": ["0xf90906914486a9c22d620e50022b38d5"]
}
```
```json tab:SnapshotBlocks
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "hash": "5d47f2e0a532923f7ee53e7b465381f197a669e86155d541b3b7f3d63f07a0e2",
              "height": "124",
              "removed": false
          },
          {
              "hash": "78b19cb84ac293d4af3f36e741938929f6d3311362e1265e87bbaa74e5fcef09",
              "height": "125",
              "removed": false
          },
          {
              "hash": "94437996b3e70afd5d8593e2020ae56f63dbbc538df1ead1633340393bd52c1a",
              "height": "126",
              "removed": false
          }
      ],
      "subscription": "0xf90906914486a9c22d620e50022b38d5"
    }
}
```
```json tab:AccountBlocks
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "hash": "9cc9ba996a4192e35ddbfe3ba448611fc06f6342463e21d3300e58e9772b348f",
              "removed": false
          },
          {
              "hash": "8b9f8067ef09aa09c8f9d652f0d9ac5e99d083722089a6d91323cffd8b2dcf08",
              "removed": false
          }
      ],
      "subscription": "0xf90906914486a9c22d620e50022b38d5"
    }
}
```
```json tab:AccountBlocksWithHeight
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "hash": "72ec861cb2f6c32a48632407f3aa1b05d5ad450ef75fa7660dd39d7be6d3ab68",
              "height": "15",
              "removed": false
          }
      ],
      "subscription": "0xf90906914486a9c22d620e50022b38d5"
    }
}
```
```json tab:UnreceivedBlocks
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "hash": "72ec861cb2f6c32a48632407f3aa1b05d5ad450ef75fa7660dd39d7be6d3ab68",
              "received": false,
              "removed": false
          },
          {
              "hash": "72ec861cb2f6c32a48632407f3aa1b05d5ad450ef75fa7660dd39d7be6d3ab68",
              "received": true,
              "removed": false
          },
          {
              "hash": "72ec861cb2f6c32a48632407f3aa1b05d5ad450ef75fa7660dd39d7be6d3ab68",
              "received": false,
              "removed": true
          }
      ],
      "subscription": "0xf90906914486a9c22d620e50022b38d5"
    }
}
```
```json tab:Vmlogs
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "vmlog": {
                  "topics": [
                      "aa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb",
                      "000000000000000000000000bb6ad02107a4422d6a324fd2e3707ad53cfed935"
                  ],
                  "data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAo="
              },
              "accountBlockHash": "de8cd1dc188fd4bf44c0cb90958ffbcccab5766840d56f7b35443a1a1c5c9d3e",
              "accountBlockHeight": "10",
              "address": "vite_78926f48bccef67a3b9cc64fdfe864f2a708ebce1d0bbe9aef",
              "removed": false
          }
      ],
      "subscription": "0xf90906914486a9c22d620e50022b38d5"
    }
}
```
:::

## subscribe_createSnapshotBlockSubscription
Start listening for new snapshot blocks. New blocks will be returned in callback
  
- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
  - `SnapshotBlocks` 
    * `subscription`: `string`  Subscription id
    * `result`: `Array<SnapshotBlockMessage>`
      * `hash`: `string hash` Hash of snapshot block
      * `height`: `string uint64` Height of snapshot block
      * `removed`: `bool` If `true`, the snapshot block was rolled back. For new snapshot block, `false` is filled

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["createSnapshotBlockSubscription"]
}
```
```json tab:Response
{
  "jsonrpc":"2.0",
  "id":1,
  "result":"0xa809145803ebb2a52229aefcbd52a99d"
}
```
```json tab:Callback
{
  "jsonrpc":"2.0",
  "method":"subscribe_subscription",
  "params":{
    "subscription":"0xa809145803ebb2a52229aefcbd52a99d",
    "result":[{"hash":"22c38acb79e2de0de3c5a09618054b93ac7c7e82f41f9e15d754e58694eefe16","height":"250","removed":false}]
  }
}
```
:::

## subscribe_createAccountBlockSubscription
Start listening for new account blocks. New blocks will be returned in callback

- **Parameters**: 
  null
  
- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
  - `AccountBlocks` 
    * `subscription`: `string` filterId
    * `result`: `Array<AccountBlockMessage>`
      * `hash`: `string hash` Hash of account block
      * `removed`: `bool` If `true`, the account block was rolled back. For new account block, `false` is filled

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["createAccountBlockSubscription"]
}
```
```json tab:Response
{
  "jsonrpc":"2.0",
  "id":1,
  "result":"0xa809145803ebb2a52229aefcbd52a99d"
}
```
```json tab:Callback
{
  "jsonrpc":"2.0",
  "method":"subscribe_subscription",
  "params":{
    "subscription":"0xa809145803ebb2a52229aefcbd52a99d",
    "result":[{
      "hash":"20009ee78d5f77122d215c3021f839b4024e4f2701e57bdb574e0cae1ae44e6c",
      "removed":false
    }]
  }
}
```
:::

## subscribe_createAccountBlockSubscriptionByAddress
Start listening for new account blocks on specified account. New blocks will be returned in callback

- **Parameters**:
  * `string address` Address of account

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
  - `AccountBlocksWithHeight`
    * `subscription`: `string` filterId
    * `result`: `Array<AccountBlockWithHeightMessage>`
      * `hash`: `string hash` Hash of account block
      * `height`: `string height` Height of account block
      * `removed`: `bool` If `true`, the account block was rolled back. For new account block, `false` is filled

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["createAccountBlockSubscriptionByAddress", "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
}
```
```json tab:Response
{
  "jsonrpc":"2.0",
  "id":1,
  "result":"0xa809145803ebb2a52229aefcbd52a99d"
}
```
```json tab:Callback
{
  "jsonrpc":"2.0",
  "method":"subscribe_subscription",
  "params":{
    "subscription":"0xa809145803ebb2a52229aefcbd52a99d",
    "result":[{
      "hash":"20009ee78d5f77122d215c3021f839b4024e4f2701e57bdb574e0cae1ae44e6c",
      "height":"1",
      "removed":false
    }]
  }
}
```
:::

## subscribe_createUnreceivedBlockSubscriptionByAddress
Start listening for unreceived transactions on specified account. Unreceived blocks will be returned in callback

- **Parameters**:
  * `address` Address of account

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
  - `UnreceivedBlocks` 
    * `subscription`: `string` filterId
    * `result`: `Array<UnreceivedBlockMessage>`
      * `hash`: `string hash` Hash of account block
      * `received`: `bool` If `true`, the transaction was received
      * `removed`: `bool` If `true`, the unreceived transaction was rolled back. In the case of `false`, if `received` is `false`, this is new unreceived block, otherwise (`received` is `true`) the transaction has been received. 

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["createUnreceivedBlockSubscriptionByAddress", "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
}
```
```json tab:Response
{
  "jsonrpc":"2.0",
  "id":1,
  "result":"0xa809145803ebb2a52229aefcbd52a99d"
}
```
```json tab:Callback
{
  "jsonrpc":"2.0",
  "method":"subscribe_subscription",
  "params":{
    "subscription":"0xa809145803ebb2a52229aefcbd52a99d",
    "result":[{
      "hash":"20009ee78d5f77122d215c3021f839b4024e4f2701e57bdb574e0cae1ae44e6c",
      "received":false,
      "removed":false
    }]
  }
}
```
:::

## subscribe_createVmlogSubscription
Start listening for new logs. New logs will be returned in callback

- **Parameters**:
  * `FilterParam`
    * `addressHeightRange`: `map[Address]Range` Query logs of the specified account address with given range. Address and height range should be filled in the map. At least one address must be specified.
      * `fromHeight`: `uint64` Start height. `0` means starting from the latest block
      * `toHeight`: `uint64` End height. `0` means no specific ending block
    * `topics`: `[][]Hash` Prefix of topics. See below example for reference

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
  - `Vmlogs` 
    * `subscription`: `string` filterId
    * `result`: `Array<VmlogMessage>`
      * `accountBlockHash`: `Hash` Hash of account block
      * `accountBlockHeight`: `uint64` Height of account block
      * `address`: `Address` Address of account
      * `vmlog`: `VmLog` Event log of smart contract
        * `topics`: `Array<string hash>` Event signature and indexed field. The signature can be generated from ABI
        * `data`: `string base64` Non-indexed field of event, can be decoded based on ABI
      * `removed`: `bool` If `true`, the log has been rolled back

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": [
    "createVmlogSubscription",
    {
      "addressHeightRange":{
        "vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f":{
          "fromHeight":"0",
          "toHeight":"0"
        }
      }
    }
  ]
}
```
```json tab:Response
{
  "jsonrpc":"2.0",
  "id":1,
  "result":"0x4b97e0674a5ebef942dbb07709c4a608"
}
```
```json tab:Callback
{
  "jsonrpc":"2.0",
  "method":"subscribe_subscription",
  "params":{
    "subscription":"0x4b97e0674a5ebef942dbb07709c4a608",
    "result":[
      {
        "vmlog":{
          "topics":[
            "aa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb",
            "000000000000000000000000bb6ad02107a4422d6a324fd2e3707ad53cfed935"
          ],
          "data":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAo="
        },
        "accountBlockHash":"23ea04b0dea4b9d0aa4d1f84b246b298a30faba753fa48303ad2deb29cd27f40",
        "accountBlockHeight": "10",
        "address":"vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f",
        "removed":false
      }
    ]
  }
}
```
:::


