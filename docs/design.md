V2 Design Note
==============

- [Overview](#overview)
  - [Background](#background)
  - [Goals](#goals)
- [NetworkNamespace Resource](#networknamespace-resource)
- [Running Placemat2 inside a kubernetes Pod](#running-placemat2-inside-a-kubernetes-pod)
- [New Features](#new-features)
  - [New Package Placemat2](#new-package-placemat2)
  - [Redfish API Support](#redfish-api-support)
- [Obsolete Features](#obsolete-features)

## Overview

Placemat is a data center network and server simulation tool using QEMU/KVM virtual machines. This document elaborates the motivation and goals of Placemat v2.

### Background

Currently, the integration test suites of Neco, our data center management software, are running on a virtual data center built with Placemat v1 on a GCP instance, and they are unstable due to problems caused by nested VMs.
To work around the issue, we're considering running the integration test suits in our Kubernetes cluster build on bare-metal servers instead of a GCP instance.

### Goals

- Remove the dependency on `rkt` because running containers inside a Kubernetes Pod is cumbersome.
- Coexists with v1 to make the migration tasks easier.
- Add new features and improvements
  - Support Redfish API in Virtual BMC.
  - Auto Path MTU discovery and configuration
  - Improve the implementation with modern methods

## NetworkNamespace Resource

Placemat v2 introduces new NetworkNamespace resource and removes Pod resource because it no longer runs applications with any container engines.
V2 creates a network namespace, executes its init-scripts, adds network interfaces, and runs applications inside the network namespace as instructed in the NetworkNamespace resource.
Users need to be careful to set up socket files and pid files of applications such as Bird and DHCP-Relay so that they do not collide.
For example,

```yaml
kind: NetworkStack
name: rack0-tor1
apps:
- name: bird
  command:
  - /usr/local/bird/sbin/bird
  args:
  - -f
  - -c
  - /etc/bird/bird_rack0-tor1.conf
  - -s
  - /var/run/bird/bird_rack0-tor1.ctl
- name: dhcp-relay
  command:
  - /usr/sbin/dnsmasq
  args: 
  - --pid-file=/var/run/dnsmasq_rack0-tor1.pid
  - --log-facility=-
  - --dhcp-relay 10.69.0.65,10.69.0.195
  - --dhcp-relay 10.69.0.65,10.69.1.131
interfaces:
- addresses:
  - 10.0.1.1/31
  network: s1-to-r0-1
- addresses:
  - 10.0.1.13/31
  network: s2-to-r0-1
- addresses:
  - 10.69.0.65/26
  network: r0-node1
```

For more information, see [here](resource.md#networknamespace).

## Running Placemat2 inside a Kubernetes Pod

The container running Placemat v2 inside a Kubernetes Pod needs to be privileged because reading and writing `/dev/kvm` is not permitted with a not privileged container, and Kubernetes doesn't provide the feature that enables users to allow access to a specific device for now.
See [specify hw devices in container #60748](https://github.com/kubernetes/kubernetes/issues/60748)

Also, `/dev/vhost-net` needs to be exposed in the container. Run `modprobe vhost-net` on the node where you run Placemat v2.

## New Package for v2

The deb package for Placemat v2 is placemat2, and the binaries installed by the package are placemat2 and pmctl2 so that v1 and v2 can coexist.
This is because we will install both Placemat v1 and v2 in the GCP image we are using for our CI, and then we will modify each application's CI.

## New Features

### Redfish API Support

Placemat v2 supports Redfish API in addition to IPMI. For more information, see [here](virtual_bmc.md#redfish-api).

### Auto path MTU discovery and configuration

Placemat v2 discovers Path MTU and configures it to the links it added to fix the problem that some packets are dropped due to the lower MTU that GCP sets on its instance.
GCP sets MTU 1460, but Placemat v1 sets MTU 1500 to the links.

## Obsolete Features

- Pod Resource
  - Placemat v2 no longer depends on rkt and doesn't create pods. Use NetworkNamespace resource to create a separated network stack instead.
- pmctl snapshot subcommand
  - Snapshot subcommand is not used so often.
