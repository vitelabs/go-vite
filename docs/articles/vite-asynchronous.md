# Explain the Vite Asynchronous Architecture with Example

This article will explain the asynchronous architecture of Vite, one of the most important innovations of Vite.

## Synchronous v.s. asynchronous

In synchronous architecture, when a process executes a request, it waits until the execution is complete and then returns the result. The process pauses during the time. After the request is sent, the asynchronous process (usually a thread) runs in the back and the main process goes ahead to handle other executions. When the asynchronous process completes, the main process will receive a notification with the return result.


The advantages of asynchronous architecture are obvious — it will be much faster when running a batch of executions. Vite adopts a DAG-based ledger, where each account has a separate chain. A transaction only changes the state of one account chain, so transactions of multiple accounts can be executed in parallel to achieve higher throughput.


## Asynchronous in Vite

Vite’s asynchronous design mainly lies in three aspects.

  1. Asynchronous request and response;

  2. Asynchronous transaction writing and confirmation;

  3. Asynchronous communication between smart contracts.

Here we focus on the third — the asynchronous communication between smart contracts.
The following is a simple example to introduce both the asynchronous syntax in Solidity++ and the execution process of VVM (Vite virtual machine).

```
pragma soliditypp ^0.4.4;
contract VoteContract {
    address private checkAddr;
    
	mapping(address => uint) public voteMap;
    mapping(address => bool) public invalidAddrsMap;    
   	message checkValid(address addr);

    onMessage vote(address addr, uint voteNum) payable {
        send(checkAddr, checkValid(addr), msg.tokenid, msg.amount); 
        voteMap[addr] = voteMap[addr] + voteNum;
    }

    onMessage isValid(address addr, bool valid) {
        if(!valid && !invalidAddrsMap[addr]) {
           	invalidAddrsMap[addr] = true;
       	}
    }

    getter getVoteNum(address addr) returns(bool isInValid, uint voteNum) {
       	return (invalidAddrsMap[addr], voteMap[addr]);
    }
}
```

As the name shows, VoteContract is a voting contract. Let us take a look at the fields.


voteMap is used to store voting records; invalidAddrs is a map to save all invalid voting addresses; checkAddr is the address of another contract that verifies voting address.

```
pragma soliditypp ^0.4.4;
contract CheckContract {

   	message isValid(address addr, bool valid);

   	onMessage checkValid(address addr) payable {
       	bool result = check(addr);
       	send(msg.sender, isValid(addr, result));
   	}

   	function check(address addr) private returns(bool checkResult) {
       	// To verify the address...
   	}    
}
```

CheckContract performs voting address verification. It has a check function that performs verification and returns checkResult.


## Asynchronous syntax in Solidity++

Unlike Ethereum, smart contracts on Vite are asynchronous. The message listener of a smart contract will listen to incoming messages from other contracts and perform business logic.


In the above example, VoteContract has defined two message listeners: vote and isValid. In the voting process, a user triggers the message listener vote to vote for a specified address addr. vote is payable, which means the user should transfer an amount of voteNum tokens to the smart contract when calling. isValid listens to address check results and puts invalid addresses into map invalidAddrs.


CheckContract has one message listener, checkVaild, which will perform address verification and send results back to isValid. It is also payable.


Solidity++ allows messages to be passed through between smart contracts via message listeners. Therefore, to start writing a smart contract you should declare the message first.


The message must have the same name with a corresponding message listener. In our example, VoteContract declares a message checkValid, which will be sent to the checkValid CheckContract for processing. CheckContract also declares a message isValid to send the verification result to VoteContract.


The syntax for sending a message is send(address addr, message msg, tokenId token, uint amount). Here addr is the contract address, msg stands for the message to be sent, and token and amount specify the amount of token transferred in the message. Please note token and amount are optional; no transfer will be made by default. You can also use addr.transfer(tokenId token, uint amount) to send tokens to another address in the smart contract.


Message listener has no return value. The calling contract will not wait for the result of a message call. As in the above example, VoteContract sends a message checkValid to CheckContract to check the address in message listener vote. When the message is sent, the rest execution will continue. It needs to be pointed out that the message listener is different from an Ethereum function that has no return value. On Ethereum, even if a function call has no code to run, it will still return a result to indicate whether the execution is successful or not. Therefore, function calls on Ethereum are synchronous.


