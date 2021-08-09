---
order: 7
---
# VEP-10: Vite TestNet-PreMainnet Data Migration Plan

## Background

Due to massive optimizations made in blockchain's data structure and consensus algorithm, old transaction data cannot be verified in the pre-mainnet. Therefore, transactions in TestNet will not be retained after the migration, but only account status will be.

## Objectives

1. Keep complete account state information
2. Ensure consensus works smoothly when pre-mainnet launches

## Migration Plan

Generate a genesis block on each account involved to store balance, contract states and other information that should be migrated. Then snapshot these genesis account blocks into a genesis snapshot block.

### User Account 

User account status includes account balance and un-received transactions.
* Balance is directly saved
* Amounts of all un-received transactions are summed up and added into account balance
* Only balances of **VITE** and **VCP** are retained

### Contract Account

Contract account status includes contract information (code, delegated consensus group that the contract belongs to), contract states (contract storage), contract account balance and un-received transactions.

#### Built-in Contracts

* Balance is directly saved
* Amounts of all un-received transactions are returned to original accounts
* Only balances of **VITE** and **VCP** are retained

Staking contract (address `vite_0000000000000000000000000000000000000003f6af7459b9`)
* Retain all staking information with new expiration height 1 (can be retrieved immediately after the pre-mainnet launches)

Original SBP registration contract, voting contract and consensus group contract are merged into new consensus group contract (address `vite_0000000000000000000000000000000000000004d28108e76b`)
* Retain all valid SBP registrations with new registration expiration height 7776000 (about 3 months after pre-mainnet launches)
* Retain all voting information
* Retain all settings in snapshot consensus group and public delegated consensus group

Mintage contract (address `vite_000000000000000000000000000000000000000595292d996d`)
* Retain token information for **VITE** and **VCP**
* Change **VITE** to re-issuable
* For other tokens, token issuance fee (1,000 VITE each) is refunded to issuer's account and the relevant token will not be retained

#### User-deployed Contracts

* User-deployed contracts will not be retained
* Contract deployment fee (10 VITE each) is refunded to contract creator
* Contract balance is returned to contract creator
* Amounts of all un-received transactions are returned to original accounts
* Only balances of **VITE** and **VCP** are retained

## Conclusion

* Consensus state is carried over from TestNet. The snapshot consensus group info and SBP ranking is unchanged
* Account information remain unchanged, including balance, staking, voting, SBP registration and token issuance information
* All user-issued tokens need to be re-minted in pre-mainnet
* All user-deployed smart contracts need to be re-deployed in pre-mainnet
