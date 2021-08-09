---
order: 6
---
# VEP 8: AccountBlock Data Content Type Definition

## Background

In general, the data field in account block can be used to carry additional information to meet specific demands when sending transaction through `tx_sendRawTx` API. 
This document explains the content definition in details.

## Implementation

We propose to set a 2-byte content type flag at beginning of data field to indicate what type of data is followed. 

Content type is a `uint16` number stored in big endian format. Number **1 (0x0001)** - **2048 (0x0800)** are reserved and should not be used by a third party.

::: warning Restriction
It is known that content type might be occasionally mis-recognized under the situation calling a smart contract, due to messing-up with the first 2 bytes of a hitting method hash.
:::

## Defined Types

### General Type
| Type | Type(Hex) | Description |
| --- | --- | --- |
| Binary data | 0x0001 | Reserved. Not in use. |

### Custom Type
| Type | Type(Hex) | Description |
| --- | --- | --- |
| Vite Grin wallet index | 0x8001 | Data of Grin transaction index |
| ViteX gateway data | 0x0bc3 | Data of cross-chain transaction of ViteX gateway |
