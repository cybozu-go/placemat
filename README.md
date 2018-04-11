[![CircleCI](https://circleci.com/gh/cybozu-go/placemat.svg?style=svg)](https://circleci.com/gh/cybozu-go/placemat)
[![GoDoc](https://godoc.org/github.com/cybozu-go/placemat?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/placemat)](https://goreportcard.com/report/github.com/cybozu-go/placemat)

Placemat
========

Placemat is a tool to simulate data center networks and servers using QEMU/KVM
and Linux networking stacks.  Placemat can simulate virtually *any* kind of
network topologies to help tests and experiments for software usually used in
a data center.

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

* Automation

    Placemat supports [cloud-init][] and [ignition][] to automate
    virtual machine initialization.  Files on the host machine can be
    exported to guests as a [VVFAT drive](https://en.wikibooks.org/wiki/QEMU/Devices/Storage).
    QEMU disk images can be downloaded from remote HTTP servers.

    All of these help implementation of fully-automated tests.

* UEFI

    Not only traditional BIOS, but placemat VMs can be booted in UEFI
    mode if [OVMF][] is installed.

Usage
-----

This project provides two commands, `placemat` and `placemat-connect`.
`placemat` is the main tool to build networks and virtual machines.
`placemat-connect` is a helper to connect to a serial console of
a VM launched by `placemat`.

### placemat command

`placemat` reads all YAML files specified in command-line arguments,
then creates resources defined in YAML.  To destroy, just kill the
process (by sending a signal or Control-C).

```console
$ placemat [OPTIONS] YAML [YAML ...]

Options:
  -nographic
        run QEMU with no graphic
  -run-dir string
        run directory (default "/tmp")
  -data-dir string
        directory to store data (default "$HOME/placemat_data")
```

### placemat-connect command

If placemat starts with `-nographic` option, VMs will have no graphic console.
Instead, they have serial consoles exposed via UNIX domain sockets.

`placemat-connect` is a tool to connect to the serial console.

```console
$ placemat-connect [-run-dir=/tmp] your-vm-name

Options:
  -run-dir
        the directory specified for placemat by -run-dir.
```

**To exit** from the console, press Ctrl-Q, Ctrl-X in this order.

Getting started
---------------

### Prerequisites

- [QEMU][]
- [OVMF][] (if UEFI boot is enabled)
- [picocom](https://github.com/npat-efault/picocom) for `placemat-connect`

For Ubuntu or Debian, you can install them as follows:

```console
$ sudo apt-get update
$ sudo apt-get install qemu-system-x86 qemu-utils ovmf picocom
```

### Install placemat

Install `placemat` and `placemat-connect`:

```console
$ go get -u github.com/cybozu-go/placemat/cmd/placemat
$ go get -u github.com/cybozu-go/placemat/cmd/placemat-connect
```

### Run examples

See [examples](examples) how to write YAML files.

To launch placemat from YAML files by the following:

```console
$ sudo $GOPATH/bin/placemat -nographic cluster.yml
```

Where `sudo` is required to create network bridge to your host.

You can connect to a serial console of a VM as follows:

```console
$ sudo $GOPATH/bin/placemat-connect VM
```

Specification
-------------

See [SPEC](SPEC.md).

License
-------

MIT

[godoc]: https://godoc.org/github.com/cybozu-go/aptutil
[cloud-init]: http://cloudinit.readthedocs.io/en/latest/index.html
[ignition]: https://coreos.com/ignition/docs/latest/
[QEMU]: https://www.qemu.org/
[OVMF]: https://github.com/tianocore/tianocore.github.io/wiki/OVMF
