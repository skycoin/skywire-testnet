# Skywire SOCKS5 proxy app

`socksproxy` app implements SOCKS5 functionality over skywire
net.
Any conventional SOCKS5 client should be able to connect to the
proxy client.
Currently the server supports authentication with a user and passcode pair
that are set in the configuration file.
If none are provided, the server does not require authentication.

## Local setup

Create 2 node config files:

- `skywire1.json`

```json
{  
  "apps": [
    {
      "app": "socksproxy",
      "version": "1.0",
      "auto_start": true,
      "port": 3,
      "args": ["-passcode", "123456"]
    }
  ]
}
```

- `skywire2.json`

```json
{
  "apps": [
    {
      "app": "socksproxy-client",
      "version": "1.0",
      "auto_start": true,
      "port": 33,
      "args": ["-srv", "024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7"]
    }
  ]
}
```

Compile binaries and start 2 nodes:

```sh
$ go build -o apps/socksproxy.v1.0 ./cmd/apps/therealproxy
$ go build -o apps/socksproxy-client.v1.0 ./cmd/apps/therealproxy-client
$ ./skywire-visor skywire1.json
$ ./skywire-visor skywire2.json
```

You should be able to connect to a secondary node via `curl`:

```sh
$ curl -v -x socks5://123456:@localhost:1080 https://api.ipify.org
```
