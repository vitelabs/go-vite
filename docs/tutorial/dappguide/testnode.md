---
order: 3
---
# Run a Local Dev Node

This section introduces how to manually set up a development node on your machine. However, to simplify the process, it's highly recommended you get a dev node with [Visual Studio Code Solidity++ Extension](../sppguide/introduction/installation.html#installing-the-visual-studio-code-extension).

:::tip

Please refer to [gvite installation guide](../node/install.html) before proceed.

:::

## Configure node_config.json

Configure 'node_config.json' file with following settings:
 * Set `"Single": "true"`, indicating this is a single node
 * Set `"RPCEnabled": "true"` to turn on RPC service and define the binding IP address and port in `HttpHost` and `HttpPort`. Defaults are `0.0.0.0` and `48132`
 * Set `"WSEnabled": "true"` and define the binding IP address and port in `WSHost` and `WSPort`. Defaults are `0.0.0.0` and `41420`
 * Set `"Miner": "true"`
 * Set mining address in `CoinBase` in format of 'index:address'. For example, `0:vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906`
 * Set coinbase's keystore path in `EntropyStorePath`, such as `vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906`
 * Set coinbase's keystore password in `EntropyStorePassword`, such as `123456`
 * Set genesis block file in `GenesisFile`. For example, `genesis.json` will read a file named 'genesis.json' in current folder
 * Set `"VmLogAll": "true"` to save the vmlog of all contracts
 * Set `"OpenPlugins": "true"` to enable statistics function. This includes the number of un-received transactions and transactions records listed by different token id

Following settings are specific to your environment:
 * Set data directory path in `DataDir`. For example, `gvite/singlemode` stands for directory `gvite/singlemode/devdata` under current folder
 * Set keystore directory path in `KeyStoreDir`. For example, `gvite/singlemode` stands for directory `gvite/singlemode/devdata/wallet` under current folder
 * Set `"VmTestEnabled": "true""` to turn on development mode. In dev mode, sending transaction won't verify sender's account balance or quota
 * Set test token ID in `TestTokenTti` and issuing account private key in `TestTokenHexPrivKey` for test token acquiring purpose. For example, `tti_5649544520544f4b454e6e40` and `7488b076b27aec48692230c88cbe904411007b71981057ea47d757c1e7f7ef24f4da4390a6e2618bec08053a86a6baf98830430cbefc078d978cf396e1c43e3a`
 * Set `"SubscribeEnabled": "true""` to enable event subscription
 * Set visible modules in `PublicModules`. Modules included will be accessible through remote RPC and/or WS calls. See [Module List](../../api/rpc/)
 
Below is a full example of node_config.json：
```json
{
  "NetID": 5,
  "Identity": "test",
  "MaxPeers": 200,
  "MaxPendingPeers": 20,
  "BootNodes": [
  ],
  "Port": 8483,
  "HttpVirtualHosts": [],
  "IPCEnabled": true,
  "TopoDisabled": true,
  "LogLevel": "info",
  "Single":true,
  "RPCEnabled": true,
  "HttpHost": "0.0.0.0",
  "HttpPort": 48132,
  "WSEnabled": true,
  "WSHost": "0.0.0.0",
  "WSPort": 41420,
  "Miner": true,
  "CoinBase": "0:vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906",
  "EntropyStorePath": "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906",
  "EntropyStorePassword": "123",
  "GenesisFile": "./genesis.json",
  "DataDir":"gvite/singlemode",
  "KeyStoreDir":"gvite/singlemode",
  "VMTestEnabled":true,
  "TestTokenTti":"tti_5649544520544f4b454e6e40",
  "TestTokenHexPrivKey":"7488b076b27aec48692230c88cbe904411007b71981057ea47d757c1e7f7ef24f4da4390a6e2618bec08053a86a6baf98830430cbefc078d978cf396e1c43e3a",
  "SubscribeEnabled":true,
  "VmLogAll":true,
  "OpenPlugins":true,
  "PublicModules": [
    "net",
    "wallet",
    "ledger",
    "contract",
    "dashboard",
    "debug",
    "util"
  ]
}
```
An example of coinbase's keystore file:
```json
{
	"primaryAddress": "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906",
	"crypto": {
		"ciphername": "aes-256-gcm",
		"ciphertext": "807e09cc3e9f48c1742b22096404d98ba61e5c892994242515d48eb84cbee5f2ed93c7805ec32adb259e166fcd62428b",
		"nonce": "41d76fee25bfa544b8212cf6",
		"kdf": "scrypt",
		"scryptparams": {
			"n": 262144,
			"r": 8,
			"p": 1,
			"keylen": 32,
			"salt": "64c5f11657f91680c53bf8b618416a2a4d0c8e46c2cc6dc753fd11c9fc77441c"
		}
	},
	"seedstoreversion": 1,
	"timestamp": 1544422238
}
```

## Configure genesis.json
See [Genesis Config](../node/genesis_config.html)

Below is a full example of genesis.json with following settings:
* Consensus group settings
  - 1 snapshot block is produced per second
  - 3 continuous blocks are produced by the same producer in a round
  - 2 snapshot block producers are selected in each round
* 2 initial block producers
* VITE token settings
* 2 initial staking accounts
* 4 initial accounts(including 2 built-in contracts)

