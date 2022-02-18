# 简介

该模块是核心模块之一，所有的account block和snapshot block插入到chain都是经过pool的组织。

实现只要功能：
1. 保证一个account chain串行的插入；
2. 保证snapshot chain串行的插入；
3. 保证snapshot chain插入时会有account chain插入；
4. 尽可能快的插入多个account chain；
5. account chain的分叉选择；
6. snapshot chain的分叉选择；



核心入口：

- pool/work.go#work   				单独启动一个协程，将内存中存在的account block数据和snapshot block数据进行排序和组合，然后进行验证，有顺序的插入到账本中；
- pool/pool_batch.go#insert 		将内存中的数据整理/验证/依序插入到账本（广播过来的数据）
- pool/pool_batch_chunk.go#insertChunks		将从p2p批量下载的账本数据，依次的插入到账本中（通常发生在节点大批量同步时）


