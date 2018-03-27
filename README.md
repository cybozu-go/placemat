# Placemat

Placemat is a provisioning tool to deploy QEMU VMs and configure networks for a
development environment.  A configuration of the VMs and networks is described
as declarative YAML.

Placemat's life-cycle is simple.  Placemat has no-daemon processes unlike
libvirt, or Docker.  The VMs and network configuration are constructed at the
beginning of the placemat's process, and they are destructed at the end of the
process with graceful shutdown.

## Usage

This project provides two commands, `placemat` and `placemat-connect`.
`placemat` command is a process to configure VMs, and `placemat-connect` is a
client tool to connect to QEMU's serial console.

```console
$ placemat [-nographic] [-run-dir=/tmp] network.yml nodes.yml other.yml
```

- `-nographic`: launch VMs without GUI console
- `-run-dir`: the directory to store sockets for VMs' consoles

You can define configuration for each `resources` to YAML files, or define them
to single files with a `---` separator.

With `-nographic` option, VMs launch without GUI console.  Their serial
consoles expose as pseudo terminals via a UNIX domain socket.
`placemat-connect` command can be used to connect them.

```console
$ placemat-connect [-run-dir=/tmp] your-vm-name
```

- `-run-dir`: the directory specified by `run-dir` of `placemat` command

## Getting started

### Prerequisites

Install following packages:

- [QEMU](https://www.qemu.org/)
- [OVMF](https://github.com/tianocore/tianocore.github.io/wiki/OVMF) (if UEFI boot is enabled)

For Ubuntu or Debian, install them by apt package manager:

```console
$ sudo apt-get update
$ sudo apt-get install qemu-system-x86 qemu-utils ovmf
```


### Install placemat

Install `placemat` and `placemat-connect`:

```console
$ go get -u github.com/cybozu-go/placemat/cmd/placemat
$ go get -u github.com/cybozu-go/placemat/cmd/placemat-connect
```

### Create a configuration and run it

Placemat constructs VMs and networks by declarative  YAML configuration.  Create
a following simple configuration into `cluster.yml`.  This configuration
includes a VM node and a network bridge.

```yaml
# cluster.yaml
kind: Network
name: net0
spec:
  addresses:
    - 172.16.0.1/24
---
kind: Node
name: debian
spec:
  interfaces:
    - net0
  volumes:
    - name: debian
      source: https://cdimage.debian.org/cdimage/openstack/9.4.0/debian-9.4.0-openstack-amd64.qcow2
```

To launch placemat from YAML files by the following:

```console
$ sudo placemat -nographic cluster.yml
```

Where `sudo` is required to create network bridge to your host.

Then you can connect to a console of the VM by the following:

```console
$ sudo placemat-connect debian
```

## Configures resources

The VMs and networks of placemat are described in YAML as *resources*.  There
are three type of resources, Network resource, Node resource, and NodeSet
resource.

### Network resource

Placemat creates a bridge network to local host machine by a Network resource.

```yaml
kind: Network
name: my-net
spec:
  addresses:
      - 10.0.0.0/22
```

The property in the `spec` is the following:

- `addresses`: IP addresses to assign bridge which accessed from host

The bridge network works as a virtual L2 network.  It connects each VMs and
exposes it to the host machine as an interface.  Placemat also set-up SNAT to
your host with `iptables`/`ip6tables` to reach to the internet.

### Node resource

Placemat creates a QEMU node by a Node resource.

```yaml
kind: Node
name: my-node
spec:
  interfaces:
    - net0
  volumes:
    - name: ubuntu
      source: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
      recreatePolicy: IfNotPresent
    - name: seed
      cloud-config:
        user-data: user-data.yml
        network-config: network.yml
      recreatePolicy: Always
    - name: data
      size: 10GB
      recreatePolicy: Never
  resources:
    cpu: 2
    memory: 4G
  bios: legacy
```

The properties in the `spec` are the following:

- `interfaces`: The network interfaces to connect Network resource(s).  They are specified by name of the Network resource.
- `volumes`: The volumes to mount to the VM.  The supported volumes are three types:
  - `size`:  Create a new disk by disk `size`.
  - `cloud-config`:  Generate a disk for cloud-init to utilize [nocloud](http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html), which allows the user to provide user-data and meta-data to the instance without running a network service.  `cloud-config` has two properties, `user-data` and `network-config`.
  - `source`:  Create a disk from URL via HTTP.
- `resources`:  `cpu` and `memory` resources to allocate to the VM.
- `bios`: BIOS mode of the VM.  If `uefi` is specified, the VM loads OVMF as BIOS.

Placemat launch a `qemu-system-x86_64` process by a Node resource.  If `size`
is specified in `volumes`, the volume is initialized by `qemu-img` command.  if
`cloud-config` is specified, the image is created by `cloud-localds`.

### NodeSet resource

Placemat creates multiple QEMU nodes by a NodeSet resource.

```yaml
kind: NodeSet
name: worker
spec:
  replicas: 3
  template:
    interfaces:
      - net0
    volumes:
      - name: system
        size: 100GB
```

The properties in the `spec` are the following:

- `replicas`: The number of the replicated nodes.
- `template`: The template of the `spec` in Node resource.

The actual name of the node is `name` of the resource with suffix `-N` (where `N` is a unique number).
The above example creates nodes named `worker-0`, `worker-1` and `worker-2`.

## Examples

See [examples](examples).

## License

MIT
