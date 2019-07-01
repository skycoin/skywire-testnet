# Skywire Chat app

Chat implements basic text messaging between visors.

Messaging UI is exposed via web interface.

Chat only supports one WEB client user at a time.

## Local setup

Create 2 visor config files:

`visor1.json`

```json
{
  "apps": [
    {
      "app": "skychat",
      "version": "1.0",
      "auto_start": true,
      "port": 1
    }
  ]
}
```

`visor2.json`

```json
{
  "apps": [
    {
      "app": "skychat",
      "version": "1.0",
      "auto_start": true,
      "port": 1,
      "args": ["-addr", ":8001"]
    }
  ]
}
```

Compile binaries and start 2 visors:

```bash
$ go build -o apps/skychat.v1.0 ./cmd/apps/skychat
$ ./visor visor1.json
$ ./visor visor2.json
```

Chat interface will be available on ports `8000` and `8001`.
