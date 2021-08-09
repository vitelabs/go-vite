---
order: 2
---
# VEP 3: Vite Wallet Key Derivation

## Summary

VEP 3 defines the private key derivation rules(with multiple curve algorithms) in Vite wallet.

## Purpose

HD wallet(Hierarchical Deterministic Wallet) has the capability of deriving multiple private keys from one single seed, thus making it easy to do wallet backup or upgrade to other compatible wallets such as hardware wallet. 

HD wallet also supports multiple tokens.

## Content

The seed is represented by a number of words that is easy to remember or write on paper, called mnemonic phrase. In protocol [BIP39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki) a mnemonic phrase is defined in 12 or 24 words.

Based on [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki), [BIP44](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki) allows wallet to store multiple tokens and accounts with one seed.

Since Vite adopts ED25519 for signature, other than secp256k1 used in BIP32, we need to find a key derivation method which is compatible with BIP32. The actual signature implementation in Vite is similar to [ed25519-bip32](https://cardanolaunch.com/assets/Ed25519_BIP.pdf).

### BIP44 Compatibility

[BIP44](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki) defines a BIP32-compatible path patten that supports multiple cryptocurrencies:
```
m / purpose' / coin_type' / account' / change / address_index
```
The prefix is always like:
```
m/44'
```
We have registered `coin_type` **666666** at [SLIP-0044](https://github.com/satoshilabs/slips/blob/master/slip-0044.md), so a BIP44 Vite path starts with:
```
m/44'/666666'/
```
Attached an index `x`:
```
m/44'/666666'/x'
```
* x: Index. Particularly, we define the address associated with `m/44'/666666'/0'` as **Primary Address** of the mnemonic.

## Test Case
### Entropy
```
87ad0e066111ed827dc1f7be4d1bf53b9a7be84021a0950418d3f45ed4d54f1c
```
### Mnemonic Phrase
```
marble half light season burst scorpion warfare discover salad hand wool jaguar police vintage above cross never camp crunch trim unhappy height detect opinion
```
### BIP39 Seed
```
2ba1d8e696d17ac4d75b9f479c527450d439c9acd2b4d542d27e3a7f3418cd241717d2db41f47d8bbae9fc90fe551c4db87f7491104f030f6eceaf1b24f15f4d
```
### Derived Vite ED25519 Seeds and Addresses

```
m/44'/666666'/0' bb369222613ad7b1a646d84d8c749c30cfa5879f5152b7bd7c1f9f6553ce0eb5  vite_da3ca9bac9f05fce8f4eead36610756b6eb48282ff10a81d6d
m/44'/666666'/1' 529892283122a9a09059a73147cb9feea480bb3feed91e7968243f4b67ccb3ea  vite_e5deb80a64f51593398ba1049af435291e3cb5c69a66755f13
m/44'/666666'/2' 9ef6f33aaf05fa1cf6e8c396b01a5ea08295a829b595f636759343d363a6a967  vite_fbdd0c038f808560f9637754cbbbfa95ed2e7cdb96113ea7eb
m/44'/666666'/3' 6da0edd6d81033b4b2d41f376574448876e7b4f841aedd7deaf8bb6d7934b800  vite_2aa258c33a2d16d01da651a9423abc384f6367112c0f73fa5d
m/44'/666666'/4' 98d2311c78e6407bf0c443ab51593c4b663ce3af3165a48a278ed0a6a2f701f3  vite_b8d401c1c7b3f32bf7d9c7a44c8d594fcdad103bb6775bd016
m/44'/666666'/5' bd18f1dfc81bc742cda2c3739a42fb622415d62b8fd6035ff8bad2a9b13f26b6  vite_7aba6649b09a43130445dd70857e77bef347e2da2a7b81f608
m/44'/666666'/6' b1cf0511a4bb7a154cf0f6c416a3186c4d6fe8cd53413c6503e80445918837e8  vite_5ef7da6c7fb79051921d0c6cf7440fb9f1b46d7aaf5607a069
m/44'/666666'/7' 2b31aa5f86e1207baa4ec93dd397c878ef53255a3ca64cafa970bb6513fb7099  vite_6ebaad8ee67e5368884ae2de652024093453ec13d8f17e0afa
m/44'/666666'/8' 87210e4ec4776ab5e6bc146255b1e649f6a4f99e754ff31e421e2f43784f2aff  vite_ec84678c2d6f1f12596552a0d676ab233d17249463973f7238
m/44'/666666'/9' e02109c47903e2f8858660f30642e3698a263f0277ad2f364527bc75275a82ec  vite_1720f21b3a66c30da966ef51dc59c091543da012bcb69ae8a4

```
