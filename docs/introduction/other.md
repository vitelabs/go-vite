# Other Designs

::: tip
This document is only for introduction. if you need more detail on vite, please move to [White Paper](https://github.com/vitelabs/whitepaper/blob/master/vite_en.pdf)
:::

Vite’s design also comes with a few other features.  They are practical solutions to problems and pain points in the realm of decentralized applications. The development of Vite's community and implementation of the VEPs( Vite Enhancement Proposals) will drive continual improvement of Vite protocol.

Vite’s protocol supports multiple tokens. Unlike the ERC20 scheme, user-issued tokens and Vite tokens share the same underlying trading protocol.  This ensures identical level of security between types of tokens, and obviates security concerns due to poor implementation (e.g., those caused by stack overflow). 

Vite will have the Loopring protocol built in, enabling exchange of multiple tokens. Therefore, the Vite wallet has functions of a decentralized exchange.

In addition, Vite proposed the VCTP cross-chain protocol that allows the transfer of assets between chains. The Vite team will implement the cross chain gateways with Ethereum.  Cross-chain gateways with other target chains will open to the community.

The resource allocation mechanism in Vite is based on the stake of VITE tokens, the one-time fee, and the difficulty of PoW for a transaction.  The mechanism provides flexibility for users with different needs. Users can choose to stake  VITE for a period, to get fixed amount of TPS quotas. They can also pay a one-time fee to increase quotas temporarily.  They could also lease resource quotas from existing VITE stake holders.  Lastly, they could even obtain quotas by computing  a PoW.

Moreover, the design of Vite includes the naming services, contract updating, timer scheduling, Solidity++ standard library, block pruning, and so on.


[The white paper contains more specifics](https://github.com/vitelabs/whitepaper/blob/master/vite_en.pdf).
