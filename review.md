

# 简介

先介绍每个目录的用途：

- .github 用来存放与github相关的配置，目前github action的配置存放在其中
- bin 一些在开发和部署阶段可能会用到的脚本
- client 实现了远程调用gvite节点的go版本客户端
- cmd 命令行脚本，gvite的入口就是这里cmd/gvite/main.go
- common 一些公用的方法组建以及常量的定义
- conf 存放配置文件，conf/node_config.json是主网默认的
- contracts-vite 存放vite相关的智能合约，contracts-vite/contracts/ViteToken.sol就是部署在eth网络上的VITE ERC20 Token的源码
- crypto 加密算法和hash算法的实现 ed25519和blake2b-512是vite链上的主要算法
- docker 存放dockerfile，配合docker编译时使用
- docs go-vite相关的wiki存放，链接文档 https://docs.vite.org/go-vite/
- interfaces 一些模块的接口定义放到这里，解决golang循环依赖问题
- ledger 包含账本相关的实现
- log15 go-vite中采用的日志框架，由于当初改了不少东西，就copy过来了，原项目地址：https://github.com/inconshreveable/log15
- monitor 压力测试时，为了方便数据统计写的工具包
- net 网络层实现，包含了节点发现，节点互通，账本广播和拉取
- node 一个ledger的上层包装，主要做一些组合的功能
- pow 本地实现的pow计算和远程调用pow-client进行计算pow
- producer 触发snapshot block和contract block生产的入口
- rpc 对于websocket/http/ipc三种远程调用方式的底层实现
- rpcapi 内部各个模块对外暴露的rpc接口
- smart-contract 类似contracts-vite
- tools 类似collection，一些集合的实现
- version 存放每次编译的版本号，make脚本会修改version/buildversion的内容
- vm 虚拟机的实现和内置合约的实现，里面大量的虚拟机执行逻辑
- vm_db 虚拟机在执行的时候需要的存储接口，是虚拟机访问chain的封装
- wallet 钱包的实现，私钥助记词管理，以及签名和验证签名


然后是gvite的实现整体框架：

主要分为以下几个大的模块：
1. net   网络互通
2. chain 底层存储
3. vm    虚拟机的执行
4. consensus  负责共识出块节点
5. pool  负责分叉选择
6. verifier 包装各种验证逻辑，是vite协议的包装
7. onroad 负责合约receive block的生产

# 数据流

## 用户发送一笔转账交易

- rpcapi/api/ledger_v2.go#SendRawTransaction    	 		rpc是所有访问节点的入口
- ledger/pool/pool.go#AddDirectAccountBlock		 			通过pool，将block直接插入到账本中
- ledger/verifier/verifier.go#VerifyPoolAccountBlock  		调用verifier，验证block的有效性
- ledger/chain/insert.go#InsertAccountBlock		 			调用chain的接口插入到链中
- ledger/chain/index/insert.go#InsertAccountBlock           维护block的hash关系，例如高度与hash间关系，send-receive之间关系等等，涉及存储文件ledger/index/*和ledger/blocks
- ledger/chain/state/write.go#Write							将block改变的state进行维护，例如余额，storage等，涉及存储文件ledger/state


## 生成一个合约的receive block

- ledger/consensus/trigger.go#update 										产生一个合约开始出块的事件，传递到下游
- producer/producer.go#producerContract										接收共识层消息，并将消息传递给onroad
- ledger/onroad/manager.go#producerStartEventFunc							onroad内部启动协程，单独处理待出块的send block
- ledger/onroad/contract.go#ContractWorker.Start							将所有合约按照配额进行排序，依次进行出块
- ledger/onroad/taskprocessor.go#ContractTaskProcessor.work					从已经排序好的队列中取出某个合约地址，进行处理
- ledger/onroad/taskprocessor.go#ContractTaskProcessor.processOneAddress 	针对该合约进行处理
- ledger/generator/generator.go#GenerateWithOnRoad							依据一个send block进行生成receive block
- vm/vm.go#VM.RunV2															运行vm逻辑
- ledger/onroad/access.go#insertBlockToPool									将生成好的block插入到pool中
- ledger/pool/pool.go#AddDirectAccountBlock		 							通过pool，将block直接插入到账本中

后面的流程参考用户发送一笔转账交易

## sbp生成一个snapshot block

- ledger/consensus/trigger.go#update 										产生一个snapshot block开始出块的事件，传递到下游
- producer/worker.go#produceSnapshot										接收共识层消息，并单独开辟一个协程来进行snapshot block生成
	- producer/worker.go#randomSeed											计算随机数种子
	- producer/tools.go#generateSnapshot
		- ledger/chain/unconfirmed.go#GetContentNeedSnapshot				计算有哪些account block需要快照
	- producer/tools.go#insertSnapshot										
		- ledger/pool/pool.go#AddDirectSnapshotBlock                        插入snapshot block到账本中
			- ledger/verifier/snapshot_verifier.go#VerifyReferred			验证snapshot block的合法性
			- ledger/chain/insert.go#InsertSnapshotBlock					插入snapshot block到chain中
				- ledger/chain/insert.go#insertSnapshotBlock				更新indexDB,stateDB

