# 简介

本模块主要负责合约的receive block的生成。


- manager.go				核心数据结构维护在Manager中，其中，对每一个gid都维护一个worker
- worker.go					定义worker的接口，每个worker对应一个共识组
	- contract.go			合约共识组的worker，处理该gid下所有合约交易，并维护一组contractTaskProcessor，并行的处理合约交易
	- taskprocessor.go		任务调度和处理
- task_pqueue.go#contractTaskPQueue			合约处理核心数据结构，待处理的地址进行排序，主要是按照配额进行排序
- manager.go#onRoadPools	维护着待receive的block，参见ledger/onroad/pool