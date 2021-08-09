---
order: 5
---
# Guess The Secret

Example of a guessing game contract that rewards winners in Vite.

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/guess-the-secret.solidity

#### Key features

- `mapping(bytes32 => uint) public balances;`
  Mappings are extremely useful data structures. In this example we use them to track an award balance for each "secret hash". Another common use is `mapping(address => uint)`, which stores a per-account balance.

- `tokenId token = tokenId("tti_5649544520544f4b454e6e40");`
  tokenId are the data type for native tokens on the Vite blockchain.
  The tokenId strings can be looked up, for example, on [vitescan.io](vitescan.io/tokens).

- `bytes32 hashval = blake2b(abi.encodePacked(body));` Here, `blake2b` is the Vite chain's hash function. To use it properly, the data must be encoded properly through use of the `abi.encodePacked`.
