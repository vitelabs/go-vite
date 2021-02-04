# Troubleshooting & FAQ - Node

## Recommended Specs
* SBP (Supernode) - 2 CPUs / 8 GB RAM
* Full node - 1 CPU / 4 GB RAM

A minimum of 5M bps stable internet connection is required.

## Troubleshooting
### Boot-up failures
* `new node error`

JSON Format error in node_config.json or genesis.json.

* `panic: The fork point abcFork can't be nil. the ForkPoints config in genesis.json is not correct, you can remove the ForkPoints key in genesis.json then use the default config of ForkPoints`

Unconfigured or incorrect fork point in genesis.json. Check `ForkPoints` in the config file and make sure it is aligned with the upcoming hard fork. For testnet node, the hard fork height can directly be specified in genesis.json.

* `Failed to prepare node, dataDir already used by another process`

The `DataDir` has been occupied by another gvite process. Kill the process and try again. 

* `Failed to prepare node, stat {datadir}/maindata/wallet/vite_xxx: no such file or directory`

The keystore file is missing. Check if `DataDir` is specified in node_config.json and the correct keystore file is in the folder. You can also specify a `KeyStoreDir` in node_config.json.

* `Failed to prepare node, error decrypt store`

Unlock account failed, usually caused by a mismatched key store file and password.

* `Failed to start node, no service for name {abc}`

Module {abc} in `PublicModules` does not exist. Remove {abc} from node_config.json.

### Node is not syncing
The node boots up successfully but the snapshot block height does not increase in 5 minutes. 

