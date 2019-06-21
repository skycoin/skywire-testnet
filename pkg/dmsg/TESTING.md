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
  - clientA and clientB is connected to a server.
- When:
  - The server restarts.
- Then:
  - Both clients will automatically reconnect to the server.
  - Transports can be established between clientA and clientB.

**`server_disconnect_should_close_transports`**

- TODO

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

