
# 简介

consensus模块主要负责产出各个sbp何时进行出块的信息。

工程上体现：
1. 提供共识信息的订阅接口：subscriber.go#Subscribe
2. 验证共识信息是否正确：consensus_impl.go#VerifySnapshotProducer 和consensus.go#VerifyAccountProducer

# 核心流程

## 共识信息产生的核心流程

- consensus_impl.go#Start  启动共识模块，开启一个新的协程不断更新共识信息
- trigger.go#update        永久循环来计算共识信息并通知订阅方
- consensus_snapshot.go#ElectionIndex    计算共识信息
	- consensus_snapshot.go#genSnapshotProofTimeIndx  通过轮次计算时间
	- chaing/chain.go#GetSnapshotHeaderBeforeTime   从某个时间开始取一个固定的snapshot block
	- consensus_snapshot.go#calVotes  根据该block信息计算余额，并通过余额去计算投票信息，然后计算出块顺序。


总体设计关键是，每次通过index能取到一个稳定不会变化的snapshot block，然后通过这个snapshot block取到的balance也不会发生变化。


## 共识信息的验证流程

- consensus_impl.go#VerifySnapshotProducer 用snapshot block的验证举例
- time_index.go#Time2Index    将时间转换成轮次
- consensus_snapshot.go#ElectionIndex   计算共识信息，并进行验证

共识的验证流程最终也会走到ElectionIndex，和共识信息的产生流程一样。

## 关于缓存的设计

关于缓存，体现在consensus_snapshot.go#calVotes中，该方法需要调用chain来查询全局的余额信息，比较耗时，但由于验证逻辑的存在，该方法调用的频次比较高，所以对calVotes的逻辑进行了缓存。
缓存结果使用leveldb进行存储，实现见cdb/consensus.go#GetElectionResultByHash



# 主要文件的主要作用

- consensus.go  		入口，声明consensus相关的接口
- consensus_impl.go		consensus的实际实现，负责协调内部各个方法
- subscriber.go			维护订阅和回调关系
- trigger.go 			一个协程，不停的计算最新的共识信息
- chain_rw.go 			chain的封装
- core/vote_algo.go		排序算法和随机打乱算法的封装
