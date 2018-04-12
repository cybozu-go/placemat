Resource Specification
======================

The VMs and networks of placemat are described in YAML as *resources*.
Following resources are available.

* Network
* Image
* Node
* NodeSet

Network resource
----------------

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

Image resource
--------------

```yaml
kind: Image
name: ubuntu-cloud-image
spec:
   url: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
```

- `url`: downloads an image file from specified url
- `file`: a local file path
- `compression`: optional field to specify decompress method.  Currently, "gzip" and "bzip2" are supported.

DataFolder resource
-------------------

A DataFolder resource represents a host's directory or a set of remote/local files to be shown to guest VMs.
This resource can be referred from the `vvfat` type of volumes in Node resources.

VVFAT is a virtual device of QEMU that exports a directory of host OS to guests as a block device having a VFAT partition.
The block device need to be mounted as a read-only VFAT filesystem in guests.

Placemat can show files in a host's directory to guests via `vvfat` volumes.
It can also show files on the Internet by downloading them into a temporary directory first, and/or scattered files in a host by copying them first.

```yaml
kind: DataFolder
name: host-dir
spec:
  dir: /home/john/exported_dir
```

```yaml
kind: DataFolder
name: gathered-files
spec:
  files:
    - name: ubuntu.img
      url: https://example.com/docker_images/ubuntu_18.04
    - name: copied_readme.txt
      file: /home/john/README.txt
```

The properties in the `spec` are the following:

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
spec:
  interfaces:
    - net0
  volumes:
    - kind: image
      name: root
      spec:
        image: image-name
      recreatePolicy: IfNotPresent
    - kind: localds
      name: seed
      spec:
        user-data: user-data.yml
        network-config: network.yml
      recreatePolicy: Always
    - kind: raw
      name: data
      spec:
        size: 10GB
      recreatePolicy: Never
    - kind: vvfat
      name: host-data
      spec:
        folder: host-dir
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
- `volumes`: Volumes attached to the VM.  These kind of volumes are supported:
    - `image`: Image resource for QEMU disk image.
    - `localds`: [cloud-config](http://cloudinit.readthedocs.io/en/latest/topics/format.html#cloud-config-data) data.
    - `raw`: Raw (and empty) block device.
    - `vvfat`: DataFolder resource for QEMU VVFAT volume.
- `ignition`: [Ignition file](https://coreos.com/ignition/docs/latest/configuration-v2_1.html).
- `resources`:  `cpu` and `memory` resources to allocate to the VM.
- `smbios`: System Management BIOS (SMBIOS) values for `manufacturer`, `product`, and `serial`.  If `serial` is not set, a hash value of the node's name is used.
- `bios`: BIOS mode of the VM.  If `uefi` is specified, the VM loads OVMF as BIOS.

### `image` volume

Attaches `Image` resource as a VM disk.
This volume type has the following parameter:

* `image`: `Image` resource name.  Required.

### `localds` volume

Attaches a QEMU disk image created by [cloud-localds](https://manpages.debian.org/testing/cloud-image-utils/cloud-localds.1.en.html) with [cloud-config](http://cloudinit.readthedocs.io/en/latest/topics/format.html#cloud-config-data) data files.
This volume type has the following parameters:

* `user-data`: [Cloud Config Data](http://cloudinit.readthedocs.io/en/latest/topics/format.html#cloud-config-data) YAML file.  Required.
* `network-config`: [Network Configuration](http://cloudinit.readthedocs.io/en/latest/topics/network-config.html) YAML file.

### `raw` volume

Attaches a RAW, empty block device.
This volume type has the following parameter:

* `size`: Disk size.  Required.

### `vvfat` volume

Attaches a QEMU [VVFAT](https://en.wikibooks.org/wiki/QEMU/Devices/Storage) volume.
This volume type has the following parameter:

* `folder`: `DataFolder` resource name.  Required.

This volume type ignores `recreatePolicy` parameter.

From the guest OS, this volume appears as a block device containing a VFAT partition.
The partition need to be mounted read-only as follows:

```console
$ sudo mount -o ro /dev/vdb1 /mnt
```

NodeSet resource
----------------

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

