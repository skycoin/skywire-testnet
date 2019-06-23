# Testing

This document is to specify the tests currently missing within `dmsg` (and that should be tested).

## Integration Tests

### Fault Injection testing

The individual entities of `dmsg` (`dmsg.Client` and `dmsg.Server`), should be capable of dealing with erroneous (and in the future, malicious) behaviour of remote entities.

Note that even though `messaging-discovery` is also considered to be an entity of `dmsg`, however it's mechanics are simple and will not be tested here.

**`failed_accepts_should_not_result_in_hang`**

- Given:
  - clientA is connected to clientB via a server.
  - There is already a single transport established between clientA and clientB.
  - The single transport is being written/read to/from in a continuous loop.
- When:
  - clientA dials transports to clientB until failure (in which clientB does not call `.Accept()`).
- Then:
  - Read/writes to/from the existing transport should still work.

**`capped_transport_buffer_should_not_result_in_hang`**

- Given:
  - A transport is established between clientA and clientB.
  - clientA writes to clientB until clientB's buffer is capped (or in other words, clientA's write blocks).
- When:
  - clientB dials to clientA and begins reading/writing to/from the newly established transport.
- Then:
  - It should work as expected still.

**`reconnect_to_server_should_succeed`**

- Given:
  - clientA and clientB are connected to a server.
- When:
  - The server restarts.
- Then:
  - Both clients will automatically reconnect to the server.
  - Transports can be established between clientA and clientB.

**`server_disconnect_should_close_transports`**

- Given:
  - clientA and clientB are connected to a server
  - clientB dials clientA
  - clientA accepts connection
  - Transports are being created
  - Some read/write operations are performed on transports
  - Server disconnects
- Then:
  - Transports should be closed
  
**`server_disconnect_should_close_transports_while_communication_is_going_on`**

- Given:
  - clientA and clientB are connected to a server
  - clientB dials clientA
  - clientA accepts connection
  - Transports are being created
  - Read/write operations are being performed
  - Server disconnects
- Then:
  - Transports should be closed

**`self_dial_should_work`**

- Given:
  - clientA is connected to a server
  - clientA dials himself
- Then:
  - clientA accept connections, transports are being created successfully
  - clientA is able to write/read to/from transports without errors

### Fuzz testing

We should test the robustness of the system under different conditions and random order of events. These tests should be written consisting of x-number of servers, clients and a single discovery.

The tests can be event based, with a probability value for each event.

Possible events:
1. Start random server.
2. Stop random server.
3. Start random client.
   1. With or without `Accept()` handling.
   2. With or without `transport.Read()` handling.
4. Stop random client.
5. Random client dials to another random client.
6. Random write (in len/count) from random established transport.

Notes:
1. We have a set number of possible servers and we are to start all these servers prior to running the test. This way the discovery has entries of the servers which the clients can access when starting.
2. We may need to log the "events" that happen to calculate the expected state of the system
and run the check every x "events".


For this test we must have a set up system consisting of X number of servers, Y number of clients, Z number of transports and a single discovery.
Also we need some kind of control panel from which we will run events. Events maybe picked as following:
  - each event has it's own probability
  - first, we pick a random number of events to be executed
  - second, we pick a corresponding number of events, each of them picked randomly
  - third, based on the probability of each event we calculate whether it will be executed or not
  - finally, we execute all the winner-events in goroutines
  
Before running each of the picked events we may need to take a snapshot of the whole system to check consistency

Event type may be an struct containing function with some signature. Also this struct should have probability of the event, maybe the type of the event. 

Running event should result in a snapshot of system's previous state. Snapshot should allow to simulate the event and return a new state. So this way we:
  - run a series of N events, get a series of N snapshots
  - for snapshots 0...N we pick (I)th snapshot and simulate an event on it which results in a new state. This new state may then be compared to the (I+1)th snapshot for consistency.
  
Snapshot should be able to dump the state in some form comfortable to examine in case something is wrong