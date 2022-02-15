# 简介

wallet的主要职责是做钱包的管理，主要实现一下功能：

- 随机生成助记词并创建钱包；manager.go#NewMnemonicAndEntropyStore 
- 通过助记词生成钱包；manager.go#RecoverEntropyStoreFromMnemonic
- 钱包解锁；manager.go#Unlock
- 签名；	account.go#Sign
- 验证签名； account.go#Verify


助记词一旦导入，会生成一个使用第一个地址命名的文件，该文件使用密码进行加密，后续可以通过密码和该文件恢复助记词。


钱包签名采用ed25519+Blake2b

# Usages

see: manager_test.go#TestManage