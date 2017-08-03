# Skycoin Messenger

## How to test

`cd skycoin-messenger/server`

`go run main.go`

`cd skycoin-messenger/client`

`go run main.go`

## Client Websocket Protocol
```
                   +--+--------+----------------------------------------------+
                   |  |        |                                              |
                   +-++-------++-------------------------+--------------------+
                     |        |                          |
                     v        v                          v
                  op type    seq                     json body
                   1 byte   4 byte

                   +----------------------------------------------------------+
           reg     |00|  seq   |{"Address":"", "PublicKey":""}                |
                   +----------------------------------------------------------+
       ^
       |           +----------------------------------------------------------+
  req  |   send    |01|  seq   |{"PublicKey":"", "Msg":""}                    |
       |           +----------------------------------------------------------+
       |
+--------------------------------------------------------------------------------------+
       |
       |           +-----------+
  resp |   ack     |00|  seq   |
       |           +-----------+
       v
                   +----------------------------------------------------------+
           push    |01|  seq   |{"PublicKey":"", "Msg":""}                    |
                   +----------------------------------------------------------+
```

## Flow Chart

```
node                                    server                                    node

+--+                                     +--+                                     +--+
|  |                                     |  |                                     |  |
|  | +------+register+pubkey+----------> |  | <------+register+pubkey+----------+ |  |
|  |                                     |  |                                     |  |
|  | <-------------+ack+---------------+ |  | +-------------+ack+---------------> |  |
|  |                                     |  |                                     |  |
|  | +------+send+msg+to+pubkey+-------> |  | +------+forward+msg+to+pubkey+----> |  |
|  |                                     |  |                                     |  |
|  | <-------------+ack+---------------+ |  | <-------------+ack+---------------+ |  |
|  |                                     |  |                                     |  |
|  | <------+forward+resp+msg+---------+ |  | <------+resp+msg+to+pubkey+-------+ |  |
|  |                                     |  |                                     |  |
|  | +-------------+ack+---------------> |  | +-------------+ack+---------------> |  |
|  |                                     |  |                                     |  |
|  |                                     |  |                                     |  |
|  |                                     |  |                                     |  |
|  |                                     |  |                                     |  |
+--+                                     +--+                                     +--+
```

## TCP Client Example

### Client 0xf1

```
f := factory.NewMessengerFactory()
conn, err := f.Connect(":8080")
if err != nil {
   panic(err)
}

key := cipher.PubKey([33]byte{0xf1})
err = conn.Reg(key)
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

### Client 0xf2

```
f := factory.NewMessengerFactory()
conn, err := f.Connect(":8080")
if err != nil {
   panic(err)
}

key := cipher.PubKey([33]byte{0xf2})
conn.GetChanOut() <- factory.GenRegMsg(key)

f1 := cipher.PubKey([33]byte{0xf1})
conn.GetChanOut() <- factory.GenSendMsg(key, f1, []byte("Hello 0xf1 1"))
conn.GetChanOut() <- factory.GenSendMsg(key, f1, []byte("Hello 0xf1 2"))
conn.Write(factory.GenSendMsg(key, f1, []byte("Hello 0xf1 3")))
conn.Write(factory.GenSendMsg(key, f1, []byte("Hello 0xf1 4")))
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

## RPC Client Example

Look inside rpc/rpc_test.go

```
client, err := rpc.DialHTTP("tcp", ":8083")
if err != nil {
    log.Fatal("dialing:", err)
}

var code int
key := cipher.PubKey([33]byte{0xf3})
err = client.Call("Gateway.Reg", &op.Reg{PublicKey:key.Hex(), Address:":8080"}, &code)
if err != nil {
    log.Fatal("calling:", err)
}
t.Log("code", code)

_ = msg.PUSH_MSG
target := cipher.PubKey([33]byte{0xf1})
err = client.Call("Gateway.Send", &op.Send{PublicKey:target.Hex(), Msg:"What a beautiful day!"}, &code)
if err != nil {
    log.Fatal("calling:", err)
}
t.Log("code", code)

var msgs []*msg.PushMsg
err = client.Call("Gateway.Receive", 0, &msgs)
if err != nil {
    log.Fatal("calling:", err)
}
t.Logf("%v", msgs)
```

