# 简介


该目录是锦上添花的功能，为了加速节点加载账本而实现的功能。

实现了将本地目录包装成一个pipeline，然后依次写入账本的功能。

配合cmd/subcmd_loadledger/load_ledger.go而实现的功能。


# 主要流程

模型上，是一个生产者消费者模型，pipeline负责将磁盘中的现有blockDB中的block数据，顺序的转换成block信息，并写入到channel中，外部可以从channel中依次的消费数据

1. pipeline_blocks.go#NewBlocksPipeline				从一个目录new一个pipeline
2. pipeline_blocks.go#newBlocksPipelineWithRun		启动一个协程向channel中阻塞的写入block数据
