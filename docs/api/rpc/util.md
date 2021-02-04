# util

:::tip Maintainer
[vite-crzn](https://github.com/vite-crzn)
:::

## util_getPoWNonce
Calculate a PoW nonce based on the given difficulty. Usually this method is called to obtain an temporary amount quota upon sending a transaction with no staking.

- **Parameters**: 
  * `string bigint`: PoW difficulty
  * `string hash`: $Blake2b(address + previousHash)$. For example, if address is `vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a` and previousHash is `0000000000000000000000000000000000000000000000000000000000000000`, the hash value is `8689fc3e7d0bcad0a1213fd90ab53437ce745408750f7303a16c75bad28da8c3`

- **Returns**: 
  - `string base64`: nonce
    
- **Example**:
::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "util_getPoWNonce",
	"params": ["67108863","35c82fe515c2982c5ef75226eab35f3fb14952f8ef59005f02893cd3dca4db09"]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "e5WeaVy7tSs="
}
```
:::

