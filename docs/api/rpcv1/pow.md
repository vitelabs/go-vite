# Pow

:::tip Maintainer
[TiantaoZhu](https://github.com/TiantaoZhu)
:::

## pow_getPowNonce
Calculate a PoW nonce by given difficulty and return. Usually this method is used to obtain temporary quota upon sending a transaction without staking.

- **Parameters**: 
1. `difficulty`: `string` - `bigInt` PoW difficulty in string. Not applied in current release.
2. `data` :`Hash` - Equivalent to $Hash(address + prehash)$. Here $address$ is a 20-byte binary account address while $prehash$ is the hash of previous transaction in the account (0 if no previous transaction)

- **return**:
1. `nonce`:`[]byte` - Nonce in base64 byte array
