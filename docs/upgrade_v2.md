Upgrade V2
==========

- [Overview](#overview)
- [Supported OS](#supported-os)
- [New Features](#new-features)
  - [New Resources](#new-resources)
  - [Virtual BMC](#virtual-bmc)
  - [Others](#others)
- [Incompatible Changes](#incompatible-changes)
  - [Obsolete Resources](#obsolete-resources)
  - [Command Line Programs](#command-line-programs)
  - [Deb Package](#deb-package)

## Overview

Placemat version2 incorporates new features and improvements based on the knowledge we have accumulated over the years.
The implementation and the structure of the source files have been significantly revamped. It includes some incompatible changes as well. Please check the following.

## Supported OS

Support Ubuntu 18.04 and later releases.

## New Features

### New Resources

- [NetworkNamespace resource](resource.md#networknamespace) that creates a separated network stack.
- [`hostPath`](resource.md#node-resource) type volume of Node resource that creates virtio-9p-device and exposes them to guests.

### Virtual BMC

- [IPMI v2.0](virtual_bmc.md#ipmi) support.
- [Redfish API](virtual_bmc.md#redfish-api) support.

### Others

- [Auto Tune MTU value](design.md#auto-tune-mtu-value)

## Incompatible Changes

### Obsolete Resources

- Pod resource. Placemat is no longer depends on `rkt` or any other container engines. You can use NetworkNamespace resource that creates a separated network stack and run commands inside it as an alternative.
- DataFolder resource.
- `vvfat` `lv` type volume from Node resource. For `vvfat`, you can use `hostPath` type volume to expose host directories to guests.

### Command line programs

- pmctl
    - Removed `pod` `net` `snapshot` subcommands.
- placemat
    - placemat now requires double hyphen to specify an option, for example `--force`.
    - Removed `-bmc-cert` `-bmc-key` `-enable-virtfs` options, For `-enable-virtfs`, you can use `hostPath` type volume as an alternative.

### Deb package

- The deb package is now `placemat2`. The programs the deb package contains are `placemat2` and `pmctl2`.

## Internal Changes

If you are only interested in external specifications, please skip this section.

### Source files structure

The structure of the source files has been revamped as follows.

- cmd
  - placemat2
    - placemat2 entry point.
  - pmctl2
    - pmctl2 entry point.
- pkg
  - dcnet
    - Network configuration components such as setting up bridge, network namespace and iptables.
  - placemat
    - Components that set up a cluster as specified using other packages' components. Also includes Placemat API server.
  - types
    - Yaml representation of resources.
  - util
    - Utilities.
  - virtualbmc
    - Virtual BMC components that start up IPMI and Redfish server.
  - vm
    - Virtual Machine configuration components using QEMU.

### Network configuration

- Use go libraries to configure networks instead of running ip command, making it easy to handle from Go program.
  - [vishvananda/netlink](https://github.com/vishvananda/netlink)
  - [vishvananda/netns](https://github.com/vishvananda/netns)
  - [containernetworking/plugins](https://github.com/containernetworking/plugins/tree/master/pkg)
  - [go-iptables/iptables](https://pkg.go.dev/github.com/coreos/go-iptables/iptables)

### Virtual BMC Implementation

- Removed dependency on an external IPMI Server library and implemented an embedded IPMI server that supports RMCP+ authentication.
- Use [gin-gonic/gin](https://github.com/gin-gonic/gin) HTTP web framework for Redfish server and Placemat API server.

### QEMU

- Use [QMP](https://wiki.qemu.org/Documentation/QMP), a machine-friendly JSON based protocol, to control QEMU instances instead of Monitor console.
