---
sidebarDepth: 4
---

# Subscribe

Vite provides two kinds of subscription API, **polling** and **callback**.

* **Polling API** creates a subscription filter and relies on the client program calls `subscribe_getChangesByFilterId` with the filter id to poll for new events. 
The filter will expire if it has not been used in 5 minutes or `subscribe_uninstallFilter` method is called explicitly.

Polling API includes 7 RPC methods:
- `subscribe_newSnapshotBlockFilter`
- `subscribe_newAccountBlockFilter`
- `subscribe_newAccountBlockByAddressFilter`
- `subscribe_newUnreceivedBlockByAddressFilter`
- `subscribe_newVmLogFilter`
- `subscribe_uninstallFilter`
- `subscribe_getChangesByFilterId`

* **Callback API** registers a subscription topic through WebSocket. Once the connection is set up, newly generated events will be returned in callback messages. 
This subscription will be cancelled when the WebSocket connection is closed.

Callback API includes one RPC method `subscribe_subscribe` and 5 topics:
- `newSnapshotBlock`
- `newAccountBlock`
- `newAccountBlockByAddress`
- `newUnreceivedBlockByAddress`
- `newVmLog`

Note that all events support revert. If an event was reverted, the `removed` field of the event is true.

:::tip Note
Add `"subscribe"` in `"PublicModules"` and set `"SubscribeEnabled":true` in node_config.json to enable the subscription API
:::

## Example

### Subscribe via Callback 

First, register a new subscription on address `vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f` by calling `subscribe_subscribe` method with topic parameter `newVmLog` and start listening to new vmlogs generated for the address.
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": [
    "newVmLog",
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
It will return a subscription id `0x4b97e0674a5ebef942dbb07709c4a608`
```json
{
  "jsonrpc":"2.0",
  "id":1,
  "result":"0x4b97e0674a5ebef942dbb07709c4a608"
}
```
When a new vmlog is generated, it will be sent as callback message
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

### Subscribe via Polling

First create a filter for address `vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f`
```json
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newVmLogFilter",
	"params": [{
      "addressHeightRange":{
        "vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f":{"fromHeight":"0","toHeight":"0"}
      }
	}]
}
```
This will return a filter id `0x61d780619649fb0872e1f94a40cec713`
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "0x61d780619649fb0872e1f94a40cec713"
}
```
The caller polls for new events by filter id
```json
{
	"jsonrpc": "2.0",
	"id": 2,
	"method": "subscribe_getChangesByFilterId",
	"params": ["0x61d780619649fb0872e1f94a40cec713"]
}
```
If a new vmlog was generated during the time, it will be returned
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
If `subscribe_getChangesByFilterId` is not be called in 5 minutes, the subscription filter will expire, and the caller has to register a new filter instead. 

The caller can also call `subscribe_uninstallFilter` to cancel a filter. 
```json
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_uninstallFilter",
	"params": ["0x61d780619649fb0872e1f94a40cec713"]
}
```

:::tip Fix Missing Blocks
A broken network connection may cause subscription accidentally closed. In this situation, the remedy is to use `ledger_getSnapshotBlocks`, `ledger_getAccountBlocks`, `ledger_getUnreceivedBlocksByAddress` and `ledger_getVmLogsByFilter` to fetch snapshot blocks, account blocks, unreceived blocks and vmlogs generated during the time.

As an example, to get the missing snapshot blocks, you should call `subscribe_newSnapshotBlockFilter` or `subscribe_subscribe` with `newSnapshotBlock` topic to resume subscription first, obtain the latest snapshot block height with `ledger_getLatestSnapshotBlock` method, then call `ledger_getSnapshotBlocks` to get the missing blocks.
:::

## subscribe_newSnapshotBlockFilter
Create a filter for new snapshot blocks.
  
- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newSnapshotBlockFilter",
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

## subscribe_newAccountBlockFilter
Create a filter for new transactions (account blocks) for all accounts

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newAccountBlockFilter",
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

## subscribe_newAccountBlockByAddressFilter
Create a filter for new transactions (account blocks) for a specified account.

- **Parameters**:
  * `string address`: Address of account

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newAccountBlockByAddressFilter",
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

## subscribe_newUnreceivedBlockByAddressFilter
Create a filter for new unreceived transactions for a specified account.

- **Parameters**:
  * `Address`: Address of account

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newUnreceivedBlockByAddressFilter",
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

## subscribe_newVmLogFilter
Create a filter for new vmlogs.

- **Parameters**:
  * `FilterParam`
    * `addressHeightRange`: `map[Address]Range` Return vmlogs of the specified account address with given range. Address and height range should be filled in the map. At least one address must be specified.
      * `fromHeight`: `uint64` Start height. `0` means starting from the latest block
      * `toHeight`: `uint64` End height. `0` means no ending height
    * `topics`: `[][]Hash` Prefix of topics. Refer to the following examples

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
	"method": "subscribe_newVmLogFilter",
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
Cancel subscription by removing the specified filter 

- **Parameters**:
  * `string`: filterId

- **Returns**:  
	- `bool` If `true`, the subscription is successfully cancelled

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
Poll for new events by filter. The return content depends on various subscription type. If no event is generated, an empty array will be returned.

- **Parameters**:
  * `string`: filterId

- **Returns (using `subscribe_newSnapshotBlockFilter`)**: 
  * `SnapshotBlocks`
    * `subscription`: `string` filterId
    * `result`: `Array<SnapshotBlockMessage>`
      * `hash`: `string hash` Hash of snapshot block
      * `height`: `string uint64` Height of snapshot block
      * `removed`: `bool` If `true`, the snapshot block was reverted. For new snapshot blocks, the field is `false`

- **Returns (using `subscribe_newAccountBlockFilter`)**: 
  * `AccountBlocks`
    * `subscription`: `string` filterId
    * `result`: `Array<AccountBlockMessage>`
      * `hash`: `string hash` Hash of account block
      * `removed`: `bool` If `true`, the account block was reverted. For new account block, the field is `false`

- **Returns (using `subscribe_newAccountBlockByAddressFilter`)**: 
  * `AccountBlocksWithHeight`
    * `subscription`: `string` filterId
    * `result`: `Array<AccountBlockWithHeightMessage>`
      * `hash`: `string hash` Hash of account block
      * `height`: `string height` Height of account block
      * `removed`: `bool` If `true`, the account block was reverted. For new account block, the field is `false`

- **Returns (using `subscribe_newUnreceivedBlockByAddressFilter`)**: 
  * `UnreceivedBlocks`
    * `subscription`: `string` filterId
    * `result`: `Array<UnreceivedBlockMessage>`
      * `hash`: `string hash` Hash of account block
      * `received`: `bool` If `true`, the transaction has been received
      * `removed`: `bool` If `true`, the unreceived transaction was reverted. For new unreceived blocks, both `removed` and `received` is `false`. 

- **Returns (using `subscribe_newVmLogFilter`)**:
  * `Vmlogs` 
    * `subscription`: `string` filterId
    * `result`: `Array<VmlogMessage>`
      * `accountBlockHash`: `Hash` Hash of account block
      * `accountBlockHeight`: `uint64` Height of account block
      * `address`: `Address` Address of account
      * `vmlog`: `VmLog` Event log of smart contract
        * `topics`: `Array<string hash>` Event signature and indexed field. The signature can be generated from ABI
        * `data`: `string base64` Non-indexed field of event, can be decoded based on ABI
      * `removed`: `bool` If `true`, the vmlog has been reverted

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
```json tab:UnreceivedTransactions
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

