Resource Specification
======================

The VMs and networks of placemat are described in YAML as *resources*.
Following resources are available.

* Network
* Image
* Node
* NetworkNamespace
* DeviceClass

Network resource
----------------

Placemat creates a bridge network to local host machine by a Network resource.

```yaml
kind: Network
name: my-net
type: external
use-nat: true
address: 10.0.0.0/22
```

The properties are:

- `type`: `internal` or `external` or `bmc`
- `use-nat`: Whether or not this network requires NAT on host to reach the Internet.  `true` or `false`.
- `address`: IP address to be assigned to the bridge which can be accessed from host.

The bridge network works as a virtual L2 network.  It connects VMs to each other.
If `type` is `external`, the bridge is exposed to the host OS as an interface.
If `use-nat` is true, placemat configures SNAT for the packets from the bridge
with iptables/ip6tables.

Type `bmc` is special.  See [Virtual BMC](virtual_bmc.md) for details.

You need not (and cannot) specify `use-nat` or `address` if `type` is `internal`.
You must specify at least 1 address if `type` is not `internal`.

Image resource
--------------

```yaml
kind: Image
name: ubuntu-cloud-image
url: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
```

- `url`: downloads an image file from specified url
- `file`: a local file path
- `compression`: optional field to specify decompress method.  Currently, "gzip" and "bzip2" are supported.

Node resource
-------------

Placemat creates a QEMU process by a Node resource.

```yaml
kind: Node
name: my-node
interfaces:
  - net0
volumes:
  - kind: image
    name: root
    image: image-name
    copy-on-write: true
  - kind: localds
    name: seed
    user-data: user-data.yml
    network-config: network.yml
  - kind: raw
    name: data
    size: 10G
    device-class: ssd
  - kind: hostPath
    name: host-data
    path: /var/lib/foo
    writable: false
ignition: my-node.ign
cpu: 2
memory: 4G
network-device-queue: 4
smbios:
  manufacturer: cybozu
  product: mk2
  serial: 1234abcd
uefi: false
tpm: true
```

The properties are:

- `interfaces`: The network interfaces to connect Network resource(s).  They are specified by name of the Network resource.
- `volumes`: Volumes attached to the VM.  These kind of volumes are supported:
    - `image`: Image resource for QEMU disk image.
    - `localds`: [cloud-config](http://cloudinit.readthedocs.io/en/latest/topics/format.html#cloud-config-data) data.
    - `raw`: Raw (and empty) block device backed by a file.
    - `hostPath`: Shared directory of the host using QEMU 9pfs.
- `ignition`: [Ignition file](https://coreos.com/ignition/docs/latest/configuration-v2_1.html).
- `cpu`: The amount of virtual CPUs.
- `memory`: The amount of memory.
- `network-device-queue`: The count of VM's network device queue. Placemat enables multi queue virtio-net if network-device-queue is greater than 1.
- `smbios`: System Management BIOS (SMBIOS) values for `manufacturer`, `product`, and `serial`.  If `serial` is not set, a hash value of the node's name is used.
- `uefi`: BIOS mode of the VM.
    - If false: The VM will load Qemu's default BIOS (SeaBIO) and enable iPXE boot by a net device.
    - If true: The VM loads OVMF as BIOS and disable iPXE boot by a net device.
- `tpm`: Create Trusted Platform Module(TPM) for the VM. This feature requires [swtpm](https://github.com/stefanberger/swtpm).
    - If false: Provide no TPM device.
    - If true: Provide a TPM device as `/dev/tpm0` on the VM.

### common volume parameters
* `kind`: kind of the volume.  Required.
* `name`: name of the volume.  Required.
* `cache`: determine how to access backend storage. Possible values are `writeback`, `none`, `writethrough`, `directsync`, `unsafe`.  Defaulted to `none`.
* `device-class`: determine where to locate backend storage. Possible values are defined in `DeviceClass` resource. If this field isn't set, unnamed device class will be assined and default path will be used.

### `image` volume

Attaches `Image` resource as a VM disk.
This volume type has the following parameter:

* `image`: `Image` resource name.  Required.
* `copy-on-write`: if `true`, create a copy-on-write image based on the specified `Image` resource.
Only the modified data will be stored in the created image file.
if `false`, the file copied entirely from specified `Image` resource will be used.
default is `false`.

### `localds` volume

Attaches a QEMU disk image created by [cloud-localds](https://manpages.debian.org/testing/cloud-image-utils/cloud-localds.1.en.html) with [cloud-config](http://cloudinit.readthedocs.io/en/latest/topics/format.html#cloud-config-data) data files.
This volume type has the following parameters:

* `user-data`: [Cloud Config Data](http://cloudinit.readthedocs.io/en/latest/topics/format.html#cloud-config-data) YAML file.  Required.
* `network-config`: [Network Configuration](http://cloudinit.readthedocs.io/en/latest/topics/network-config.html) YAML file.

### `raw` volume

Attaches a RAW, empty block device backed by a file.
This volume type has the following parameters:

* `size`: Disk size.  Required.
* `format`: QEMU disk image format.  `qcow2` (default) or `raw`.

### `hostPath` volume

Attaches a QEMU [9p](https://wiki.qemu.org/Documentation/9psetup) volume.
This volume type has the following parameter:

* `path`: An absolute path of the host-side directory.  Required.
* `writable`: If `true`, then an attached volume is writable. If `false`, then it is readonly and that is the default.

You can mount the shared folder using

```console
$ sudo mount -t 9p -o trans=virtio MOUNT_TAG MOUNT_NAME -oversion=9p2000.L
```

`mount tag` is a volume name as specified.

NetworkNamespace
----------------

Placemat creates a network namespace by referencing a NetworkNamespace resource.

Placemat prepares the network stack that consists of the given interfaces. Each network stack has its dedicated routing tables, iptables rules, etc.

In the network namespace, IP-forwarding is enabled by default.

```yaml
kind: NetworkNamespace
name: my-netns
init-scripts:
  - /path/to/script
interfaces:
  - network: net0
    addresses:
      - 10.0.0.1/24
apps:
  - name: bird
    command:
    - /usr/local/bird/sbin/bird
    - -f
    - -c
    - /etc/bird/bird_core.conf
```

Properties are described in the following sub sections.

### init-scripts

These scripts will be executed to initialize environments before running each application inside the network namespace.

### interfaces

List of network interfaces assigned to the network namespace. Each interface will be attached to a Network resource specified by `network`, and have IP addresses listed in `addresses`.
Interfaces will be named `eth0`, `eth1`, ... in the order of definition.

### apps

List of applications running inside the network namespace.

DeviceClass resource
--------------------

Placemat creates backend storage at the location specified by this resource.

```yaml
kind: DeviceClass
name: ssd
path: /var/scratch/ssd
```

The properties are:

- `path`: The path to locate backend storage.
