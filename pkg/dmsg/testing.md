# Testing

This document is to specify the tests currently missing within `dmsg` (and that should be tested).

## Integration Tests

### Fault Injection testing

The individual entities of `dmsg` (`dmsg.Client` and `dmsg.Server`), should be capable of dealing with erroneous (and in the future, malicious) behaviour of remote entities.

Note that even though `messaging-discovery` is also considered to be an entity of `dmsg`, however it's mechanics are simple and will not be tested here.

#### Ensure that `Client.Serve()` does not hang

**`failed_accepts_should_not_result_in_hang`**

- Given:
  - clientA is connected to clientB via a server.
  - clientB is NOT calling `.Accept()`.
  - clientA dials transports to clientB until failure.
- When:
  - clientA tries to dial to clientC.
  - clientC tries to dial to clientA.
- Then:
  - Transports established should still read/write as expected.

**`failed_accepts_should_not_result_in_hang_2`**

- Given:
  - clientA has transports established with various remotes.
  - The transports are being written/read to in a continuous loop.
- When:
  - clientA dials to clientB (which is not calling `.Accept()`) until failure.
- Then:
  - read/writes to existing transports should not hang.

**`capped_transport_buffer_should_not_result_in_hang`**

- Given:
  - A transport is establised between clientA and clientB.
  - clientA writes to clientB until clientB's buffer is capped

### Fuzz testing

We should test the robustness of the system under different conditions and random order of events. Integration-based tests should be written consisiting of x-number of servers, clients and a single 