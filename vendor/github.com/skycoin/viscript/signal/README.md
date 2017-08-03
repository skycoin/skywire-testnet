## package "signal"

Able to create signal-server that can communicate with running nodes. Node must run func:
```
signal.InitSignalNode("port").ListenForSignals()
```
And when signal-server can connect to this node with func:
```
AddSignalNodeConn(address string, port string)
```

After connection is set signal-server can send commands to node like ping, res_usage, shutdown.

Also see demo in signal/demo. First run  signal-client.go in signal/demo/client signal/demo/client2.
When run signal-server.go in signal/demo/server. Add running nodes by typing:
```
add_node 0.0.0.0 8001
add_node 0.0.0.0 8008
```

Use commands with appIds 1,2.