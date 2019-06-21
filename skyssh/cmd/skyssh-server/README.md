# Skywire skyssh app

`skyssh-server` app implements skyssh functionality over skywirenet.

`skyssh-cli` is used to initiate communication via client RPC
exposed by `skyssh` app. 

`skyssh` app implements common skyssh operations:

- starting remote shell
- and executing commands remotely

PubKey whitelisting is performed by adding public key to the
authentication file (`$HOME/.skyssh/authorized_keys` by default).

** Local setup

Create 2 node config files:

`skywire1.json`

```json
  "apps": [
    {
      "app": "skyssh-server",
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
      "app": "skyssh-client",
      "version": "1.0",
      "auto_start": true,
      "port": 22
    }
  ]
```

Compile binaries and start 2 nodes:

```bash
$ go build -o apps/skyssh-server.v1.0 ./skyssh/cmd/skyssh-server
$ go build -o apps/skyssh-client.v1.0 ./skyssh/cmd/skyssh-client
$ go build ./skyssh/cmd/skyssh-cli
$ ./skywire-node skywire1.json
$ ./skywire-node skywire2.json
```

Add public key of the second node to the auth file:

```bash
$ mkdir ~/.skyssh
$ echo "0348c941c5015a05c455ff238af2e57fb8f914c399aab604e9abb5b32b91a4c1fe" > ~/.skyssh/authorized_keys
```

Connect to the first node using CLI:

```bash
$ ./skyssh-cli 024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7
```

This should get you to the $HOME folder of the user(you in this case), which
will indicate that you are seeing remote PTY session.
