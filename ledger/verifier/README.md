
# 简介


该模块是协议的封装，主要职责是验证account block和snapshot block的合法性。

核心接口：

1. verifier.go#VerifyNetSnapshotBlock  	验证snapshot block的hash和签名，主要供网络层使用；
2. verifier.go#VerifyNetAccountBlock	验证account block的hash和签名，主要供网络层使用
3. verifier.go#VerifyPoolAccountBlock	验证account block整体，包括vm执行结果
4. verifier.go#VerifyNetSnapshotBlock	验证snapshot block整体，包括快照数据等


