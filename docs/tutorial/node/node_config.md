---
order: 4
---

# Node Config

## node_config.json

``` javascript
{
  "NetID": 1, // Vite network ID. The field is used to connect different network. Vite Pre-Mainnet -> 1. TestNet -> 2. 
  "Identity": "vite-node-name",  // Node name
  "MaxPeers": 10, // Maximum peers connected. No need to change
  "MaxPendingPeers": 10, // Maximum peers waiting to connect. No need to change
  "BootSeed": [
    "https://bootnodes.vite.net/bootmainnet.json" // URL for a list of bootstrap nodes
  ], 
  "Port": 8483, // UDP/TCP port. Default is 8483. Please ensure the port is not occupied by other process
  "RPCEnabled": true, // Enable RPC access. Default is enabled. 
  "HttpHost": "0.0.0.0", // Http listening address. Do not change unless you must specify a particular network address.
  "HttpPort": 48132, // Http listening port. Default is 48132.
  "WSEnabled": true, // Enable websocket access，Default is enabled. 
  "WSHost": "0.0.0.0", // Websocket listening address. Do not change unless you must specify a particular network address.
  "WSPort": 41420, // Websocket listening port. Default is 41420.
  "IPCEnabled": true, // Enable local command line console
  "PublicModules":[
    "ledger",
    "public_onroad",
    "net",
    "contract",
    "pledge",
    "register",
    "vote",
    "mintage",
    "consensusGroup",
    "tx",
    "debug",
    "dashboard",  // Full node statistics
    "sbpstats"
  ], // A list of gvite modules accessable in RPC, no need to change
  "Miner": true, // Enable mining. This field must be set to true for SBP node, otherwise it can be turned off
  "CoinBase": "0:vite_d2fef1e5ffa7d9139bd7c80a672e0530789bac6c7c9ff58dc6", // SBP block producing address in format of index:address
  "EntropyStorePath": "vite_d2fef1e5ffa7d9139bd7c80a672e0530789bac6c7c9ff58dc6", // The name of keystore file. Must conform to above coinbase address and should be stored in wallet directory.
  "EntropyStorePassword": "", // Keystore password
  "DashboardTargetURL":"wss://stats.vite.net",  // Full node statistics URL
  "RewardAddr":"vite_xxx",   // Full node reward receiving address
  "LogLevel": "info", // gvite log level. Default is info. No need to change.
  "VmLogAll": false, // true will save the vmlog of all contracts. By default, vmlog (ie event) is not saved
  "VmLogWhiteList": ["vite_bc68fb14f8a81015af1d28e6f88edff1e1db48473aca563a34"] // means that only the vmlog of the specified contract is saved.
}
```



