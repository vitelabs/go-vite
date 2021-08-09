---
order: 10
---
# VEP-15: Introduce Block Producing Rate in SBP Selection

## Background

In practice, SBP node may be temporarily down due to various reasons, such as poor internet connection, or temporary hardware overload. SBP node would fail to produce snapshot blocks during the time. 
Considering every SBP in top 25 has the responsibility to maintain the stability and security of the network by producing snapshot blocks on time, it's necessary to downgrade the "non-available" SBP in the round.

## Scheme

- Downgrade two nodes at most in a round
- Set the threshold of block producing rate in last hour at 80%. SBP nodes that are below the threshold will be downgraded
- Assign the first 25 nodes are consisted of Group A and those in 26-75 belong to Group B
  - Downgrade and replace a node with block producing rate below 80% in Group A with another node in Group B higher than 80%. In this case, the latter node is upgraded
  - Keep the number of nodes in Group A and B unchanged

For example, suppose the producing rate of a node ranking 24th is lower than 80%, and that of another node ranking 26 is above 80%, in the following round, the latter will replace the former to produce block.
