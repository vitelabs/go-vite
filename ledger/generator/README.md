# 简介


该目录的核心逻辑是构造一笔交易，是VM调用的一个入口。

场景是：
1. 使用节点钱包发送交易  				rpcapi/api/wallet.go#CreateTxWithPassphrase
2. 合约共识组生成合约的receive block	ledger/onroad/taskprocessor.go#processOneAddress
3. 验证一个recive block的合法性			ledger/verifier/account_verifier.go#vmVerify


