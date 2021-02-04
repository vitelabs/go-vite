# Net

:::tip Maintainer
[jerry-vite](https://github.com/jerry-vite)
:::

## net_syncInfo
Return sync status of the node

- **Parameters**: `none`

- **Returns**: 

`SyncInfo`
  -  `from` : `string` Sync start height 
  -  `to` : `string` Sync target height
  -  `current` : `string` Current snapshot chain height
  -  `state` : `uint` Sync state: 0 - not start, 1 - syncing, 2 - complete, 3 - error, 4 - cancelled
  -  `status` : `string` Sync state description

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 2,
	"method": "net_syncInfo",
	"params": null
}
```

```json tab:Response
{
	"jsonrpc": "2.0",
	"id": 2,
	"result": {
    "from": "0",
    "to": "30000",
    "current": "0",
    "state": 1,
    "status": "Synchronising"
  }
}
```
```json test
{
	"jsonrpc": "2.0",
	"id": 2,
	"method": "net_syncInfo",
	"params": null
}
```
:::

## net_syncDetail
synchron detail

- **Parameters**: `none`

- **Returns**: 

    `SyncDetail`
     -  `from` : `string` start height
     -  `to` : `string` target height
     -  `current` : `string` current snapshot chain height
     -  `status` : `string` synchron status description
     -  `tasks` : `array` download tasks
     -  `Connections` : `array` network connections to download ledger chunks
     -  `Chunks` : `array` downloaded ledger chunks have been verified, wait to insert into local chain
     -  `Caches` : `array` downloaded ledger chunks, but have not verified.

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 2,
	"method": "net_syncDetail",
	"params": null
}
```

```json tab:Response
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "from": 692001,
    "to": 727000,
    "current": 692084,
    "state": "Synchronising",
    "status": "Synchronising",
      "tasks": [
        "692001-693000 done",
        "693001-694000 pending"
      ],
      "connections": [
        {
          "address": "24a160122317e6e4940ef2a91242b07f@118.25.49.80:8484",
          "speed": "0.00 Byte/s",
        },
        {
          "address": "04508fbe0512c01a48c8a450425547de@118.24.112.185:8484",
          "speed": "0.00 Byte/s",
        }
      ],
      "chunks": [
        [
          {
            "height": 692000,
            "hash": "74add6f7f61c33dd741276d97d8ade4456c47485da78752587aef8a209fe7e88"
          },
          {
            "height": 693000,
            "hash": "b6af1c6fb3b502268b17928b1d91206e71003f614c134f8865bf6886d88d8e30"
          }
        ]
      ],
      "caches": [
          {
            "Bound": [
              691001, 693000
            ],
            "Hash": "b6af1c6fb3b502268b17928b1d91206e71003f614c134f8865bf6886d88d8e30",
            "PrevHash": "028e2730c8fad34b53b8d8f5a41881024fa8b87a97a9cfc61f0e0c83984336a0"
          },
          {
            "Bound": [
              694001, 696000
            ],
            "Hash": "e14662c3e9a9751b28822f2640be79cf13bd778c6b3158c8f6ff584fbf89fa24",
            "PrevHash": "c6cd65717345f017309ee961a6bcda9ba021e0ed5913d8111471ff09fc95590c"
          },
          {
            "Bound": [
              698001, 699000
            ],
            "Hash": "4e067e54b9e966b264053c271c7d976065b8a0796d6995b9dda45e11339e0b57",
            "PrevHash": "26a6bdcfd3d9f58951eb76ebd4784160e98098564cc31e236618f045cb90f365"
          }
      ]
    }
}
```
```json test
{
	"jsonrpc": "2.0",
	"id": 2,
	"method": "net_syncDetail",
	"params": null
}
```
:::


## net_nodeInfo
Return the detailed information of current Node, like peers.

- **Parameters**: `none`

- **Returns**: 

`NodeInfo`
  -  `id` : `string` Node ID
  -  `name` : `string` Node name, configured in `Identity` field of `node_config.json`
  -  `netId` : `int`  The ID of Vite network connected
  -  `peerCount` : `int` Number of peers connected
  -  `peers` : `[]*PeerInfo` Information of peers connected

- **Example**: 

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 3,
	"method": "net_nodeInfo",
	"params": null
}
```

```json tab:Response
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "name": "vite-sbp-0",
    "netId": 1,
    "address": "0.0.0.0:8483",
    "peerCount": 2,
    "peers": [
      {
        "name": "vite-sbp-1",
        "height": 726575,
        "address": "119.28.221.175:8483",
        "createAt": "2019-05-31 03:05:09"
      },
      {
        "name": "vite-full-node",
        "height": 726576,
        "address": "117.50.66.76:8483",
        "createAt": "2019-05-31 12:33:44"
      },
    ]
  }
}
```

```json test
{
	"jsonrpc": "2.0",
	"id": 3,
	"method": "net_nodeInfo",
	"params": null
}
```
:::

`PeerInfo`
 -  `name` : `string` Peer's name
 -  `height` : `int` Current snapshot chain height
 -  `address` : `string` Peer's ip address
 -  `createAt` : `string` The time when this peer connected


## net_peers

the same with method `net_nodeInfo`
