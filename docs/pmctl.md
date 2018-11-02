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
      "mynet": "pm0"
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
      "mynet": "pm1"
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
$ sudo pmctl node enter node1
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

```console
$ pmctl pod list --json | jq .
pod1
```

```console
$ pmctl pod list --json | jq .
[
  {
    "name": "pod1",
    "uuid": "03464bd4-eff7-408a-8bc2-1f218bd7b83f",
    "veths": {
      "mynet": "pm8"
    },
    "volumes": [],
    "apps": [
      "ubuntu"
    ]
  }
]
```

### `pmctl pod enter [--app=<APP>] [COMMANDS...]`

Enter the namespace of a pod.

```console
$ sudo pmctl pod enter pod1
root@pod1:/#
```

If the pod has multiple containers, you should specify `--app` option.

```console
$ sudo pmctl pod enter pod1 --app=ubuntu
root@pod1:/#
```

`COMMANDS` is specified, it will be executed in the pod.

```console
$ sudo pmctl pod enter pod1 uname -- -a
Linux pod1 4.15.0-38-generic #41-Ubuntu SMP Wed Oct 10 10:59:38 UTC 2018 x86_64 x86_64 x86_64 GNU/Linux
```

`net` subcommand
----------------

In this subcommand you need to specify network device name.

The name can be obtained with `pmclt node list` or `pmctl pod list` as following.

```console
$ DEVICE=$(pmctl node list --json | jq -r '.[] | select(.name=="node1").taps."mynet"')
$ echo $DEVICE
pm0
```

```console
$ DEVICE=$(pmctl pod list --json | jq -r '.[] | select(.name=="pod1").veths."mynet"')
$ echo $DEVICE
pm8
```

### `pmctl net action up <DEVICE>`

Change state of the device to UP.

```console
$ pmctl net action up $DEVICE
```

### `pmctl net action down <DEVICE>`

Change state of the device to DOWN.

```console
$ pmctl net action down $DEVICE
```
### `pmctl net action delay [--delay=<DELAY>] <DEVICE>`

Ddd delay to the packets going out of the device with `tc` command.

* `--delay`: Specify the delay time. (default: 100ms)

```console
$ pmctl net action delay --delay=1s $DEVICE
```

```console
node1 ~ $ ping 10.0.0.102
PING 10.0.0.102 (10.0.0.102) 56(84) bytes of data.
64 bytes from 10.0.0.102: icmp_seq=1 ttl=64 time=1000 ms
64 bytes from 10.0.0.102: icmp_seq=2 ttl=64 time=1000 ms
64 bytes from 10.0.0.102: icmp_seq=3 ttl=64 time=1001 ms
```

### `pmctl net action loss [--loss=<LOSS>] <DEVICE>`

Drop packets randomly going out of the device with `tc` command.

* `--loss`: Specify the percentage of loss. (default: 10%)

```console
$ pmctl net action loss --loss=80% $DEVICE
```

```console
node1 ~ $ ping 10.0.0.102
PING 10.0.0.102 (10.0.0.102) 56(84) bytes of data.
64 bytes from 10.0.0.102: icmp_seq=3 ttl=64 time=0.972 ms
64 bytes from 10.0.0.102: icmp_seq=8 ttl=64 time=1.09 ms
64 bytes from 10.0.0.102: icmp_seq=11 ttl=64 time=0.885 ms
```

### `pmctl net action clear <DEVICE>`

Clear the effect by "delay" and "loss" action.

```console
$ pmctl net action clear $DEVICE
```
