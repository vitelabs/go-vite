# An Overview of Vite Storage Layer


Get to know the magics of Vite storage? You should never miss out.


The storage layer is responsible for the persistence of various data on the Vite chain, and providing key functions such as caching and sophisticated query. The data stored in Vite storage layer includes transactions (Transaction/AccountBlock), snapshots (SnapshotBlock), account states (AccountState), and virtual machine states (VmState).


## Business Requirements

Each type of data satisfies a special use case with unique data reading/writing requirements:


### Transaction

In Vite’s DAG ledger, a transaction (Tx/Transaction) presents as an AccountBlock. Except in some rare situation, a transaction is always initiated by a FromAddress and sent to a ToAddress, and called SendTx. When the account of ToAddress receives the transaction, a ReceiveTx will be generated and linked to the SendTx. Similar to other blockchains, it is possible to use the Tx hash to query the transaction, or find all transactions belonging to a certain account by the address on Vite. Additionally, querying based on the relationship between SendTx and ReceiveTx must also be supported.


### Snapshot

The snapshot chain is a special blockchain on Vite. Each SnapshotBlock in the snapshot chain stores the basic information of all transactions in the snapshot, and a link relationship is established between the SnapshotBlock and each Tx that has been snapshotted. Therefore, in addition to the query and traversal demands of the SnapshotBlock, SnapshotBlock and Tx must also be queried based on the indexed relationship between SnapshotBlock and Tx.


### Account state and virtual machine state

The account state and the virtual machine state are very similar. They are both storage data structures used to indicate the state of an address. The difference is that the account state is associated with a common account, while the virtual machine state belongs to a contract. The state keeps changing during the execution of a transaction, so the state storage must be versioned to satisfy the update, traversal and revert. In addition, in order to have the external applications being aware of the state change in time, Event/VmLog should be supported for the convenience trace of changes.


## System Design

### Conceptual design

The general design purpose is to provide support for upper-layer business modules and improve reusability and system reliability by following the subsequent guidelines.

1. Small file storage

Using small files of fixed size for both temporary storage and permanent storage of blocks. The performance of incremental writing and batch reading of small files is highly efficient, and it also takes into account the needs of random reading. It is very suitable for storing the massive blockchain transactions.


2. Indexing

Using LevelDB to implement index storage. LevelDB has an excellent performance in batch incremental writing, which is very suitable for blockchain that has more writes but fewer updates. LevelDB supports endianness that facilitates the read/write of multi-version states through customized keys. It can also perform K-V based random reading and writing.


3. Cache

Using cache to store hot data to make full use of the performance advantage of memory to speed up reads. Cache can be implemented to align with certain strategies.


4. Asynchronous flush

In order to improve I/O performance, the persistent storage of data is flushed asynchronously. Two-phase commit is introduced to ensure data consistency, while ‘redolog’ is used to avoid the loss of uncommitted data.


5. Data compression

Using data compression to reduce the amount of data stored.


### Engineering implementation

Vite storage layer is implemented as the following three modules.


1. blockDB

blockDB realizes the storage of AccountBlock and SnapshotBlock. Considering that the data format is fixed, most blocks have a fixed size, and storing in small files. Multiple blocks are allowed to store in one file to reduce fragments and facilitate indexing.


2. indexDB

indexDB is used to index the location of blocks, and it also stores the relationship between various blocks.


3. stateDB

stateDB is used to store account states and virtual machine states. By carefully designing the byte positions of the LevelDB key, it can support multiple versions of data.


## Summary

This article briefly introduces a high-level perspective on how the Vite storage layer has been designed in alignment with actual business needs. In the next article, we start to introduce the design details of the three modules mentioned above. Stay tuned!

