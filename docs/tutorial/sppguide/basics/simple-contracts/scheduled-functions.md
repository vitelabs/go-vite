---
order: 6
---
# Scheduled Functions

This contract, once initiated, proceeds to call itself and repeatedly check the current time. If enough time has elapsed it prints a message and stops.

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/schedule.solidity

#### Key Features

- `onMessage()` the fallback message listener will execute on any received message that does not have a cooresponding message listener.

- This requires a non-zero `ResponseLatency` and is intended to execute over many different snapshot block heights.

- `block.timestamp` gets the current block's timestamp (seconds since unix epoch).
