pmctl2
======

pmctl2 is a command-line tool to control nodes, network namespaces, and networks on placemat

Usage
-----

```console
$ pmctl2 [--endpoint http://localhost:10808] <subcommand> [args...]
```

| Option       | Default Value            | Description                           |
| ------------ | ------------------------ | ------------------------------------- |
| `--endpoint` | `http://localhost:10808` | `API endpoint of the target placemat` |


`node` subcommand
-----------------

### `pmctl2 node list [--json]`

Show nodes list.

* `--json`: Show detailed information of a node in JSON format.

```console
$ pmctl2 node list
node1
node2
```

```console
$ pmctl2 node list --json | jq .
[
  {
    "name": "node1",
    "taps": {
      "mynet": "pm0"
    },
    "volumes": [
      "root",
      "data"
    ],
    "cpu": 1,
    "memory": "3G",
    "uefi": false,
    "tpm": false,
    "smbios": {
      "manufacturer": "",
      "product": "",
      "serial": "e5e2a9518607915ae99ab77d575bfe7a7dcf2a99"
    },
    "power_status": "On",
    "socket_path": "/tmp/host1.socket"
  },
  {
    "name": "node2",
    "taps": {
      "mynet": "pm1"
    },
    "volumes": [
      "root",
      "data"
    ],
    "cpu": 1,
    "memory": "3G",
    "uefi": false,
    "tpm": false,
    "smbios": {
      "manufacturer": "",
      "product": "",
      "serial": "e0d87a849f5a1e3d140e8b666446536edfe92089"
    },
    "power_status": "On",
    "socket_path": "/tmp/host2.socket"
  }
]
```

### `pmctl2 node show <NODE>`

Show a node info.

```console
$ pmctl2 node node1 | jq .
{
  "name": "node1",
  "taps": {
    "mynet": "pm0"
  },
  "volumes": [
    "root",
    "data"
  ],
  "cpu": 1,
  "memory": "3G",
  "uefi": false,
  "tpm": false,
  "smbios": {
    "manufacturer": "",
    "product": "",
    "serial": "e5e2a9518607915ae99ab77d575bfe7a7dcf2a99"
  },
  "power_status": "On",
  "socket_path": "/tmp/host1.socket"
}
```

### `pmctl2 node enter <NODE>`

Connect to a node via serial console.

If placemat starts without `-graphic` option, VMs will have no graphic console.
Instead, they have serial consoles exposed via UNIX domain sockets.

```console
$ sudo pmctl2 node enter node1
picocom v2.2

port is        : /tmp/placemat_node1
flowcontrol    : none
baudrate is    : 9600
parity is      : none
databits are   : 8
stopbits are   : 1
escape is      : C-q
local echo is  : no
noinit is      : no
noreset is     : no
nolock is      : no
send_cmd is    : sz -vv
receive_cmd is : rz -vv -E
imap is        :
omap is        :
emap is        : crcrlf,delbs,

Type [C-q] [C-h] to see available commands

Terminal ready

This is node1 (Linux x86_64 4.14.55-coreos) 03:09:18

node1 ~ $
```

**To exit** from the console, press Ctrl-Q, Ctrl-X in this order.

### `pmctl2 node action start <NODE>`

Start a node.

```console
$ pmctl2 node action start node1
```

### `pmctl2 node action stop <NODE>`

Stop a node.

```console
$ pmctl2 node action stop node1
```

### `pmctl2 node action restart <NODE>`

Restart a node.

```console
$ pmctl2 node action restart node1
```

`forward` subcommand
--------------------

`forward` subcommand manages port-forward settings from the host to internal networks.

### `pmctl2 forward list [--json]`

Show list of forward settings.

* `--json`: Show detailed information of forward settings in JSON format.

```console
$ pmctl2 forward list
30000 external:10.72.32.0:80
30001 external:10.72.32.1:80
```

```console
$ pmctl2 forward list --json
[
  {
    "local_port": 30000,
    "netns": "external",
    "remote_host": "10.72.32.0",
    "remote_port": 80
  },
  {
    "local_port": 30001,
    "netns": "external",
    "remote_host": "10.72.32.1",
    "remote_port": 80
  }
]
```

### `pmctl2 forward add <LOCAL PORT> <NETWORK NS>:<REMOTE HOST>:<REMOTE PORT>`

Add a forward setting.

This listens on `0.0.0.0:<LOCAL PORT>` in TCP, and forwards connections to `<REMOTE HOST>:<REMOTE PORT>` in the network namespace.

```console
$ pmctl2 forward add 30000 external:10.72.32.0:80
```

### `pmctl2 forward delete <LOCAL PORT>`

Delete a forward setting listening on `<LOCAL PORT>`.

```console
$ pmctl2 forward delete 30000
```

`completion` subcommand
-----------------------

Generates shell completion functions.

Usage:

```console
$ complete -r pmctl2
$ . <(pmctl2 completion  [bash|zsh|fish|powershell])
```
