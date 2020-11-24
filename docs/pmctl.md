pmctl
=====

pmctl is a command-line tool to control nodes, network namespaces, and networks on placemat

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

### `pmctl node show <NODE>`

Show a node info.

```console
$ pmctl node node1 | jq .
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
}
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

`net` subcommand
----------------

In this subcommand you need to specify network device name.

The name can be obtained with `pmctl node show` or `ip netns exec ip link` as following.

```console
$ DEVICE=$(pmctl node show node1 | jq -r '.taps."mynet"')
$ echo $DEVICE
pm0
```

```console
# pm20 is the peer of the device eth0 inside the core network namespace, and it has been added to the bridge internet.
$ ip netns exec core ip link
51: eth0@if52: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default qlen 1000
    link/ether f6:55:f8:08:d4:61 brd ff:ff:ff:ff:ff:ff link-netnsid 0

$ ip link
52: pm20@if51: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master internet state UP mode DEFAULT group default qlen 1000
    link/ether 9e:42:e4:3b:bb:34 brd ff:ff:ff:ff:ff:ff link-netnsid 10
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

`forward` subcommand
--------------------

`forward` subcommand manages port-forward settings from the host to internal networks.

### `pmctl forward list [--json]`

Show list of forward settings.

* `--json`: Show detailed information of forward settings in JSON format.

```console
$ pmctl forward list
30000 external:10.72.32.0:80
30001 external:10.72.32.1:80
```

```console
$ pmctl forward list --json
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

### `pmctl forward add <LOCAL PORT> <NETWORK NS>:<REMOTE HOST>:<REMOTE PORT>`

Add a forward setting.

This listens on `0.0.0.0:<LOCAL PORT>` in TCP, and forwards connections to `<REMOTE HOST>:<REMOTE PORT>` in the network namespace.

```console
$ pmctl forward add 30000 external:10.72.32.0:80
```

### `pmctl forward delete <LOCAL PORT>`

Delete a forward setting listening on `<LOCAL PORT>`.

```console
$ pmctl forward delete 30000
```

`completion` subcommand
-----------------------

Generates bash completion functions.

Usage:

```console
$ complete -r pmctl
$ . <(pmctl completion)
```
