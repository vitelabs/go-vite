---
sidebarDepth: 4
---

# Subscribe

:::tip Maintainer
[viteLiz](https://github.com/viteLiz)
:::

## Event Subscription 

To address different scenarios, subscription through **polling** and **callback** are both provided in Vite as two options.

Polling API requires client program to create a filter first, then polls `subscribe_getFilterChanges` method by `filterId` for new events. 
Remember, filter will expire if it has not been used for more than 5 minutes, in this case you should create a new filter for further subscription. 
You can also manually un-subscribe by calling `subscribe_uninstallFilter` method.

`subscribe_newSnapshotBlocksFilter`, `subscribe_newAccountBlocksFilter`, `subscribe_newAccountBlocksByAddrFilter`, `subscribe_newOnroadBlocksByAddrFilter`, `subscribe_newLogsFilter`, `subscribe_uninstallFilter` and `subscribe_getFilterChanges` are polling APIs.

Callback API registers new subscription through WebSocket connection. Afterwards, subscribed events will be returned in callbacks when they are generated. 
This kind of subscription will close automatically when the WebSocket connection is closed.

`subscribe_newSnapshotBlocks`, `subscribe_newAccountBlocks`, `subscribe_newAccountBlocksByAddr`, `subscribe_newOnroadBlocksByAddr` and `subscribe_newLogs` are callback APIs.

5 kinds of events are supported: new snapshot event, new transaction event, new transaction event on certain account, un-received transaction event on certain account and new log event. 
Event supports rollback. When rollback occurs, `removed` field in the event content is set to true.

:::tip To enable event subscription, you should
* Install a minimum 2.1.4 version of 'gvite' software
* Add `"subscribe"` into `"PublicModules"` and set `"SubscribeEnabled":true` in node_config.json
:::

## Usage Example

### Subscribe via Callback 

First, register a new subscription on address `vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f` by calling `subscribe_subscribe` method with name `newLogs`.
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": [
    "newLogs",
    {
      "addrRange":{
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
        "log":{
          "topics":[
            "aa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb",
            "000000000000000000000000bb6ad02107a4422d6a324fd2e3707ad53cfed935"
          ],
          "data":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAo="
        },
        "accountBlockHash":"23ea04b0dea4b9d0aa4d1f84b246b298a30faba753fa48303ad2deb29cd27f40",
        "accountHeight": "10",
        "addr":"vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f",
        "removed":false
      }
    ]
  }
}
```
Subscription under callback is not necessarily cancelled explicitly, but will end once the connection is closed.

### Subscribe via Polling

Create a log filter on address `vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f`
```json
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newLogsFilter",
	"params": [{
      "addrRange":{
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
Now poll for new events periodically by filter id
```json
{
	"jsonrpc": "2.0",
	"id": 2,
	"method": "subscribe_getFilterChanges",
	"params": ["0x61d780619649fb0872e1f94a40cec713"]
}
```
New logs generated after last poll will be returned
```json
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "result": [
            {
                "log": {
                    "topics": [
                        "96a65b1cd08da045d0318cafda7b8c8436092851d5a4b7e75054c005a296e3fb",
                        "0000000000000000000000ab24ef68b84e642c0ddca06beec81c9acb1977bb00"
                    ],
                    "data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHsAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF"
                },
                "accountBlockHash": "802b82821ec52bdadb8b79a53363bf2f90645caef95a83c34af20c640a6c320b",
                "addr": "vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f",
                "removed": false
            }
        ],
        "subscription": "0x61d780619649fb0872e1f94a40cec713"
    }
}
```
You should cancel subscription by calling `subscribe_uninstallFilter` explicitly under polling mode
```json
{
	"jsonrpc": "2.0",
	"id": 17,
	"method": "subscribe_uninstallFilter",
	"params": ["0x61d780619649fb0872e1f94a40cec713"]
}
```

### Log Completion

Accidental connection interrupt may cause subscription closed. Under this situation, historical snapshot blocks, account blocks or un-received blocks can be fetched by `ledger_getSnapshotBlocks`, `ledger_getBlocksByHeight` or `onroad_getOnroadBlocksByAddress`, while historical logs can be retrieved by `subscribe_getLogs` method.

For example, in order to re-subscribe to new snapshot event, you should call `subscribe_newSnapshotBlocks` to resume subscription first, then obtain latest snapshot block height by `ledger_getSnapshotChainHeight`, finally call `ledger_getSnapshotBlocks` to get all snapshot blocks missed.


## subscribe_newSnapshotBlocksFilter
Create a filter for polling for new snapshot blocks by passing into `subscribe_getFilterChanges` as parameter

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newSnapshotBlocksFilter",
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

## subscribe_newAccountBlocksFilter
Create a filter for polling for new transactions on all accounts by passing into `subscribe_getFilterChanges` as parameter

- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newAccountBlocksFilter",
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

## subscribe_newAccountBlocksByAddrFilter
Create a filter for polling for new transactions on specified account by passing into `subscribe_getFilterChanges` as parameter

- **Parameters**:
  * `string address`: Address of account
  
- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newAccountBlocksByAddrFilter",
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

## subscribe_newOnroadBlocksByAddrFilter
Create a filter for polling for un-received transactions on specified account by passing into `subscribe_getFilterChanges` as parameter. 
The events include new un-received transaction, transaction is received and un-received transaction is rolled back.

- **Parameters**:
  * `string address`: Address of account
  
- **Returns**:  
	- `string` filterId

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newOnroadBlocksByAddrFilter",
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

## subscribe_newLogsFilter
Create a filter for polling for new logs by passing into `subscribe_getFilterChanges` as parameter
  
- **Parameters**:

  * `FilterParam`
    1. `addrRange`: `map[Address]Range` Query logs of the specified account address with given query range. Addresses and height ranges are filled in a map. At least one address must be specified.
    2. `topics`: `[][]Hash` Subscribed topics, see below example for usage.
    
`Range`
  1. `fromHeight`: `uint64` Start height. 0 means starting from the first block
  2. `toHeight`: `uint64` End height. 0 means the latest block

```
Topic examples：
 {} matches all logs
 {{A}} matches the logs having "A" as the first element
 {{},{B}} matches the logs having "B" as the second element
 {{A},{B}} matches the logs having "A" as the first element and "B" as the second element
 {{A,B},{C,D}} matches the logs having "A" or "B" as the first element, and "C" or "D" as the second element
```
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_newLogsFilter",
	"params": [{
		"addrRange":{
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
Cancel event polling by removing specified filter 

- **Parameters**:
  * `string`: filterId

- **Returns**:  
	- `bool` result

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

## subscribe_getFilterChanges
Poll for new events. The content of return value depends on the specific subscription. If no event is generated since last poll, an empty array will be returned.

- **Parameters**:
  * `string`: filterId
    
- **Return in `subscribe_newSnapshotBlocksFilter`**: 
  * `subscription`: `string` filterId
  * `result`: `Array<SnapshotBlocksMsg>`
    1. `hash`: `Hash` The hash of snapshot block
    2. `heightStr`: `string` The height of snapshot block
    3. `removed`: `bool` Whether the block was rolled back. New transaction will be marked as `false`
  
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_getFilterChanges",
	"params": ["0xf90906914486a9c22d620e50022b38d5"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "hash": "5d47f2e0a532923f7ee53e7b465381f197a669e86155d541b3b7f3d63f07a0e2",
              "heightStr": "124",
              "removed": false
          },
          {
              "hash": "78b19cb84ac293d4af3f36e741938929f6d3311362e1265e87bbaa74e5fcef09",
              "heightStr": "125",
              "removed": false
          },
          {
              "hash": "94437996b3e70afd5d8593e2020ae56f63dbbc538df1ead1633340393bd52c1a",
              "heightStr": "126",
              "removed": false
          }
      ],
      "subscription": "0xf90906914486a9c22d620e50022b38d5"
    }
}
```
:::  
  
- **Return in `subscribe_newAccountBlocksFilter`**: 
  * `subscription`: `string` filterId
  * `result`: `Array<AccountBlocksMsg>`
    1. `hash`: `Hash` The hash of account block
    2. `removed`: `bool` Whether the block was rolled back. New transaction will be marked as `false`
  
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_getFilterChanges",
	"params": ["0xf90906914486a9c22d620e50022b38d5"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "Hash": "9cc9ba996a4192e35ddbfe3ba448611fc06f6342463e21d3300e58e9772b348f",
              "Removed": false
          },
          {
              "Hash": "8b9f8067ef09aa09c8f9d652f0d9ac5e99d083722089a6d91323cffd8b2dcf08",
              "Removed": false
          }
      ],
      "subscription": "0xf90906914486a9c22d620e50022b38d5"
    }
}
```
:::

- **Return in `subscribe_newAccountBlocksByAddrFilter`**: 
  * `subscription`: `string` filterId
  * `result`: `Array<AccountBlocksWithHeightMsg>`
    1. `hash`: `Hash` The hash of account block
    2. `heightStr`: `string` The height of account block
    3. `removed`: `bool` Whether the block was rolled back. New transaction will be marked as `false`
  
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_getFilterChanges",
	"params": ["0x4f18a08c6e6801aeb7a8cfbad0ca90af"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "hash": "72ec861cb2f6c32a48632407f3aa1b05d5ad450ef75fa7660dd39d7be6d3ab68",
              "heightStr": "15",
              "removed": false
          },
          {
              "hash": "9d0df2fbc311ceb232e851a758e88fcc0a9f16d7a4240c2aa486f26f1b36d8f2",
              "heightStr": "16",
              "removed": false
          },
          {
              "hash": "18914060ba6fe9474b4c724dfe3ff5999d9cb90b5128222ade210d11fe3216f0",
              "heightStr": "17",
              "removed": false
          }
      ],
      "subscription": "0x4f18a08c6e6801aeb7a8cfbad0ca90af"
    }
}
```
:::

- **Return in `subscribe_newOnroadBlocksByAddrFilter`**: 
  * `subscription`: `string` filterId
  * `result`: `Array<OnroadMsg>`
    1. `hash`: `Hash` The hash of account block
    2. `closed`: `bool` Whether the transaction was received
    3. `removed`: `bool` Whether the un-received transaction was rolled back
  
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_getFilterChanges",
	"params": ["0x64e1eb3d26517a0d736b3d85ae9ce299"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "hash": "72ec861cb2f6c32a48632407f3aa1b05d5ad450ef75fa7660dd39d7be6d3ab68",
              "closed": false,
              "removed": false
          },
          {
              "hash": "72ec861cb2f6c32a48632407f3aa1b05d5ad450ef75fa7660dd39d7be6d3ab68",
              "closed": true,
              "removed": false
          },
          {
              "hash": "18914060ba6fe9474b4c724dfe3ff5999d9cb90b5128222ade210d11fe3216f0",
              "closed": false,
              "removed": true
          }
      ],
      "subscription": "0x64e1eb3d26517a0d736b3d85ae9ce299"
    }
}
```
:::

