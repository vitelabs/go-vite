
chain目录代表着go-vite的存储层，所有链上数据均会落到chain中。

chain提供能改变链上状态的的写入功能是：
1. snapshot block的插入；
2. account block的插入；

为了需要，在存储层主要需要存储以下数据：

1. 快照数据（snapshot block）
2. 交易数据（account block）
3. 用户状态（多代币余额等）
4. 合约状态（合约storage、合约meta、合约调用深度等）

为此设计了三个db来进行存储信息：
1. blockDB  
2. indexDB
3. stateDB


### blockDB

存储snapshot block和account block数据，底层是一个个小文件，各个小文件链接在一起，可以构成一个大文件。
snapshot block和account block顺序存储在其中，append写入，能够大幅度提高写入速度。

### indexDB
存储索引，底层为leveldb，主要包含以下索引：
1. 每个block在blockDB中的location信息；
2. snapshot hash -> height
3. account hash -> height
4. address+height -> hash
5. receive block <-> send block
6. account block与snapshot block之间关系
7. ................

### stateDB

存储状态信息，底层用leveldb进行实现，主要包含以下信息：
1. 用户余额
2. 合约meta
3. 合约storage信息
4. 合约当前调用栈深度（Vite支持合约异步调用合约，因此需要合约调用栈深度信息）等

