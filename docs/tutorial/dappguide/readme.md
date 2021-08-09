---
order: 1
parent:
    order: 3
    title: Vite dApp Development Guide
---

# Get Started

## How dApp Works in Vite
![](./dapp-architecture.png)

Above diagram shows how dApp works in Vite. In general, a dApp consists of one (or several) smart contract and companion web application. dApp communicates to a full node (private or non-private) via HTTP or WebSocket connection, fetches data from blockchain and then displays on the webpage. Usually dApp doesn't manage private keys and addresses. Instead, a dApp establishes a connection to Vite wallet (where mnemonic seed and private keys are managed) for sending/receiving transactions, calling smart contract deployed on blockchain, etc. 
To set up the connection between dApp and Vite wallet, at present there are three options.

1. Scan a QR code on the dApp by using Vite wallet for every transaction;
2. Integrate the dApp into Vite wallet;
3. **Use ViteConnect**. 

The first option is not good enough because you have to scan QR code each time you send transaction or call smart contract. It only applies to very simple dApp where user should only talk to smart contract once! A simple voting program could be an example of this case. 

Integrating dApp into Vite wallet is a good choice once you got approval from Vite team, so that you have a place for your dApp in Vite wallet! So far so good! However, you need request first and there is no guarantee that it must be approved. 

