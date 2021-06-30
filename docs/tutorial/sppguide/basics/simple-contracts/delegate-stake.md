---
order: 6
---
# Quota Bank

This contract demonstrates a simple quota bank that stakes to obtain quota for given address.

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/delegate-stake.solidity

#### Key Features

- `onMessage () payable {}` the payment fallback message listener allows in-bound transfers.

- `hash = send(stakeContract, StakeForQuota(addr), viteTokenId, amount);` calls the built-in quota contract's StakeForQuota message listener. `viteTokenId` and `amount` specify the token type and amount sent with the call. 

- `message StakeForQuota(address recipient) payable;` declares a payable StakeForQuota message, which matches the message listener with the same name in the built-in quota contract.