Follow the below steps to check.
* Make sure the node has been upgraded to the latest [stable version](https://github.com/vitelabs/go-vite/releases);
* Make sure the timestamp on the node is accurate;
* Check connected peers through [net_peers](../../api/rpc/net.html#net_peers). If peerCount=0, make sure port 8483/8484 are exposed, and run `curl https://bootnodes.vite.net/bootmainnet.json` to check if the node is connected to internet;
```sh
curl -X POST \
  http://127.0.0.1:48132/ \
  -H 'content-type: application/json' \
  -d '{
        "jsonrpc": "2.0",
        "id": 1,
        "method": "net_peers",
        "params": null
      }'
```
* If the node has connected to other peers, check sync status through [net_syncDetail](../../api/rpc/net.html#net_syncDetail). If the chunk returned is empty, wait 5 minutes and check again;
```sh
curl -X POST \
  http://127.0.0.1:48132/ \
  -H 'content-type: application/json' \
  -d '{
        "jsonrpc": "2.0",
        "id": 1,
        "method": "net_syncDetail",
        "params": null
      }'
```
* Reboot gvite, wait for 5 minutes and watch if the snapshot chain height increases;
* Examine the status of snapshot chain through debug_poolSnapshot (add `debug` in `PublicModules` of node_config.json to enable the debug tool). If there is a difference gap between Head and Tail, and Tail does not change, send the return values together with the latest log under {datadir}/maindata/runlog/ to Vite technical support for investigation.
```sh
curl -X POST \
  http://127.0.0.1:48132/ \
  -H 'content-type: application/json' \
  -d '{
      	"jsonrpc": "2.0",
      	"id": 1,
      	"method":"debug_poolSnapshot",
      	"params":null
      }'
```

### SBP does not produce block
The SBP node is synced up but does not produce snapshot block. 

Follow the below steps to check.
* Make sure the node has been upgraded to the latest [stable version](https://github.com/vitelabs/go-vite/releases);
* Make sure the timestamp on the node is accurate;
* Make sure the node has 4 CPUs and 8GB RAM installed with 5M bps internet connection, and there is no other program occupying the CPU, RAM, disk I/O and network bandwidth;
* Check node_config.json, make sure `Miner` is set to true, `EntropyStorePath` and `EntropyStorePassword` are correctly configured;
* Check the registration information of the SBP on [Vite block explorer](https://explorer.vite.net/SBPList) to make sure the current block creation address matches the address configured in `EntropyStorePath`;
* Check the SBP rank on [Vite block explorer](https://explorer.vite.net/SBPList). SBP nodes ranked after 25 have a smaller rate to produce blocks. SBP ranked after 100 do not have chance to produce snapshot block;
* Restart the node;
* If the problem still exists, send the latest log under ~/.gvite/maindata/runlog/ to Vite technical support for further investigation.

### SBP is missing blocks
The SBP node is in sync but missed some blocks. Follow the steps below to check.
* Make sure the node has been upgraded to the latest [stable version](https://github.com/vitelabs/go-vite/releases).
* Make sure the timestamp on the node is accurate.
* Make sure the node has installed 4 CPUs and 8GB RAM with 5M bps internet connection, and there is no other program occupying the CPU, RAM, disk I/O and network bandwidth.
* Restart the node.
* If the problem still exists, send the latest log under {datadir}/maindata/runlog/ to Vite technical support for investigation.

### Node reports too many open files
It is a common error in Linux. Increase the maximum number of open files in the system. Follow the steps below to fix.

Run
```sh
sudo vim /etc/security/limits.conf 
```
Add 
```
* soft nofile 10240  
* hard nofile 10240
```
or 
```
* - nofile 10240
```
or (if you have root access)
```
root soft nofile 10240  
root hard nofile 10240
```
Save and quit

Logout and login again

Run
```sh
ulimit -n
``` 
to check the new value is applied

You can also run
```sh
ps -ef | grep gvite
```
Get the pid of gvite
```sh
cat /proc/{pid}/limits | grep open
```
Check the value displayed

## FAQ
### What is the difference between SBP staking address and block creation address?

When an SBP is registered, the registration account should stake 1m VITE. The registration address is the staking address and becomes the owner of the SBP. After the staking lock-up period expires, the staking address can cancel the SBP and retain the 1m VITE staked.

Once registered, the staking address cannot be changed. This is different with block creation address and reward withdrawal address, the both can be updated after registration.

The private key of block creation address is saved on the node (corresponding to `EntropyStorePath` and `EntropyStorePassword` in node_config.json), and the only purpose of block creation address is to sign snapshot blocks. 

Reward withdrawal address is used to withdraw SBP rewards. 

:::tip Tip
It is highly recommended to separate staking address from block creation address. Do NOT store assets in block creation address.
:::

### Do I need to upgrade my node when there is a new release? 
There are two types of gvite release. For the releases tagged "Upgrade is required", you should complete the upgrade within the time in order to be compatible with the hard fork. No worries, you have enough time to finish the upgrade. Announcements will be declared on Vite social channels including Telegram group, Discord channel, Vite block explorer notifications, and Vite forum usually one month ahead of the hard fork.
Another type of release is general release, including improved stability and performance, network optimization, new toolkit interface, etc. Upgrade is not required for this kind of release, but recommended.

If there are no special instruction in the announcement, to upgrade a node you should just replace the gvite file and restart it. This is convenient because there is no need to modify node_config.json. If there are instructions in the release announcement, follow the instructions.

After reboot, watch if the snapshot chain height is increasing (for full node and SBP node) and snapshot blocks are produced normally (for SBP node). If yes, the upgrade is successful.

:::tip Tip
To avoid unnecessary block missing for SBP node during upgrade, replace gvite file first then reboot the node.
:::

### How to check sync status?
You have two alternatives.
* Check through [net_syncinfo](../../api/rpc/net.html#net_syncinfo)，`Sync done` indicates the node is synced up. 
```sh
curl -X POST \
  http://127.0.0.1:48132/ \
  -H 'content-type: application/json' \
  -d '{
      	"jsonrpc": "2.0",
      	"id": 1,
      	"method":"net_syncinfo",
      	"params":null
      }'
```
* Check through gvite.log or [ledger_getSnapshotChainHeight](../../api/rpc/ledger_v2.html#ledger_getSnapshotChainHeight) to check the latest snapshot block height on the node and compare to explorer.
```sh
curl -X POST \
  http://127.0.0.1:48132/ \
  -H 'content-type: application/json' \
  -d '{
      	"jsonrpc": "2.0",
      	"id": 1,
      	"method":"ledger_getSnapshotChainHeight",
      	"params":null
      }'
```

### How is the SBP reward allocated?
See [SBP reward rules](../rule/sbp.html#sbp-rewards). You can download a detailed voting spreadsheet from the SBP details page of explorer. 

### Can I copy the ledger files to a new node? 
Copying ledger files to another node is allowed. Be noted to stop the node first, and you just need to copy `{datadir}/maindata/ledger` folder. 