## Asynchronous Vite virtual machine

In this section, let’s take a look at the implementation of asynchronous contract execution in the Vite virtual machine.


Here are some concepts I would like to introduce first:


  1. Transactions on Vite are classified into request transactions and response transactions. Either sending a transfer or calling a contract will create two separate transactions on the blockchain. When the request transaction is complete, it will return immediately without waiting for the completion of the response transaction.

  2. GAS is charged on Ethereum as transaction fees. The EVM will calculate the consumption of GAS and deduct it in the execution of the transaction. Unlike Ethereum, Vite does not charge GAS but replaces it with a different resource called Quota. Because a transaction on Vite is split into a request transaction and a response transaction, the request transaction does not know the quota that will be consumed by the response, quota will be calculated and consumed separately in the request transaction and response transaction.


In the Vite virtual machine, the execution of the request transaction and the response transaction are separated. A request transaction is usually a message sent by the user, in the form of either transfer or contract call. As in the example, the vote message sent by the user to VoteContract is a request transaction. There is also a passive way to initiate request transactions, which will be introduced later.
In the first stage, the Vite virtual machine starts to handle the request transaction. Assuming that there is no exception thrown during the execution, the virtual machine will first calculate the quota consumed by the request transaction, then deduct the corresponding amount of quota from the sender’s account, and finally, update the request transaction block and return.


In the second stage, when the block producing node of the delegated consensus group of VoteContract receives the request transaction, it starts to construct a response block, and the virtual machine will perform a few operations like depth inspection, quota calculation, and balance increment, etc., then execute the code of the message listener in the response transaction. In the example, a checkValid message is sent to CheckContract, in this case, the virtual machine needs to initiate another request transaction to CheckContract after all the business logic in vote is complete. Let me explain in detail.


1. First, the request transaction (to CheckContract) is generated in the response transaction of VoteContract. To handle this situation, the Vite virtual machine keeps a list of request transactions in the response and will send them when the response transaction is complete. Here in the example, a checkValid message is sent from the listener method vote, the latter will continue to run regardless of whether CheckContract sends back a response or not.

2. Second, VoteContract initiates a request transaction within the contract. Please note this internal request transaction does not require quota. In addition, because it carries token transfer, the corresponding amount of token needs to be deducted from the balance. In the example, when more than one users vote, VoteContract will initiate multiple transfers to CheckContract and deduct the balance to reflect the changes.

3. Third, regardless of when and in what order these request messages are received by CheckContract, the final amount received by CheckContract must be equal.


In the third stage, CheckContract will respond to the request sent by VoteContract in a response transaction. More precisely, in the checkValid message listener. The execution process is similar to the second stage, and an isValid message is sent to VoteContract in a new request.


In the fourth stage, a response transaction is initiated by CheckContract to perform the logic in isValid. This is what we mentioned at the beginning of the article: when a request returns a result, the process receives a notification and starts processing.


In the above entire execution process, if an exception occurs in the request, the request transaction will be reverted; if the request transaction is successful and an exception occurs in the response, the response transaction will be reverted, and if the request transaction has changed the balance, a separate request transaction will be initiated by the contract to send the tokens back. For example, the balance of VoteContract will be deducted after checkValid is initiated. If the response of CheckContract fails, a refund will be initiated in another request back to VoteContract.


## getter

Asynchronous message listener has no return values. So in order to get contract states, we introduce the getter method.

```
getter getVoteNum(address addr) returns(bool isInValid, uint voteNum) {
	return (invalidAddrsMap[addr], voteMap[addr]);
}
```

In the example, the getter method getVoteNum is provided to query the contract state in the contract. It has a return value. However, you should remember the following two p when using getter methods.


1. First, you should only use the getter method to query the state of the contract and cannot modify it;

2. Second, getter methods are not allowed to access the data on the blockchain.


Therefore, getter methods are also called off-chain query methods. Actually, they are more like the public method for off-chain queries in Ethereum.


## Summary

This article briefly introduces the asynchronous design and the related implementation in Vite by studying a real example. If you have more questions, don’t hesitate to raise them in Vite’s Telegram group or on Discord. I will be very happy to answer.