We recommend ViteConnect. ViteConnect establishes WebSocket connection between dApp and Vite wallet. More safe, and more convenient. User just need scan QR code once and the subsequent transactions can be auto-signed (if you turn on the switch). For how to incorporate ViteConnect into your dApp, see [Vite Connect SDK](https://github.com/vitelabs/vite-connect-client). 

## dApp Release Process
 
* Complete dApp (smart contract and companion web application) development and testing;
* Set up a full node that provides both HTTP and WebSocket RPC services;
* Deploy smart contract on blockchain and stake VITE coins for the contract's account to make sure it has enough quota;
* Deploy companion web application;
* Test dApp's functionalities to make sure it works well. If it should integrate with Vite wallet, test in [Vite Test Wallet](./testdapp.html).

## Before Development

### Run Local Dev Node

See [Run Local Development Node](./testnode.html) to install your local development node.

### Install Vite.js

Install the latest release of [Vite.js](../../api/vitejs/README.md). Vite.js is the Javascript SDK provided by Vite team. 

### Download `solppc`

Download the latest Solidity++ compiler at [solppc Releases](https://github.com/vitelabs/soliditypp-bin/releases)

### Install Test Wallet (optional)

Setup [Vite Test Wallet](./testdapp.html) to connect to your development node.

## Debug Contract

Install VSCode IDE and Soliditypp Extension Plugin at [https://code.visualstudio.com](https://code.visualstudio.com/). 
Follow [This Guide](./debug.html#debugging-in-vs-code) to write and debug your smart contract in VSCode IDE.

## Deploy Contract

We suggest to deploy smart contract through Vite.js SDK. 

:::tip Quota Required
Do not forget to get some quota for your contract by staking.
:::

Now let's see an example. The following code presents a simple HelloWorld contract.
```
pragma soliditypp ^0.4.2;
contract HelloWorld {
   event transfer(address indexed addr,uint256 amount);
     onMessage SayHello(address addr) payable {
        addr.transfer(msg.tokenid ,msg.amount);
        emit transfer(addr, msg.amount);
     }
}
```
Compile the contract by running the following command. Now we get contract's ABI and binary code.
```bash
./solppc --abi --bin HelloWorld.solpp
```

:::tip Tips
ABI and binary code can also be generated in the IDE. 
:::

Now deploy the contract through a node. 
Below Javascript code shows how to deploy smart contract and obtain quota using Vite.js:
```javascript
const { HTTP_RPC } = require('@vite/vitejs-http');
const { ViteAPI, accountBlock, wallet, constant } = require('@vite/vitejs');

let provider = new HTTP_RPC("http://127.0.0.1:23456");
let client = new ViteAPI(provider);

// import account
let mnemonic = "sadness bright mother bid tongue same pear recycle useless hub beauty frozen toward nominee glide cheese picnic vibrant vague thought hurry sleep hold lizard";
let myAccount = wallet.getWallet(mnemonic).deriveAddress(0);

let abi = [{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"SayHello","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"addr","type":"address"},{"indexed":false,"name":"amount","type":"uint256"}],"name":"transfer","type":"event"}];
let binaryCode ='608060405234801561001057600080fd5b50610141806100206000396000f3fe608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806391a6cb4b14610046575b600080fd5b6100896004803603602081101561005c57600080fd5b81019080803574ffffffffffffffffffffffffffffffffffffffffff16906020019092919050505061008b565b005b8074ffffffffffffffffffffffffffffffffffffffffff164669ffffffffffffffffffff163460405160405180820390838587f1505050508074ffffffffffffffffffffffffffffffffffffffffff167faa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb346040518082815260200191505060405180910390a25056fea165627a7a72305820f50fe89a37e6cda887aaeae59bf45670d552e27215ee8aed9b83fe0c8d525fcb0029';

// create a new contract
let block = accountBlock.createAccountBlock('createContract', {
        address: myAccount.address,
        abi,
        code: binaryCode,
        params: []
    }).setProvider(client).setPrivateKey(myAccount.privateKey);

block.autoSetPreviousAccountBlock().then(() => {
    block.sign().send().then((result) => {
        console.log('Smart contract %s deployed!', result.toAddress);

        // stake 10000 VITE for the new contract for quota
        let contractAddress = result.toAddress;
        let block = accountBlock.createAccountBlock('stakeForQuota', {
            address: myAccount.address,
            beneficiaryAddress: contractAddress,
            amount: '10000000000000000000000'
        }).setProvider(client).setPrivateKey(myAccount.privateKey);
        block.autoSetPreviousAccountBlock().then(() => {
            block.sign().send().then(() => {
                console.log('Staked %s VITE to address %s!', 10000, contractAddress);
            }).catch((err) => {
                console.error('Error', err);
            });
        });
    });
}).catch(err => {
    console.error(err);
});
```

To verify a deployed smart contract, use [Contract Query API](../../api/rpc/contract_v2.html#contract_getcontractinfo). If the returned summary has binary code contained, the contract is successfully deployed. 

## Call Contract

In above example, the smart contract we deployed has one function `SayHello(address addr)`, which accepts an address parameter. Let's call it.

```javascript
async function callContract(contractAddress, methodName, abi, params, amount) {
    const block = accountBlock.createAccountBlock('callContract', {
        address: myAccount.address,
        abi,
        methodName,
        amount,
        toAddress: contractAddress,
        params
    }).setProvider(client).setPrivateKey(myAccount.privateKey);

    await block.autoSetPreviousAccountBlock();
    await block.sign().send();
}
// say hello to vite_d8f67aa50fd158f1394130a554552204d90586f5d061c6db4f
callContract(contractAddress,'SayHello', abi, ['vite_d8f67aa50fd158f1394130a554552204d90586f5d061c6db4f'], '100000000000000000000')
.then(res => console.log(res))
.catch(err => console.error(err));
```

## Remote Signing Library

In most cases, dApp should not manage private keys and mnemonics, which, for safety reasons, should always be stored in user's Vite wallet. Therefore, how dApp calls up Vite wallet and sends transactions through becomes the real concern. How is it addressed in Vite? 

We provide two libraries.
- [@vite/bridge](https://www.npmjs.com/package/@vite/bridge)   
    Vite Bridge is recommended for dApps that are integrated into Vite wallet. Through Vite Bridge client SDK, you are able to
    - Obtain current user's Vite address within web application, and
    - Send transaction or call smart contract from web application.
    
:::warning Important
If your dApp is not to be integrated with Vite wallet app. Do NOT use Vite Bridge.
:::

Let's see an example of sending 1 VITE to a second address.
```javascript
import Bridge from "@vite/bridge";
import { utils, constant } from "@vite/vitejs";

// initiate bridge instance
const bridge = new Bridge();

// get current account address
bridge['wallet.currentAddress']().then(accountAddress => {
    // send 1 vite to target address
    bridge["wallet.sendTxByURI"]({
        address: accountAddress, 
        uri: utils.uriStringify({ 
            target_address: 'vite_9de8095568105ee9a5297fd4237e2c466e20200c9fb012f573', 
            params: { 
                amount: 1, // 1 vite
                tti: constant.Vite_TokenId // default is vite. use another tti if you need to send other tokens
             }
        })
    }).then(accountBlock => {
      console.log(accountBlock);
    });
}).catch(err => {
    console.error(err);
});
```
Here below is another example of calling a smart contract.
```javascript
import Bridge from "@vite/bridge";
import { abi, utils } from "@vite/vitejs";

const bridge = new Bridge();
// encode function call
const hexdata = abi.encodeFunctionCall([{
    "name": "SayHello",
    "type": "function",
    "inputs": [{
        "type": "address",
        "name": "addr"
    }]
}], ['vite_9de8095568105ee9a5297fd4237e2c466e20200c9fb012f573'], 'SayHello');
// convert to base64
const base64data = utils._Buffer.from(hexdata, 'hex').toString('base64');
// send the call
bridge["wallet.sendTxByURI"]({
    address: accountAddress, // your account address
    uri: utils.uriStringify({
        target_address: contractAddress, // smart contract address
        function_name: 'SayHello',
        params: {
            data: base64data
        }
    })
}).then(accountBlock => {
  console.log(accountBlock);
});
```
To learn more about Vite Bridge, access our source code on [Github](https://github.com/vitelabs/bridge/).

- [@vite/connector](https://github.com/vitelabs/vite-connect-client)
    Vite Connect is the recommended solution for signing transactions for dApps that are not hosted in Vite wallet. At present the following features are supported
    - Establish connection sessions from Vite wallet to dApp by scanning QR code displayed on dApp's web page
    - Connection is retained for the whole session until disconnected
    - Transactions triggered on dApp are signed and sent out through Vite wallet app, not on dApp
    - Once enabled, transactions can be auto-signed
    
:::tip Recommended
Vite Connect is the general remote signing solution for all dApps that will not be integrated into Vite Wallet.
:::
Let's see a piece of code that defines how Vite Connect should be setup in Javascript client.
```javascript
import Connector from '@vite/connector'

const BRIDGE = 'http://192.168.31.82:5001' // url to vite connect server

const vbInstance = new Connector({ bridge: BRIDGE })

vbInstance.createSession().then(() => {
    // vbInstance.uri can converted into an QR code image.
    // in most scenarios, you should display the QR code here so it can be scanned by the vite wallet in order to establish connection
    console.log('connect uri', vbInstance.uri)
});

vbInstance.on('connect', (err, payload) => { // connection established
    /* 
     * Payload is an Object following the following definition: (usually the peer is Vite App)

     *  {
     *      version: number,    // vc protocol version, 2 at present
     *      peerId: string,     // can be ignored
     *      peerMeta: {         // Vite App meta info
     *          bridgeVersion: number,
     *          description: string,
     *          url: string,
     *          icons: string[],
     *          name: string,
     *      },
     *      chainId: number,    // can be ignored
     *      accounts: string[]  // the address returned from Vite wallet.
     *  }
     */
    const { accounts } = payload.params[0];
    if (!accounts || !accounts[0]) throw new Error('address is null');

    const address = accounts[0];
    console.log(address)
})

// send transaction
vbInstance.sendCustomRequest({
    method: 'vite_signAndSendTx',
    params: {
        /*
         * block should have the following parameters:
           {
                toAddress: string;   // account address or contract address
                tokenId: string;    // asset id that you would like to send
                amount: string;     // in atomic unit (with full decimals)
                fee?: string;       // in atomic unit (with full decimals)
                data? string;       // base64 encoded string, necessary when calling smart contract
           }
         */
        block: {
            accountAddress: "vite_61404d3b6361f979208c8a5c442ceb87c1f072446f58118f68",
            amount: "2000000000000000000",
            data: "c2FkZmFzZg==",
            toAddress: "vite_61404d3b6361f979208c8a5c442ceb87c1f072446f58118f68",
            tokenId: "tti_5649544520544f4b454e6e40",
        },
    }
}).then(signedBlock => console.log(signedBlock), err => console.error(err))
// register disconnection listener
vbInstance.on('disconnect', err => {
    console.log(err) // any handling logic here
})  
```
To learn more about Vite Connect, access our source code on [Github](https://github.com/vitelabs/vite-connect-client).

## Useful APIs

You may use the following APIs in your dApp.

### RPC Query API

|  API  | Description |
|:------------|:-----------|
| [ledger_getLatestAccountBlock](../../api/rpc/ledger_v2.html#ledger_getlatestaccountblock) | Get the latest transaction of the specified account |
| [ledger_getAccountInfoByAddress](../../api/rpc/ledger_v2.html#ledger_getaccountinfobyaddress) | Get account summary by address, including chain height, balances, etc. |
| [ledger_getAccountBlocksByAddress](../../api/rpc/ledger_v2.html#ledger_getaccountblocksbyaddress) | Get transaction list of the specified account |
| [ledger_getAccountBlockByHeight](../../api/rpc/ledger_v2.html#ledger_getaccountblockbyheight) | Get transaction by block height |
| [ledger_getAccountBlockByHash](../../api/rpc/ledger_v2.html#ledger_getaccountblockbyhash) | Get transaction by hash  |
| [ledger_getVmLogs](../../api/rpc/ledger_v2.html#ledger_getvmlogs) | Get smart contract execution logs by log hash |
| [ledger_getUnreceivedTransactionSummaryByAddress](../../api/rpc/ledger_v2.html#ledger_getunreceivedtransactionsummarybyaddress) | Get summary of unreceived transactions for the specified account |
| [ledger_getUnreceivedBlocksByAddress](../../api/rpc/ledger_v2.html#ledger_getunreceivedblocksbyaddress) | Get unreceived transaction list for the specified account |
| [contract_getContractInfo](../../api/rpc/contract_v2.html#contract_getcontractinfo) | Get contract summary, including code, consensus group, etc. |
| [contract_callOffChainMethod](../../api/rpc/contract_v2.html#contract_calloffchainmethod) | Call contract's' off-chain method |

For API definitions for all RPC methods, please visit [RPC API](../../api/rpc/)

To learn more abut calling RPC API in Vite.js, please visit [Vite.js SDK](../../api/vitejs/ViteAPI/GViteRPC.html#how-to-call-gvite-rpc-methods)

### Event Subscription

Event subscription is an advanced function that can be used in dApp to monitor state change of smart contract.

See [Event Subscription](../../contract/subscribe.md) and [Subscription API](../../api/rpc/subscribe_v2.md) for more information.

## FAQ

* How to know smart contract execution result in time?
  
  Since smart contract is executed asynchronously on Vite, you do not know execution result immediately after a request function call has been sent. 
  You must get the execution info in the response transaction, which is performed a bit later according to various contract parameters and Vite network status. 
  
  One way to obtain execution result is polling `ledger_getAccountBlockByHash` by your request transaction hash to see if it is received by smart contract. 
  Another is to use event subscription.
      
  Once the request transaction is successfully received, you can check the 33th byte in data field of the response transaction. `0` means execution succeeded while `1` stands for failure. 
  Usually a failed execution may result from `REVERT` instruction in function call, insufficient quota of smart contract or insufficient balance upon transferring to 3rd account.
  
  Usually (depending on how your smart contract is written), some events were triggered during execution and saved in `logHash` field of the response transaction. Use `ledger_getVmLogs` to get the events.
  
