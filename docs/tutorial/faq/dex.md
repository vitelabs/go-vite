# How to Integrate 3rd Party Exchange

Follow the below steps to integrate Vite (as well as the other coins issued on Vite) in a 3rd-party exchange.

1. [Get to Understand the Vite Chain](../../introduction/README.md)

2. [Set up Your Node](../node/install.md)

3. Process Deposit

You can choose either **Bank Account Style Wallet** (based on individual addresses) or **Shared Account Style Wallet** (based on memo) as the deposit account. A memo-based deposit account is recommended. 

* Generate a deposit address through [Node Wallet API](../../api/rpc/wallet_v2.md#wallet_createentropyfile)
* User deposits to the address with a unique deposit id (memo).
* The gateway polls for [Unreceived Transactions](../../api/rpc/ledger_v2.md#ledger_getunreceivedblocksbyaddress) on the deposit address, [Receive the Transactions](../../api/rpc/ledger_v2.html#ledger_sendrawtransaction), then generate deposit records.
* The gateway checks the deposit [Transaction Confirmations](../../api/rpc/ledger_v2.md#ledger_getaccountblockbyhash) periodically. When the confirmation number reaches **180**, it is safe to believe the deposit is complete.

4. Process Withdrawal

* The gateway sends a [Withdrawal Transaction](../../api/rpc/ledger_v2.md#ledger_sendrawtransaction) to user's withdrawal address and generate a record.
* The gateway checks the withdrawal [Transaction Confirmations](../../api/rpc/ledger_v2.md#ledger_getaccountblockbyhash) periodically. When the confirmation number reaches **180**, it is safe to believe the withdrawal is complete.

:::tip
For convenience, [Vite JavaScript SDK](https://vite.wiki/api/vitejs/) and [Vite Java SDK](https://vite.wiki/api/javasdk/) can be used to call the above RPC APIs.
:::