- **Return in `subscribe_newLogsFilter`**: 
  * `subscription`: `string` filterId
  * `result`: `Array<LogsMsg>`
    1. `accountBlockHash`: `Hash` The hash of account block
    2. `addr`: `Address` Account address
    3. `log`: `VmLog` Log
    4. `removed`: `bool` Whether the log was rolled back. New log will be marked as `false`

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "subscribe_getFilterChanges",
	"params": ["0x8f34ddeb22b87fdfd2acb6c9f5a2b50d"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
      "result": [
          {
              "log": {
                  "topics": [
                      "aa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb",
                      "000000000000000000000000bb6ad02107a4422d6a324fd2e3707ad53cfed935"
                  ],
                  "data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAo="
              },
              "accountBlockHash": "de8cd1dc188fd4bf44c0cb90958ffbcccab5766840d56f7b35443a1a1c5c9d3e",
              "addr": "vite_78926f48bccef67a3b9cc64fdfe864f2a708ebce1d0bbe9aef",
              "removed": false
          }
      ],
      "subscription": "0x8f34ddeb22b87fdfd2acb6c9f5a2b50d"
    }
}
```
:::

## subscribe_newSnapshotBlocks
Start listening for new snapshot blocks via callback

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
`Object`
  * `subscription`: `string`  Subscription id
  * `result`: `Array<SnapshotBlocksMsg>`
      1. `hash`: `Hash` The hash of snapshot block
      2. `height`: `uint64` The height of snapshot block
      3. `removed`: `bool` Whether the block was rolled back. New transaction will be marked as `false`

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newSnapshotBlocks"]
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
    "result":[{"hash":"22c38acb79e2de0de3c5a09618054b93ac7c7e82f41f9e15d754e58694eefe16","heightStr":"250","removed":false}]
  }
}
```
:::

