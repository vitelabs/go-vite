---
order: 4
---
# Batch Transfer

Below contract defines a batch transfer which accepts a list of addresses and amounts and then transfers specified amount to specified address in the list

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/batch-transfer.solidity

#### Key features

- `address[] calldata addrs`
  This shows how data can be passed to a contract in the form of an array and parsed.

- `uint256 totalAmount = 0;` This variable is local, and is not stored on the blockchain.

- `require` statements. If the argument is false, the transaction will be reverted.

- `for`, `if` statements, standard control flow.

