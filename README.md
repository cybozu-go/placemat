Placemat
========

Placemat is a provisioning tool to deploy QEMU VMs and configure networks for a
development environment.  A configuration of the VMs and networks is described
as declarative YAML.

Placemat's life-cycle is simple.  Placemat has no-daemon processes unlike
libvirt, or Docker.  The VMs and network configuration are constructed at the
beginning of the placemat's process, and they are destructed at the end of the
process with graceful shutdown.

Usage
-----

This project provides two commands, `placemat` and `placemat-connect`.
`placemat` command is a process to configure VMs, and `placemat-connect` is a
client tool to connect to QEMU's serial console.

### placemat command

```console
$ placemat [OPTIONS] network.yml nodes.yml other.yml

Options:
  -nographic
        run QEMU with no graphic
  -run-dir string
        run directory (default "/tmp")
  -data-dir string
        directory to store data (default "$HOME/placemat_data")
```

You can define configuration for each `resources` to YAML files, or define them
to single files with a `---` separator.

### placemat-connect command

If placemat starts with `-nographic` option, VMs will launch without GUI console.
Their serial consoles expose as pseudo terminals via a UNIX domain socket.

`placemat-connect` command can be used to connect them.

```console
$ placemat-connect [-run-dir=/tmp] your-vm-name
```

- `-run-dir`: the directory specified by `run-dir` of `placemat` command

Getting started
---------------

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

### Run examples

See [examples](examples) how to write YAML files.

To launch placemat from YAML files by the following:

```console
$ sudo placemat -nographic cluster.yml
```

Where `sudo` is required to create network bridge to your host.

Then you can connect to a console of the VM by the following:

```console
$ sudo placemat-connect debian
```

Specification
-------------

See [SPEC](SPEC.md).

License
-------

MIT
