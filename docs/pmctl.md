pmctl
=====

pmctl is a command-line tool to control nodes, pods, and networks on placemat

Usage
-----

```console
$ pmctl [--endpoint http://localhost:10808] <subcommand> [args...]
```

| Option       | Default Value            | Description                           |
| ------------ | ------------------------ | ------------------------------------- |
| `--endpoint` | `http://localhost:10808` | `API endpoint of the target placemat` |


`node` subcommand
-----------------

### `pmctl node list [--json]`

Show nodes list.

* `--json`: Show detailed information of a node in JSON format.

```console
$ pmctl node list
node1
node2
```

```console
$ pmctl node list --json | jq .
[
  {
    "name": "node1",
    "taps": {
      "ext-net": "pm0"
    },
    "volumes": [
      "root",
      "data"
    ],
    "cpu": 1,
    "memory": "3G",
    "uefi": false,
    "smbios": {
      "manufacturer": "",
      "product": "",
      "serial": "e5e2a9518607915ae99ab77d575bfe7a7dcf2a99"
    },
    "is_running": true,
    "socket_path": "/tmp/host1.socket"
  },
  {
    "name": "node2",
    "taps": {
      "ext-net": "pm1"
    },
    "volumes": [
      "root",
      "data"
    ],
    "cpu": 1,
    "memory": "3G",
    "uefi": false,
    "smbios": {
      "manufacturer": "",
      "product": "",
      "serial": "e0d87a849f5a1e3d140e8b666446536edfe92089"
    },
    "is_running": true,
    "socket_path": "/tmp/host2.socket"
  }
]
```

### `pmctl node enter <NODE>`

Connect to a node via serial console.

If placemat starts without `-graphic` option, VMs will have no graphic console.
Instead, they have serial consoles exposed via UNIX domain sockets.

```console
$ pmctl node enter node1
```

**To exit** from the console, press Ctrl-Q, Ctrl-X in this order.

### `pmctl node action start <NODE>`

Start a node.

```console
$ pmctl node action start node1
```

### `pmctl node action stop <NODE>`

Stop a node.

```console
$ pmctl node action stop node1
```

### `pmctl node action restart <NODE>`

Restart a node.

```console
$ pmctl node action restart node1
```

`pod` subcommand
----------------

### `pmctl pod list [--json]`

Show pods list.

* `--json`: Show detailed information of a pod in JSON format.


### `pmctl pod enter [--app=<APP>] [COMMANDS...]`


`net` subcommand
----------------

### `pmctl net up <DEVICE>`

### `pmctl net down <DEVICE>`

### `pmctl net delay [--delay=<DELAY>] <DEVICE>`

### `pmctl net loss [--loss=<LOSS>] <DEVICE>`

### `pmctl net clear <DEVICE>`

