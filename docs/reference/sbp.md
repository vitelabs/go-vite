# Snapshot Block Producer

::: tip
This document explains the concept of Vite SBP, SBP staking requirement and how SBP is selected and incentivized. 

Definitions of Terms:
* **SBP**: Snapshot Block Producer
* **Staking**： Locking an amount of **VITE** of the account for certain purpose. In this document, this is SBP registration.
* **A Round**: 75 seconds approximately, in which votes of SBPs are re-calculated. In ideal condition, 75 snapshot blocks are produced in a round.
* **A Cycle**: Refers to 1152 rounds, approximately one day.
* **Registration Address**: Also known as **Staking Address**. Refers to the address by which the SBP is registered.
* **Block Creation Address**: Refers to the address installed on the SBP node for producing blocks.
:::

## SBP Registration

Registering SBP is free in Vite through **Staking**.

### Staking Rules

**Staking Amount：**

*1,000,000 VITE*

**Staking Period：**

Staking cannot be retrieved immediately after registration. The lock-up period is approximately 3 months (**7776000** snapshot blocks). 
After the locking period expires, the SBP's owner (registration account) can cancel the SBP registration and retrieve staked VITE tokens. 

### Registration and Cancellation

In the Mainnet, registering SBP, producing blocks and withdrawing rewards can be from 3 different addresses. 

To register one SBP, the account shall send an **SBP Registration** transaction to Vite's built-in consensus smart contract. When the transaction is confirmed, the registration is completed.

Similarly, to cancel an SBP whose locking period has expired, the registration account needs send **Cancel Registration** transaction to the consensus contract. In this case, the node will be removed from SBP list once the transaction is confirmed.

#### Parameters

* **Block Creation Address**: 
It is highly recommended to use a different address with SBP registration address for security reason.
Block creation address can be changed by sending an **Update Registration** transaction by registration account.

* **SBP Name**: String of 1-40 characters, including Chinese and English, numbers, underscores and dots. Duplicated names are not allowed. SBP name is mainly used for voting.

## How SBP Works

### Round

A round is approximately 75 seconds. Voting result is re-calculated in every 75 seconds in order to select the SBPs in next round. Each SBP will have chance to produce 3 consecutive blocks in a round.

### Selection

In each round, 25 SBPs are selected according to following rules:

* 23 SBPs are randomly selected from the top 25 (sorted by votes) supernodes. Ratio is $23/25$
* 2 SBPs are randomly selected from supernodes ranking from 26 to 100. Ratio is $2/75$

## SBP Rewards

An annual inflation of up to 3% of circulation of VITE are minted as SBP rewards. In the Mainnet, the reward for each snapshot block is ${0.951293759512937595}$ **VITE**

### Reward Allocation

The rewards are equally split in 2 parts:

#### Rewards Allocated to SBP Who Produced Blocks

50% of rewards will be given to block producers in number of the blocks created, called **Block Creation Reward**.

#### Rewards Allocated to Top 100 SBPs 

50% of rewards will be given to the top 100 SBPs, called **Candidate Additional Reward**.
 
**Candidate Additional Reward** has following rules:

* Voting reward is proportional to the number of votes the SBP is entitled in last round of the cycle. 
* Rewards are generated daily in every *1152* rounds. Rewards generated *48* rounds ago (about 1 hour) are eligible for allocation. 
* In each cycle, SBP's up-time is calculated as $\frac{Total Blocks Produced}{Total Blocks On Target}$.

### Reward Calculation

**A cycle**: Refers to 1152 rounds, approximately 1 day. The first cycle starts at genesis snapshot.

* `Q`: the total reward of the SBP in one cycle
* `l`: the number of blocks that are produced by the SBP in one cycle
* `m`: the number of blocks that are expected to produce by the SBP in one cycle. If no block is produced by any SBP in a round, `m` should substrate the target block producing number of that round
* `X`: the total number of blocks that are produced by all SBP nodes in one cycle
* `W`: the total number of votes of the top 100 SBPs in the last round of one cycle, plus the total staking amounts of registration of the top 100 SBPs (1m VITE for each SBP in the Mainnet)
* `V`: the number of votes of the SBP in the last round of one cycle, plus the staking amounts of registration of this node (1m VITE for each SBP in the Mainnet)
* `R`: The reward for each block, fixed at `0.951293759512937595` VITE in the first year

$$Q = \frac{l}{m}*\frac{V}{W}*X*R*0.5 + l*R*0.5$$

Note:
* According to the above formula, if one SBP ranks more than 100 in the last round of a cycle, the reward for the SBP in this cycle is 0
* There is no reward for the first cycle the SBP is registered
* There is no reward for the last cycle the SBP is cancelled
* If 0 block is produced by all SBPs in a round (75 seconds), this round will be excluded from reward calculation by setting total blocks on target of the round as 0

#### Reward Withdrawal

In the Mainnet, SBP should explicitly send a **Reward Withdrawal** transaction from the staking address to get the rewards.

**Reward withdrawal rules:**

* Reward withdrawal transaction must be initiated by the registration account, or an account designated by it
* Rewards generated **48** rounds ago (about one hour) are available for withdrawal
* If no cycle is specified, all available rewards will be withdrawn. Available rewards are the rewards allocated since the time of last withdrawal till one hour ago
* Reward receiving address can be any Vite address
* Reward withdrawal transaction will consume `68200` quota


## FAQ
  
* I don't have enough **VITE** in my account. Can I register with someone else together?

   Currently, no. However, you can write a co-staking smart contract and have the contract send registration request once enough VITE are collected.

* If the uptime of one SBP in a cycle is 0, I understand the **Candidate Additional Reward** will be 0, but will this "missing" reward be distributed to other SBPs?

   Rewards distribution will take place only when requested by the SBP. If some SBP's rewards is 0, no VITE token will be minted, so as that other SBPs won't receive extra rewards.
  
* If I register an SBP but do not have a physical node, can I get reward?

   No. Your SBP reward will be 0 due to 0 uptime.
  
* I already have a SBP node running, but why do I find out my uptime is 0?

   This may happen. The possible reasons include poor internet connection, or your node does not have chance to produce block in the cycle due to low ranks.
  
* My SBP node produced a snapshot block, can I get both block creation reward and candidate additional reward?

   Not 100%. There is no reward for the cycle of registration and the cycle of cancellation. Also, if the SBP ranks out of 100 in the last round of the cycle, it does not have reward either.
  
* I staked 1m when I registered my SBP. Is it also eligible for quota?

   No. You cannot re-use these tokens for other purpose until your SBP's registration is cancelled. 
  
* Can I register multiple SBPs from the same account address?

   Yes, you can, by specifying a different block producing address for each new SBP node.
  
* If one SBP node has 100% uptime in the first half cycle, and due to some reason, no block is produced by any SBP in the second half cycle, what is the uptime of this SBP node in the cycle?

  The uptime is 100%. Because no block is produced in the second half cycle, only the first half will be counted for uptime calculation. 
  
