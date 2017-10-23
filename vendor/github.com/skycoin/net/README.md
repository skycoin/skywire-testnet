# Skycoin Networking Framework

Skycoin Networking Framework is a simplified TCP and UDP networking framework. 

[Skycoin Messenger](https://github.com/skycoin/net/tree/master/skycoin-messenger) is based on this infrastructure.

#### Skycoin-messenger

[Skycoin Messenger](https://github.com/skycoin/net/tree/master/skycoin-messenger) is an anonymous instant messenger. You can send messages to others by public keys on the messenger.

![messenger](https://blog.skycoin.net/skywire/skywire-and-viscript/messenger.png)

It also provides discovery service, which is using by skywire, cxo and bbs.

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