## subscribe_newAccountBlocks
Start listening for new transactions via callback

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
`Object`
  * `subscription`: `string`  Subscription id
  * `result`: `Array<AccountBlocksMsg>`
     1. `hash`: `Hash` The hash of account block
     2. `removed`: `bool` Whether the block was rolled back. New transaction will be marked as `false`

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newAccountBlocks"]
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

## subscribe_newAccountBlocksByAddr
Start listening for new transactions on specified account via callback

- **Parameters**:

  * `address` Account address

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
`Object`
  * `subscription`: `string`  Subscription id
  * `result`: `Array<AccountBlocksWithHeightMsg>`
    1. `hash`: `Hash` The hash of account block
    2. `heightStr`: `string` The height of account block
    3. `removed`: `bool` Whether the block was rolled back. New transaction will be marked as `false`

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newAccountBlocksByAddr", "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
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
      "heightStr":"1",
      "removed":false
    }]
  }
}
```
:::

## subscribe_newOnroadBlocksByAddr
Start listening for un-received transactions on specified account via callback

- **Parameters**:

  * `address` Account address

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
`Object`
  * `subscription`: `string`  Subscription id
  * `result`: `Array<OnroadMsg>`
    1. `hash`: `Hash` The hash of account block
    2. `closed`: `bool` Whether the transaction was received
    3. `removed`: `bool` Whether the un-received transaction was rolled back

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": ["newOnroadBlocksByAddr", "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"]
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
      "closed":false,
      "removed":false
    }]
  }
}
```
:::

