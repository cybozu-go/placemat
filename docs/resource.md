Resource Specification
======================

The VMs and networks of placemat are described in YAML as *resources*.
Following resources are available.

* Network
* Image
* DataFolder
* Node
* Pod

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

DataFolder resource
-------------------

A DataFolder resource provides a virtual data folder to VMs and containers.
The folder can simply be a host directory, or it can be a set of files from Internet or local host.

DataFolder can be referenced from Node resources as a `vvfat` volume type.
DataFolder can also be referenced from Pod resources as a `host` volume type.

```yaml
kind: DataFolder
name: host-dir
dir: /home/john/exported_dir
```

```yaml
kind: DataFolder
name: gathered-files
files:
  - name: ubuntu.img
    url: https://example.com/docker_images/ubuntu_18.04
  - name: copied_readme.txt
    file: /home/john/README.txt
```

The properties are:

- `dir`: Local directory name to be shown.
- `files`: List of file specs.
  - `name`: File name in the exported directory.
  - `url`: URL of a remote file to be downloaded.
  - `file`: Path to local file on host.

You must specify only one of `dir` or `files` for a DataFolder resource.
You must specify only one of `url` or `file` for each file in `files`.

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
  - kind: lv
    name: data2
    size: 10G
    vg: vg1
    cache: writeback
  - kind: vvfat
    name: host-data
    folder: host-dir
ignition: my-node.ign
cpu: 2
memory: 4G
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
    - `lv`: Raw (and empty) block device backed by a logical volume in LVM.
    - `vvfat`: DataFolder resource for QEMU VVFAT volume.
- `ignition`: [Ignition file](https://coreos.com/ignition/docs/latest/configuration-v2_1.html).
- `cpu`: The amount of virtual CPUs.
- `memory`: The amount of memory.
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

### `lv` volume

Attaches a RAW, empty block device backed by a logical volume in LVM.
This volume type has the following parameter:

* `size`: Disk size.  Required.
* `vg`: Volume group.  Required.

### `vvfat` volume

Attaches a QEMU [VVFAT](https://en.wikibooks.org/wiki/QEMU/Devices/Storage) volume.
This volume type has the following parameter:

* `folder`: `DataFolder` resource name.  Required.

From the guest OS, this volume appears as a block device containing a VFAT partition.
The partition need to be mounted read-only as follows:

```console
$ sudo mount -o ro /dev/vdb1 /mnt
```

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
    args:
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
