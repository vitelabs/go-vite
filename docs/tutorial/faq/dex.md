# How to Integrate Vite into Exchange

Follow the below steps to integrate Vite into a 3rd-party exchange. This guide also applies to any other tokens issued on Vite.

1. [Get to Understand the Vite Chain](../../introduction/README.md)

2. [Set up Your Node](../node/install.md)

3. Process Deposit

You can choose either **Bank Account Style Wallet** (based on individual addresses) or **Shared Account Style Wallet** (based on memo) as the deposit account. For the simplicity of integration, it's highly recommended to use a memo-based deposit account. 

* Exchange generates a deposit address through [Node Wallet API](../../api/rpc/wallet_v2.md#wallet_createentropyfile)
* User deposits VITE to the address with a unique deposit id (memo).
* Exchange listens to [Unreceived Transactions](../../api/rpc/ledger_v2.md#ledger_getunreceivedblocksbyaddress) for the deposit address, [Receive the Transactions](../../api/rpc/ledger_v2.html#ledger_sendrawtransaction), then generates deposit records.
* Exchange checks the [Transaction Confirmations](../../api/rpc/ledger_v2.md#ledger_getaccountblockbyhash) for the deposit transaction. When the confirmation number reaches **180**, it is safe to believe the deposit is complete.

4. Process Withdrawal

* Exchange sends a [Withdrawal Transaction](../../api/rpc/ledger_v2.md#ledger_sendrawtransaction) to user-specified withdrawal address and generates a withdrawal record.
* Exchange checks the [Transaction Confirmations](../../api/rpc/ledger_v2.md#ledger_getaccountblockbyhash) for the withdrawal transaction. When the confirmation number reaches **180**, it is safe to believe the withdrawal is complete.

:::tip
For convenience, we recommend using [Vite JavaScript SDK](https://vite.wiki/api/vitejs/) or [Vite Java SDK](https://vite.wiki/api/javasdk/) to access the above RPC APIs.
:::