## subscribe_subscribe (Callback API)
### newSnapshotBlock
Start listening for new snapshot blocks. Newly generated snapshot blocks will be returned in callback
  
- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
  - `SnapshotBlocks` 
    * `subscription`: `string`  Subscription id
    * `result`: `Array<SnapshotBlockMessage>`
      * `hash`: `string hash` Hash of snapshot block
      * `height`: `string uint64` Height of snapshot block
      * `removed`: `bool` If `true`, the snapshot block was reverted. For new snapshot block, the field is `false`

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newSnapshotBlock"]
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

### newAccountBlock
Start listening for new account blocks. Newly generated account blocks will be returned in callback

- **Parameters**: 
  null
  
- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
  - `AccountBlocks` 
    * `subscription`: `string` filterId
    * `result`: `Array<AccountBlockMessage>`
      * `hash`: `string hash` Hash of account block
      * `removed`: `bool` If `true`, the account block was reverted. For new account block, the field is `false`

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newAccountBlock"]
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

### newAccountBlockByAddress
Start listening for new account blocks for the specified account. Newly generated account blocks will be returned in callback

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
      * `removed`: `bool` If `true`, the account block was reverted. For new account block, the field is `false`

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newAccountBlockByAddress", "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
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

### newUnreceivedBlockByAddress
Start listening for unreceived transactions for the specified account. New unreceived blocks will be returned in callback

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
      * `removed`: `bool` If `true`, the unreceived transaction was reverted. For new unreceived blocks, both `removed` and `received` is `false`. 

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newUnreceivedBlockByAddress", "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
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

### newVmLog
Start listening for new vmlogs. New event logs will be returned in callback

- **Parameters**:
  * `FilterParam`
    * `addressHeightRange`: `map[Address]Range` Return logs of the specified account address with given range. Address and height range should be filled in the map. At least one address must be specified.
      * `fromHeight`: `uint64` Start height. `0` means starting from the latest block
      * `toHeight`: `uint64` End height. `0` means no ending height
    * `topics`: `[][]Hash` Prefix of topics. 

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
      * `removed`: `bool` If `true`, the vmlog has been reverted

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": [
    "newVmLog",
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


