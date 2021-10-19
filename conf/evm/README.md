

```
# run local node
docker run -v ~/.gvite/ipc:/root/ipc -p 127.0.0.1:48132:48132 --rm vitelabs/gvite-nightly:latest --config conf/evm/node_config.json

# query local node height
docker run -v ~/.gvite/ipc:/root/ipc --rm vitelabs/gvite-nightly:latest rpc /root/ipc/gvite.ipc ledger_getSnapshotChainHeight

# mine a snapshot block
docker run -v ~/.gvite/ipc:/root/ipc --rm vitelabs/gvite-nightly:latest rpc /root/ipc/gvite.ipc miner_mine


# query local node height
curl --location --request POST 'http://127.0.0.1:48132' --header 'Content-Type: application/json' --data-raw '{  "jsonrpc": "2.0",  "id": 1,  "method":"ledger_getSnapshotChainHeight",  "params":null}'


# mine a snapshot block
curl --location --request POST 'http://127.0.0.1:48132' --header 'Content-Type: application/json' --data-raw '{  "jsonrpc": "2.0",  "id": 1,  "method":"miner_mine",  "params":null}'

```