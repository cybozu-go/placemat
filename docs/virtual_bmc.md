Virtual BMC
===========

Overview
--------

Placemat provides virtual BMC functionality. Users in a placemat node can control other nodes via BMCs.
They can, for example, power-on/off nodes, reset nodes, and retrieve info of nodes by communicating with BMCs.

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

- Chassis Power On
- Chassis Power Off
- Chassis Power Reset / Cycle

Redfish API
-----------

### Supported Resources

- [Chassis](https://www.dell.com/support/manuals/ja-jp/idrac9-lifecycle-controller-v4.x-series/idrac9_4.00.00.00_redfishapiguide_pub/chassis?guid=guid-8cf4bcc2-28b2-4304-9f9e-85549fe81ee8&lang=en-us)
- [ChassisCollection](https://www.dell.com/support/manuals/ja-jp/idrac9-lifecycle-controller-v4.x-series/idrac9_4.00.00.00_redfishapiguide_pub/chassiscollection?guid=guid-c4ac8700-44d2-46e9-b90f-67eed0774fce&lang=en-us)

Placemat v2 returns the following fixed ChassisCollection.

```json
{
  "@odata.context": "/redfish/v1/$metadata#ChassisCollection.ChassisCollection",
  "@odata.id": "/redfish/v1/Chassis/",
  "@odata.type": "#ChassisCollection.ChassisCollection",
  "Description": "Collection of Chassis",
  "Members": [
    {
      "@odata.id": "/redfish/v1/Chassis/1"
    }
  ],
  "Members@odata.count": 1,
  "Name": "Chassis Collection"
}
```

If you use this feature, prepare certificate files and specify command line options `-bmc-cert` and `-bmc-key`.

### Authentication

Placemat v2 supports Basic authentication. In this method, user name and password are provided for each Redfish API request.
For example, `https://<USER>:<PASSWORD>@<BMC ADDRESS>/redfish/v1/Chassis` The user name and password are fixed value `cybozu`.