## subscribe_newLogs
Start listening for new logs via callback
  
- **Parameters**:

  * `FilterParam`
    1. `addrRange`: `map[Address]Range` Query logs of the specified account address with given query range. Addresses and height ranges are filled in a map. At least one address must be specified.
    2. `topics`: `[][]Hash` Subscribed topics, see example in `subscribe_newLogsFilter` for usage.
    
`Range`
  1. `fromHeight`: `uint64` Start height. 0 means starting from the first block
  2. `toHeight`: `uint64` End height. 0 means the latest block

- **Returns**:  
	- `string` Subscription id
	
- **Callback**:  
`Object`
  * `subscription`: `string`  Subscription id
  * `result`: `Array<LogsMsg>`
    1. `accountBlockHash`: `Hash` The hash of account block
    2. `addr`: `Address` Account address
    3. `log`: `VmLog` Log
    4. `removed`: `bool` Whether the log was rolled back. New log will be marked as `false`

::: demo
```json tab:Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "subscribe_subscribe",
  "params": [
    "newLogs",
    {
      "addrRange":{
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
        "log":{
          "topics":[
            "aa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb",
            "000000000000000000000000bb6ad02107a4422d6a324fd2e3707ad53cfed935"
          ],
          "data":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAo="
        },
        "accountBlockHash":"23ea04b0dea4b9d0aa4d1f84b246b298a30faba753fa48303ad2deb29cd27f40",
        "addr":"vite_f48f811a1800d9bde268e3d2eacdc4b4f8b9110e017bd7a76f",
        "removed":false
      }
    ]
  }
}
```
:::

## subscribe_getLogs
Return historical logs

- **Parameters**: 

  * `FilterParam`
    1. `addrRange`: `map[Address]Range` Query logs of the specified account address with given query range. Addresses and height ranges are filled in a map. At least one address must be specified.
    2. `topics`: `[][]Hash` Subscribed topics, see example in `subscribe_newLogsFilter` for usage.
    
`Range`
  1. `fromHeight`: `uint64` Start height. 0 means starting from the first block
  2. `toHeight`: `uint64` End height. 0 means the latest block

- **Returns**:  
  * `result`: `Array<LogsMsg>`
    1. `accountBlockHash`: `Hash` The hash of account block
    2. `addr`: `Address` Account address
    3. `log`: `VmLog` Log
    4. `removed`: `bool` Whether the log was rolled back. New log will be marked as `false`
	
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 17,
	"method": "subscribe_getLogs",
	"params": [{
		"addrRange":{
			"vite_8810e12ec2d4d61e7568cac25ebd5dd44735d36a405b94f1fa":{
				"fromHeight":"1",
				"toHeight":"10"
			}
		}
	}]
}
```
```json tab:Response
{
  "jsonrpc":"2.0",
  "id":1,
  "result": [
    {
      "log": {
      "topics": [
        "28e6ea56797f4a1b22a1d1986cb6c22e80099ba8e4fd14d42accfaedfe5f6640"
      ],
      "data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGQQurTFV9WklB2DRvsX8wLCgyoVomYHSCebb9Br/hQ+RAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABwYLIcJLnbQjGl+qeU7YWlTWwfsoF6mescP5xz2fDTEg="
    },
    "accountBlockHash": "e4917f357a4588ec1752797ee5516939f46078f5356b14422d4a9dfe45f88bf5",
    "addr": "vite_8810e12ec2d4d61e7568cac25ebd5dd44735d36a405b94f1fa",
    "removed": false
    }
  ]
}
```
:::

