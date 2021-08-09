---
order: 11
---
# VEP-16 Specification of Address Generation

In the Mainnet, the account address is a 21-byte length number (hereinafter referred to **Original Address**), where the first 20 bytes is address and the last byte represents account type.

The first 20 bytes (hereinafter referred to **Address Body**) come from:

* For user account, a 20-byte hash based on the public key of the account.
* For contract account, a 20-byte hash based on the address of contract creation account, the height of contract creation block, and the hash of the previous account block of contract creation transaction. 

The last 1 byte (hereinafter referred to **Type Flag**)  is:

* 0 for user account
* 1 for contract account

In practice, a 55-length string is used as **Literal Address**.

Convert original address to literal address:

* For user account, use `vite_` + Hex string of **Address Body** + Hex string of the checksum of **Address Body**.
* For contract account, use `vite_` + Hex string of **Address Body** + Hex string of the flipped checksum of **Address Body** by bit.

Convert literal address to original address:

Take characters between 6-45 of **Literal Address** as the 20-byte **Address Body** and check:

* If the hex string of checksum of **Address Body** matches the characters from 46 to 55 in **Literal Address**, set 0 to **Type Flag** as the address is a user address. 
* If the hex string of flipped checksum of **Address Body** is the same as the 46th to 55th chars of **Literal Address**, set 1 to **Type Flag** as the address is a contract address.

The checksum is calculated by taking a 5-byte hash on the basis of **Address Body**.
