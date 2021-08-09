# Mintage

::: tip
Token issuance and related topics are described in this page.

Definitions of Terms:
* **Token**: A digital asset
* **Mintage**: Issuing new token
* **Re-issuance**: Issuing an additional amount of token after initial mintage
* **Burn**: Destroying an amount of token
* **Token owner**: The account that issued the token. Token ownership can be transferred to another account. In this case, the previous account is not the owner any more.
:::

## What is Token?

In Ethereum, new token can be issued by deploying a smart contract that complies with Ethereum's token standard such as ERC20. 
In this operation, an amount of token equal to total supply are generated and a holding relationship map is maintained within the contract. 
Transfers of the token are executed internally, and do not change the status of user's accounts.

Unlike Ethereum, Vite supports users to issue their own tokens natively. 
User can send a mintage transaction to issue new token, specifying token type(re-issuable or fixed-supply), total supply, token name, token symbol, and optional maximum supply if the token is re-issuable. 
Issued token (equivalent to total supply) will be sent to token owner's account. Token balance is maintained in each Vite account, not just in a smart contract as Ethereum does.

For re-issuable token, the owner can re-issue an additional amount of token after initial mintage. In this case, token's total supply will increase accordingly. 
Similarly, re-issuable token can be destroyed by burning a certain amount through burn transaction. As the result, this will reduce the total supply. 
In addition, the ownership and token type of re-issuable token can also be changed.

### Token Type ID
A TTI (Token Type ID) is a 28 characters length hex string that is composed of three parts:
* "tti_": 4-char long prefix
* TTI body: a byte array of 10 bytes length 
* Checksum: a 2 bytes blake2b hash of the TTI body. 

The TTI body comes from the following equation:
$$TTI_{body}=blake2b\left(IssuerAddress+MintageHeight+PrevBlockHash\right)$$
IssuerAddress, MintageHeight and PrevBlockHash are byte arrays of the token issuer address, the mintage contract height and the previous block hash of the issuer's account. Note that MintageHeight is in 8-byte length, if the converted value does not reach this length, zeros will be padded at left.

## Mintage

Mintage transaction is the transaction of issuing new token on Vite. Token issuer can send such a transaction to Vite's built-in mintage contract with given parameters. 
Once the transaction is done, minted token will be sent to the issuer's account and the issuer is marked as owner.

Issuing a new token will burn ***1,000 VITE***.

### Parameters

* `tokenName`: Name of token, 1-40 characters, including uppercase/lowercase letters, spaces and underscores. Cannot have consecutive spaces or start/end with space
* `tokenSymbol`: Symbol of token, 1-10 characters, including uppercase/lowercase letters, spaces and underscores. Cannot use `VITE`, `VCP` or `VX`
* `totalSupply`: Total supply. Having $totalSupply \leq 2^{256}-1$. For re-issuable token, this is the initial total supply
* `decimals`: Decimal places. Fill 0 if there is no decimal 
* `isReIssuable`: If `true`, newly additional amount of token can be issued after initial mintage
* `maxSupply`: Maximum supply. Having $totalSupply \leq maxSupply \leq 2^{256}-1$. For non-reissuable token, fill in 0
* `isOwnerBurnOnly`: If `true`, the token can burned only by owner. Mandatory for re-issuable token. For non-reissuable token, fill in `false`

## Re-issuance

Re-issuance transaction must be initiated by the token owner. 
Once the transaction is done, newly issued token will be transferred to the specified account. 

### Parameters

* `tokenId`: Re-issuable token id
* `amount`: Issuance amount
* `receiveAddress`: Address to receive newly issued token

## Burn

Burn transaction can be initiated by token owner or holders. 
Once the transaction is done, the amount of token are burned and the total supply is reduced to reflect the change. 

If `ownerBurnOnly` is turned on, only the owner can burn this token.

### Parameters
* `tokenId`: Re-issuable token id
* `amount`: Burn amount. Having $newTotalSupply = oldTotalSupply-amount$. Destroyed token amount will be deducted from the account of transaction initiator.

## Transfer Ownership

Ownership of re-issuable tokens can be transferred. In this case, the original owner should send a transaction to initiate the transfer. 
Once the transaction is done, the token has been transferred to new owner.

A token can only have one owner at a time. 

### Parameters
* `tokenId`: Re-issuable token id
* `newOwner`: Address of the new owner

## Change Token Type

Re-issuable tokens can be changed to non-reissuable. Note this operation is one-way only and cannot be reversed. 
Once the transaction is done, the token's type is permanently set to fixed-supply token.

### Parameters
* `tokenId`: Re-issuable token id

## Tokens in Vite

In addition to native coin **VITE**, two other tokens **VCP** (Vite Community Points) and **VX** (ViteX Coin) are issued in Vite. The parameters are as follows:

| Token Id | Token Name | Token Symbol | Total Supply | Decimals |
|:------------:|:-----------:|:-----------:|:-----------:|:-----------:|
| tti_5649544520544f4b454e6e40 | Vite Coin | VITE | 1 billion | 18 |
| tti_251a3e67a41b5ea2373936c8 | Vite Community Point | VCP | 10 billion | 0 |
| tti_564954455820434f494e69b5 | ViteX Coin | VX | 10 million | 18 |

## FAQ

* Can I issue multiple tokens in my account?

Yes, there is no limitation of mintage number of an account.

* Can I use existing token name and symbol?

Token name can be reused. But token symbol is unique. A suffix number from '000' to '999' will be added to token name by issuance time. For example, BTC can have symbol 'BTC-000', 'BTC-001', 'BTC-002', etc.

* Does mintage transaction consume quota?

Yes. Mintage, re-issuance, burning, ownership transfer and change of token type consume quota. For specific quota consumption please refer to [Quota Consumption Rules](./quota.html#quota-consumption-rules).

* I understand token mintage costs **1,000 VITE**. Does re-issuance also cost Vite coins?

No, re-issuance transaction does not cost Vite coin.

* Can I change fix-supply tokens to re-issuable?

No. Change of token type is one-way and not reversible.
