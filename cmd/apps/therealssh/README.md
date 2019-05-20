# Skywire SSH app

`therealssh` app implements SSH functionality over skywirenet.

`therealssh-cli` is used to initiate communication via client RPC
exposed by `therealssh` app. 

`therealssh` app implements common SSH operations:

- starting remote shell
- and executing commands remotely

PubKey whitelisting is performed by adding public key to the
authentication file (`$HOME/.therealssh/authorized_keys` by default).

** Local setup

Create 2 node config files:

`skywire1.json`

```json
  "apps": [
    {
      "app": "therealssh",
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
      "app": "therealssh-client",
      "version": "1.0",
      "auto_start": true,
      "port": 22
    }
  ]
```

Compile binaries and start 2 nodes:

```bash
$ go build -o apps/therealssh.v1.0 ./cmd/apps/therealssh
$ go build -o apps/therealssh-client.v1.0 ./cmd/apps/therealssh-client
$ go build ./cmd/therealssh-cli
$ ./skywire-node skywire1.json
$ ./skywire-node skywire2.json
```

Add public key of the second node to the auth file:

```bash
$ mkdir `/.therealssh
$ echo "0348c941c5015a05c455ff238af2e57fb8f914c399aab604e9abb5b32b91a4c1fe" > `/.therealssh/authorized_keys
```

Connect to the first node using CLI:

```bash
$ ./therealssh-cli 024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7
```

This should get you to the $HOME folder of the user(you in this case), which
will indicate that you are seeing remote PTY session.
