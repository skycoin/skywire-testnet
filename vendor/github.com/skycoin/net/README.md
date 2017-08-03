# Skycoin Networking Framework



## Protocol

```
                  +--+--------+--------+--------------------+
msg protocol      |  |        |        |                    |
                  +-++-------++-------++---------+----------+
                    |        |        |          |
                    v        |        v          v
                  msg type   |     msg len    msg body
                   1 byte    v     4 bytes
                          msg seq
                          4 bytes



                  +-----------+--------+--------------------+
normal msg        |01|  seq   |  len   |       body         |
                  +-----------+--------+--------------------+


                  +-----------+
ack msg           |80|  seq   |
                  +-----------+


                  +--------------------+
ping msg          |81|    timestamp    |
                  +--------------------+


                  +--------------------+
pong msg          |82|    timestamp    |
                  +--------------------+
```


## Client Example

```
tcpFactory := factory.NewTCPFactory()
conn, err := tcpFactory.Connect(":8080")
if err != nil {
   panic(err)
}
for {
   select {
   case m, ok := <-conn.GetChanIn():
      if !ok {
         return
      }
      log.Printf("received msg %s", m)
   }
}
```

## Server Example

```
go func() {
   log.Println("listening udp")
   udpFactory := factory.NewUDPFactory()
   udpFactory.AcceptedCallback = func(connection *factory.Connection) {
      connection.GetChanOut() <- []byte("hello")
   }
   udpFactory.Listen(":8081")
}()
log.Println("listening tcp")
tcpFactory := factory.NewTCPFactory()
tcpFactory.AcceptedCallback = func(connection *factory.Connection) {
   connection.GetChanOut() <- []byte("hello")
}
if err := tcpFactory.Listen(":8080"); err != nil {
   panic(err)
}
```