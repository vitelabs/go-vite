---
order: 9
---
# VEP-13: Rules of SBP Rewards Calculation and Distribution

## Background
In each year, an additional amount of Vite tokens which is no more than 3% of circulation will be issued for SBP incentivization. Taking circulation of 1 billion as an example, the reward for each snapshot block in the first year is `0.951293759512937595` VITE.

SBP rewards are calculated by cycle. A cycle has 1152 rounds, approximately equivalent to one day. To be eligible for rewards, the SBP node must be in top 100 in the last round of the cycle.

SBP rewards come from token inflation.

## Rules of Reward Calculation

SBP rewards consist of two parts: **Block Creation Reward** and **Candidate Additional Reward**.

* **Block Creation Reward**: 50% of rewards will be given to block producers in number of the blocks they created

* **Candidate Additional Reward**: 50% of rewards will be allocated to the top 100 SBPs in the last round of cycle based on block producing rate and the number of votes in the cycle.

Reward calculation:

$$Q = \frac{l}{m}*\frac{V}{W}*X*R*0.5 + l*R*0.5$$

* `Q`: the total reward of the SBP in one cycle
* `l`: the number of blocks that are produced by the SBP in one cycle
* `m`: the number of blocks that are expected to produce by the SBP in one cycle. If no block is produced by any SBP in a round, `m` should substrate the target block producing number of that round
* `X`: the total number of blocks that are produced by all SBP nodes in one cycle
* `W`: the total number of votes of the top 100 SBPs in the last round of one cycle, plus the total staking amounts of registration of the top 100 SBPs (1m VITE for each SBP in the Mainnet)
* `V`: the number of votes of the SBP in the last round of one cycle, plus the staking amounts of registration of this node (1m VITE for each SBP in the Mainnet)
* `R`: The reward for each block, fixed at `0.951293759512937595` VITE in the first year

For example:

Suppose S1 and S2 are SBPs and are in the top 100 in the last round of one cycle. The voting number of S1 and S2 is 200k and 300k respectively.

In each round, S1 and S2 are supposed to produce 3 blocks. 

In the first round, S1 produced 1 block, and the actual block number produced by S2 is 2.

In the second round S1 produced 2 block, and S2 produced all 3 blocks.

Start from the third round till cycle's end, neither S1 nor S2 produced any more block.

Rewards of S1 and S2 in the cycle can be calculated as below:

$$Q_{S1} = \frac{1+2}{3+3}*\frac{200000+1000000}{200000+1000000+300000+1000000}*(1+2+2+3)*0.951293759512937595*0.5+(1+2)*0.951293759512937595*0.5$$

$$Q_{S2} = \frac{2+3}{3+3}*\frac{300000+1000000}{200000+1000000+300000+1000000}*(1+2+2+3)*0.951293759512937595*0.5+(2+3)*0.951293759512937595*0.5$$

## Rules of Reward Withdrawal

* Reward must be withdrawn by the owner of SBP (registration account) or an account authorized by the owner
* Reward in latest *48* rounds (approximately one hour) cannot be withdrawn
* Reward must be withdrawn by cycles (days)
* Incomplete cycle is not eligible for reward, such as the first cycle of registration and the last cycle in which the registration is cancelled

## Reward Distribution

* First, SBP registration account sends a withdrawal request to Vite's built-in consensus contract, and specifies SBP name and an address for receiving reward (the default receiving address is registration address)
* The consensus contract calculates the actual amount of reward available for withdrawal, then sends a request to token issuance contract to mint new Vite coins. The issuing amount is the number calculated in this step. The recipient's address is the reward receiving address
* The token issuance contract mints new coins, and transfers to the reward receiving address
* Reward is received by receiving address

VITE is re-issuable. The token's owner is the consensus contract. In the protocol of Vite, this is the only re-issuance scenario of VITE.
