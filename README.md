[![GitHub release](https://img.shields.io/github/release/cybozu-go/placemat.svg?maxAge=60)][releases]
[![CI](https://github.com/cybozu-go/placemat/actions/workflows/ci.yaml/badge.svg)](https://github.com/cybozu-go/placemat/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/cybozu-go/placemat/v2.svg)](https://pkg.go.dev/github.com/cybozu-go/placemat/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/placemat)](https://goreportcard.com/report/github.com/cybozu-go/placemat)

Placemat
========

Placemat is a tool to simulate data center networks and servers using
QEMU/KVM virtual machines, and Linux networking stacks.  Placemat can simulate
virtually *any* kind of network topologies to help tests and experiments for software
usually used in data centers.

Features
--------

* No daemons

    Placemat is a single binary executable.  It just builds networks and
    virtual machines when it starts, and destroys them when it terminates.
    This simplicity makes placemat great for a continuous testing tool.

* Declarative YAML

    Networks, virtual machines, and other kind of resources are defined
    in YAML files in a declarative fashion.  Users need not mind the order
    of creation and/or destruction of resources.

* Virtual BMC

    Power on/off/reset of VMs can be done by [IPMI][] commands and [Redfish][] API.
    See [virtual BMC](docs/virtual_bmc.md) for details.

* Automation

    Placemat supports [cloud-init][] and [ignition][] to automate
    virtual machine initialization.  Files on the host machine can be
    exported to guests as a [9pfs](https://wiki.qemu.org/Documentation/9psetup).
    QEMU disk images can be downloaded from remote HTTP servers.

    All of these help implementation of fully-automated tests.

* UEFI

    Not only traditional BIOS, but placemat VMs can be booted in UEFI
    mode if [OVMF][] is available.

Usage
-----

This project provides these commands:

* `placemat2` is the main tool to build networks and virtual machines.
* `pmctl2` is a utility tool to control VMs and Pods.

### placemat2 command

`placemat2` reads all YAML files specified in command-line arguments,
then creates resources defined in YAML.  To destroy, just kill the
process (by sending a signal or Control-C).

```console
$ placemat2 [OPTIONS] YAML [YAML ...]

Options:
  --cache-dir string
        directory for cache data
  --data-dir string
        directory to store data (default "/var/scratch/placemat")
  --debug
        show QEMU's stdout and stderr
  --force
        force run with removal of garbage
  --graphic
        run QEMU with graphical console
  --listen-addr string
        listen address (default "127.0.0.1:10808")
  --run-dir string
        run directory (default "/tmp")
```

If `--cache-dir` is not specified, the default will be `/home/${SUDO_USER}/placemat_data`
if `sudo` is used for `placemat`.  If `sudo` is not used, cache directory will be
the same as `--data-dir`.
`--force` is used for forced run. Remaining garbage, for example virtual networks, mounts, socket files will be removed.

### pmctl2 command

`pmctl2` is a command line tool to control VMs and Networks.

See [pmctl](docs/pmctl.md)

Getting started
---------------

### Prerequisites

- [QEMU][]
- [OVMF][] for UEFI.
- [picocom](https://github.com/npat-efault/picocom) for `pmctl2`.
- [socat](http://www.dest-unreach.org/socat/) for `pmctl2`.
- *(Optional)* [swtpm](https://github.com/stefanberger/swtpm) for providing TPM of `Node` resource.

For Ubuntu or Debian, you can install them as follows:

```console
$ sudo apt-get update
$ sudo apt-get install qemu-system-x86 qemu-utils ovmf picocom socat cloud-utils
```

### Install placemat

You can choose `go get` or debian package for installation.

Install `placemat2` and `pmctl2`:

```console
$ go install github.com/cybozu-go/placemat/v2/cmd/placemat2@latest
$ go install github.com/cybozu-go/placemat/v2/cmd/pmctl2@latest
```

or

```console
$ wget https://github.com/cybozu-go/placemat/releases/download/v${VERSION}/placemat2_${VERSION}_amd64.deb
$ sudo dpkg -i placemat2_${VERSION}_amd64.deb
```

### Run examples

See [examples](examples) how to write YAML files.

To launch placemat from YAML files, run it with `sudo` as follows:

```console
$ sudo $GOPATH/bin/placemat2 cluster.yml
```

To connect to a serial console of a VM, use `pmctl2 node enter`:

```console
$ sudo $GOPATH/bin/pmctl2 node enter VM
```

This will launch `picocom`.  To exit, type `Ctrl-Q`, then `Ctrl-X`.

Specification
-------------

See specifications under [docs directory](docs/).

License
-------

placemat is licensed under the Apache License, Version 2.0.

[releases]: https://github.com/cybozu-go/placemat/releases
[godoc]: https://godoc.org/github.com/cybozu-go/placemat
[cloud-init]: http://cloudinit.readthedocs.io/en/latest/index.html
[ignition]: https://coreos.com/ignition/docs/latest/
[QEMU]: https://www.qemu.org/
[OVMF]: https://github.com/tianocore/tianocore.github.io/wiki/OVMF
[IPMI]: https://en.wikipedia.org/wiki/Intelligent_Platform_Management_Interface
[Redfish]: https://www.dmtf.org/standards/redfish
