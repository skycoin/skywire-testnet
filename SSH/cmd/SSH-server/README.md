# Skywire SSH app

`SSH-server` app implements SSH functionality over skywirenet.

`SSH-cli` is used to initiate communication via client RPC
exposed by `SSH` app. 

`SSH` app implements common SSH operations:

- starting remote shell
- and executing commands remotely

PubKey whitelisting is performed by adding public key to the
authentication file (`$HOME/.skywire-messenger-ssh/authorized_keys` by default).

** Local setup

Create 2 node config files:

`skywire1.json`

```json
  "apps": [
    {
      "app": "SSH-server",
      "version": "1.0",
      "auto_start": true,
      "port": 2
    }
  ]
```

`skywire2.json`

```json
  "apps": [
    {
      "app": "SSH-client",
      "version": "1.0",
      "auto_start": true,
      "port": 22
    }
  ]
```

Compile binaries and start 2 nodes:

```bash
$ go build -o apps/SSH-server.v1.0 ./skywire-messenger-ssh/cmd/server
$ go build -o apps/SSH-client.v1.0 ./skywire-messenger-ssh/cmd/client
$ go build ./sjywire-messenger-ssh/cmd/cli
$ ./skywire-node skywire1.json
$ ./skywire-node skywire2.json
```

Add public key of the second node to the auth file:

```bash
$ mkdir ~/.SSH
$ echo "0348c941c5015a05c455ff238af2e57fb8f914c399aab604e9abb5b32b91a4c1fe" > ~/.skywire-messenger-ssh/authorized_keys
```

Connect to the first node using CLI:

```bash
$ ./SSH-cli 024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7
```

This should get you to the $HOME folder of the user(you in this case), which
will indicate that you are seeing remote PTY session.
