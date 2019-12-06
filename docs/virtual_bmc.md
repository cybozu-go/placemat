Virtual BMC
===========

Overview
--------

Placemat provides virtual BMC functionality.
Users in a placemat node can control other nodes via BMCs.
They can, for example, power-on/off nodes, reset nodes, and retrieve info
of nodes by communicating with BMCs.

The address range of BMC network should be given in a resources YAML file.

```yaml
kind: Network
name: bmc
spec:
  type: bmc
  use-nat: false
  addresses:
  - 10.0.0.1/24
```

In this example, `10.0.0.0/24` is the address range of BMC network.

How it works
------------

1. Each node chooses an IP address of BMC from the range and notify of it
   to the Placemat process via a special character device `/dev/virtio-ports/placemat`.

2. The Placemat process starts listening on the address.

3. An IPMI client on a placemat node sends commands to the BMC address.

4. The Placemat process interpret commands and controls the QEMU process
   of the node via its monitor socket.

Supported IPMI commands
-----------------------

(TBD)

HTTPS server
------------

For test purpose, Placemat deploys HTTPS servers on https://<BMC address>:443.
They return 200 OK only.

**If you use this feature, prepare certificate files and specify command line option**
