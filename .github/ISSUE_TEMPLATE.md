Please make sure you have search your question in our [forum](https://forum.vite.net/).

**Describe the bug briefly**


**Information**

Some necessary information can help use handle your issue quickly. 

- OS

- Hardware (CPU info and Memory size)

- gvite version

- peer\` number, can get from this [API](https://vite.wiki/api/rpc/net.html#net-peers)

- Startup log, is in file `gvite.log`, usually in the same directory which gvite belongs.

    Command ```grep -i "error" gvite.log``` can find the startup error. If the output is not null, then paste it into this issue.


- Runtime log, runtime log directory will be printed into gvite.log when gvite start.

    1. You can find it by input ```grep -i "NodeServer.DataDir" gvite.log``` on your terminal. eg. the terminal output:
    ```bash
    t=2019-02-27T15:17:41+0800 lvl=info msg=NodeServer.DataDir:/root/.gvite/testdata module=gvite/node_manager
    ```
    
    2. The log directory is `/root/.gvite/testdata/runlog`. Every startup will create a directory named by Date. eg.
    ```bash
    2019-03-19T13-20  2019-03-21T22-52  2019-03-22T10-36 
    ```
    
    3. Go into the latest directory, fileList maybe like following:
    ```bash
    error  vite-2019-03-22T10-57-22.797.log.gz  vite-2019-03-22T11-25-25.901.log.gz  vite.log
    ```
    Some log files are compressed, decompress them by ```gzip -d *.log.gz```.
    
    4. You can grep the keyword in logfile, eg. "download" is about sync file download. ```grep -i "download" vite.log``` will show:
    ```bash
    t=2019-03-22T11:25:31+0800 lvl=info msg="download <file subgraph_2581201_2584800> from 108.61.170.32:8484 elapse 7.846745653s" module=net/fileClient
    t=2019-03-22T11:25:38+0800 lvl=info msg="begin download <file subgraph_2584801_2588400> from 108.61.170.32:8484" module=net/fileClient
    t=2019-03-22T11:25:47+0800 lvl=info msg="download <file subgraph_2584801_2588400> from 108.61.170.32:8484 elapse 8.800968866s" module=net/fileClient
    t=2019-03-22T11:25:51+0800 lvl=info msg="begin download <file subgraph_2588401_2592000> from 108.61.170.32:8484" module=net/fileClient
    t=2019-03-22T11:26:02+0800 lvl=info msg="download <file subgraph_2588401_2592000> from 108.61.170.32:8484 elapse 11.184523779s" module=net/fileClient
    t=2019-03-22T11:26:05+0800 lvl=info msg="begin download <file subgraph_2592001_2595600> from 108.61.170.32:8484" module=net/fileClient
    t=2019-03-22T11:26:11+0800 lvl=info msg="download <file subgraph_2592001_2595600> from 108.61.170.32:8484 elapse 6.579606422s" module=net/fileClient
    ```
    
    5. Tail the `error/vite.error.log`.
    
    6. Paste the log info get by previous step4 and step5 into this issue.
