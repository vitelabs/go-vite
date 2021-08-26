# Full Node Rewards

A brand new Full Node Rewards Program is launched on April 3, 2021 to encourage participation and provide more flexibility for full node providers to earn rewards. 

## Full Node

Please visit [What is gvite node](../tutorial/node/install.md#What-is-gvite-node) for full node introduction.

## Instructions for Running a Full Node

Configuration details can be found at [Installation](../tutorial/node/install.md#full-node-reward).

## Reward Program Details

* A Snapshot Block Producer (SBP) named "FullNode.Pool" will be set up and maintained by Vite Labs. It will be backed by 10 million VITE votes from the Vite Foundation. All income for "FullNode.Pool", including block creation rewards and voting rewards, will be shared with all eligible full nodes.
* The size of the full node rewards pool varies on a daily basis. It will determine the incentives provided by the Vite Foundation. The calculation is as follows:
    1. On a daily basis, when the size of the full node rewards pool ( x ) is no more than 8,000 VITE, an equal mount will be provided by the Vite Foundation. 
    $$y=x \left\{ 0 \leq x \leq 8000 \right\}$$
    2. On a daily basis, when the size of the full node rewards pool ( x ) is above 8,000 VITE, the incentives are determined by the following equation.
    $$y=\left( x - 8000 \right) \times \frac{8000}{x} + 8000 \left\{ x \gt 8000 \right\}$$
* Rewards will be shared equally by all live (uptime is above 90%) full nodes in the pool.

More details please refer to [Announcement of Full Node Reward Upgrade](https://medium.com/vitelabs/vite-incentive-plan-full-node-reward-program-upgrade-c6e96c6405bb).

::: warning Distinct IP Required
Note that if multiple full nodes run with the same IP address, only one node can get reward. Do NOT setup your nodes with the same IP address! 
:::
