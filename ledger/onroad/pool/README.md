# 简介


一个存储地址和待接收交易的数据结构，支持添加和删除。
添加场景：
1. 新的send block产生；
2. 某个receive block回滚；

删除场景：
1. send block产生了receive block；
2. send block回滚；



核心数据结构：contract_pool.go#contractOnRoadPool->callerCache

维护一个三级关系：

- 合约地址receiver
	- sender地址
		- send block list （按照高度排序）

主要是满足以下约束，同一个地址的send block需要按顺序被receive。







