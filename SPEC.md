Specification
=============

Resources
---------

The VMs and networks of placemat are described in YAML as *resources*.
Following resources are available.

* Network
* Image
* Node
* NodeSet

### Network resource

Placemat creates a bridge network to local host machine by a Network resource.

```yaml
kind: Network
name: my-net
spec:
  internal: false
  use-nat: true
  addresses:
      - 10.0.0.0/22
```

The properties in the `spec` are the following:

- `internal`: Whether or not this network should be configured as an internal switch.  `true` or `false`.
- `use-nat`: Whether or not this network requires NAT on host to reach the Internet.  `true` or `false`.
- `addresses`: IP addresses to be assigned to the bridge which can be accessed from host.

The bridge network works as a virtual L2 network.  It connects VMs to each other.
If `internal` is false, the bridge is exposed to the host OS as an interface.
If `use-nat` is true, placemat configures SNAT for the packets from the bridge
with iptables/ip6tables.

You need not (and cannot) specify `use-nat` or `addresses` if `internal` is true.
You must specify at least 1 address if `internal` is false.

### Image resource

```yaml
kind: Image
name: ubuntu-cloud-image
spec:
   url: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
```

- `url`: downloads an image file from specified url
- `file`: a local file path
- `compression`: optional field to specify decompress method.  Currently, "gzip" and "bzip2" are supported.
- `copy-on-write`: create a copy-on-write image based on downloaded file. default is false.

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
      source: ubuntu-cloud-image
      recreatePolicy: IfNotPresent
    - name: seed
      cloud-config:
        user-data: user-data.yml
        network-config: network.yml
      recreatePolicy: Always
    - name: data
      size: 10GB
      recreatePolicy: Never
  ignition: my-node.ign
  resources:
    cpu: 2
    memory: 4G
  smbios:
    manufacturer: cybozu
    product: mk2
    serial: 1234abcd
  bios: legacy
```

The properties in the `spec` are the following:

- `interfaces`: The network interfaces to connect Network resource(s).  They are specified by name of the Network resource.
- `volumes`: The volumes to mount to the VM.  The supported volumes are three types:
  - `size`:  Create a new disk by disk `size`.
  - `cloud-config`:  Generate a disk for cloud-init to utilize [nocloud](http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html), which allows the user to provide user-data and meta-data to the instance without running a network service.  `cloud-config` has two properties, `user-data` and `network-config`.
  - `source`:  Name of an image resource.
- `ignition`: [Ignition file](https://coreos.com/ignition/docs/latest/configuration-v2_1.html).
- `resources`:  `cpu` and `memory` resources to allocate to the VM.
- `smbios`: System Management BIOS (SMBIOS) values for `manufacturer`, `product`, and `serial`.  If `serial` is not set, a hash value of the node's name is used.
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