```json
{
  "GenesisAccountAddress": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792",
  "ForkPoints":{
      "SeedFork":{
        "Height":1,
        "Version":1
      },
      "DexFork":{
        "Height":2,
        "Version":2
      },
      "DexFeeFork":{
        "Height":3,
        "Version":3
      },
      "StemFork":{
        "Height":4,
        "Version":4
      },
      "LeafFork":{
        "Height":5,
        "Version":5
      },
      "EarthFork":{
        "Height":6,
        "Version":6
      },
      "DexMiningFork":{
        "Height":7,
        "Version":7
      },
      "DexRobotFork":{
        "Height":8,
        "Version":8
      },
      "DexStableMarketFork":{
        "Height":9,
        "Version":9
      }
    },
  "ConsensusGroupInfo": {
    "ConsensusGroupInfoMap": {
      "00000000000000000001": {
        "NodeCount": 2,
        "Interval": 1,
        "PerCount": 3,
        "RandCount": 2,
        "RandRank": 100,
        "Repeat": 1,
        "CheckLevel": 0,
        "CountingTokenId": "tti_5649544520544f4b454e6e40",
        "RegisterConditionId": 1,
        "RegisterConditionParam": {
          "PledgeAmount": 100000000000000000000000,
          "PledgeToken": "tti_5649544520544f4b454e6e40",
          "PledgeHeight": 0
        },
        "VoteConditionId": 1,
        "VoteConditionParam": {},
        "Owner": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792",
        "PledgeAmount": 0,
        "WithdrawHeight": 1
      },
      "00000000000000000002": {
        "NodeCount": 2,
        "Interval": 3,
        "PerCount": 1,
        "RandCount": 2,
        "RandRank": 100,
        "Repeat": 48,
        "CheckLevel": 1,
        "CountingTokenId": "tti_5649544520544f4b454e6e40",
        "RegisterConditionId": 1,
        "RegisterConditionParam": {
          "PledgeAmount": 100000000000000000000000,
          "PledgeToken": "tti_5649544520544f4b454e6e40",
          "PledgeHeight": 7776000
        },
        "VoteConditionId": 1,
        "VoteConditionParam": {},
        "Owner": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792",
        "PledgeAmount": 0,
        "WithdrawHeight": 1
      }
    },
    "RegistrationInfoMap": {
      "00000000000000000001": {
        "s1": {
          "NodeAddr": "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906",
          "PledgeAddr": "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906",
          "Amount": 100000000000000000000000,
          "WithdrawHeight": 7776000,
          "RewardTime": 1,
          "CancelTime": 0,
          "HisAddrList": [
            "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906"
          ]
        },
        "s2": {
          "NodeAddr": "vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
          "PledgeAddr": "vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
          "Amount": 100000000000000000000000,
          "WithdrawHeight": 7776000,
          "RewardTime": 1,
          "CancelTime": 0,
          "HisAddrList": [
            "vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87"
          ]
        }
      }
    },
    "HisNameMap": {
      "00000000000000000001": {
        "vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87": "s2",
        "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906": "s1"
      }
    },
    "VoteStatusMap": {
      "00000000000000000001":{
        "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792":"s1",
        "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23":"s2"
      }
    }
  },
  "MintageInfo": {
    "TokenInfoMap": {
      "tti_5649544520544f4b454e6e40": {
        "TokenName": "Vite Token",
        "TokenSymbol": "VITE",
        "TotalSupply": 1000000000000000000000000000,
        "Decimals": 18,
        "Owner": "vite_00000000000000000000000000000000000000042d7ef71894",
        "PledgeAmount": 0,
        "PledgeAddr": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792",
        "WithdrawHeight": 0,
        "MaxSupply": 115792089237316195423570985008687907853269984665640564039457584007913129639935,
        "OwnerBurnOnly": false,
        "IsReIssuable": true
      }
    },
    "LogList": [
      {
        "Data": "",
        "Topics": [
          "3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6",
          "000000000000000000000000000000000000000000005649544520544f4b454e"
        ]
      }
    ]
  },
  "PledgeInfo": {
    "PledgeBeneficialMap": {
      "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792": 1000000000000000000000,
      "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": 1000000000000000000000
    },
    "PledgeInfoMap": {
      "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792": [
        {
          "Amount": 1000000000000000000000,
          "WithdrawHeight": 259200,
          "BeneficialAddr": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792"
        }
      ],
      "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": [
        {
          "Amount": 1000000000000000000000,
          "WithdrawHeight": 259200,
          "BeneficialAddr": "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23"
        }
      ]
    }
  },
  "AccountBalanceMap": {
    "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792": {
      "tti_5649544520544f4b454e6e40": 899798000000000000000000000
    },
    "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": {
      "tti_5649544520544f4b454e6e40": 100000000000000000000000000
    },
    "vite_00000000000000000000000000000000000000042d7ef71894":{
      "tti_5649544520544f4b454e6e40": 200000000000000000000000
    },
    "vite_000000000000000000000000000000000000000309508ba646":{
      "tti_5649544520544f4b454e6e40": 2000000000000000000000
    }
  }
}
```

This will generate a genesis block of Type 7 in genesis account with corresponding states(balance, contract status, etc.) during first launch. 
:::tip
Whenever you need to work with a different genesis setting, you should change genesis.json file, delete `ledger` folder in your `DataDir` directory and restart gvite process.
:::

## Start Dev Node

* Make sure your node_config.json, genesis.json and gvite executable are in the same directory.
* Create sub-folder `wallet` in your `KeyStoreDir` directory and place coinbase's keystore file `vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906` inside.
* Start gvite and verify it launches successfully.

