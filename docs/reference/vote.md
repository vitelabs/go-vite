# Voting

::: tip
Please note this is not a technical document, but mainly describes voting-related topics. Technical details will be introduced in the yellow paper.

The Definitions of Terms:
* **Voting**：A solution of on-chain governance by calculating the Vite coins held by voter as voting weight to elect super node.
* **Super node**： The node in snapshot consensus group who is eligible for producing snapshot block.
* **Delegated node**： The node in delegated consensus group who is eligible for producing blocks for corresponding smart contract.
:::

## What is voting

Vite uses a protocol-based voting mechanism for governance. There are two voting categories: global voting and delegated voting. Global voting is based on the VITE held by the user to calculate voting weight, mainly used for the super node election of the snapshot consensus group. The delegated voting is for smart contract. When the contract is deployed, a certain token is designated as the voting token, which can be used to elect the delegated node of the delegated consensus group that the contract belongs to.

In addition to the confirmation of transaction, the super node of the snapshot consensus group is able to choose whether to perform non-compatible upgrade on Vite system. Similarly, the delegated consensus group has the right to decide whether to upgrade an existing contract, so that avoiding the risk of contract upgrade failure. This helps improve the efficiency of decision-making and prevent decision failure from insufficient voting. These super nodes or delegated nodes are also subject to the consensus protocol. No upgrade will be implemented if the majority cannot reach agreement. Additionally, if they do not behave correctly as expected, users can vote them out.

## Voting rules

For each consensus group, user can vote for a delegated node by sending a voting transaction to the built-in contract with specified consensus group ID and the delegated node address he votes for. After the corresponding response transaction is confirmed by snapshot chain, voting is successfully completed.

Votes are calculated every round. The delegated nodes for next round will be elected based on the voting result at the time being.

Voting can be cancelled at any time, by sending a cancel-voting transaction with a specified consensus group ID.

## FAQ

* Can I vote for multiple super nodes in the snapshot consensus group at the same time?

No. An individual user can only vote for one super node at a time, however, you can still vote for delegated node in delegated consensus group in the meantime.

* If the delegate node I voted for has cancelled the qualification (delegate node can cancel the stake after 3 months of registration), what is going to happen to my voting?

If the delegate node cancels the stake, the belonging consensus group will no longer count the votes of this node. All votes for this delegated node will become invalid. You should vote for another node instead.

* If I don't have any VITE in my wallet, can I vote?

Voting consumes quota, not VITE, so that you can vote even if you do not have VITE in the account. However, SBP(snapshot block producer) is elected based on voting weight, if your VITE balance is 0, your vote has 0 voting weight and will be regarded as invalid.

* Can I vote for the delegated node in public delegated consensus group?

No, you cannot. However, since public delegated consensus group shares the same block producing nodes with snapshot consensus group, you should vote for super node in snapshot consensus group instead.
