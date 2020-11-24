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

- [ChassisCollection](https://www.dell.com/support/manuals/ja-jp/idrac9-lifecycle-controller-v3.3-series/idrac9_3.36_redfishapiguide/chassiscollection?guid=guid-c4ac8700-44d2-46e9-b90f-67eed0774fce&lang=en-us)
- [Chassis](https://www.dell.com/support/manuals/ja-jp/idrac9-lifecycle-controller-v3.3-series/idrac9_3.36_redfishapiguide/chassiscollection?guid=guid-c4ac8700-44d2-46e9-b90f-67eed0774fce&lang=en-us)
  - [Supported Action - Reset](https://www.dell.com/support/manuals/ja-jp/idrac9-lifecycle-controller-v3.3-series/idrac9_3.36_redfishapiguide/supported-action-%E2%80%94-reset?guid=guid-eae5f0af-bfdf-4915-b097-2f6f771e5c08&lang=en-us)

Placemat v2 returns the following fixed ChassisCollection.

```json
{
  "@odata.context": "/redfish/v1/$metadata#ChassisCollection.ChassisCollection",
  "@odata.id": "/redfish/v1/Chassis/",
  "@odata.type": "#ChassisCollection.ChassisCollection",
  "Description": "Collection of Chassis",
  "Members": [
    {
      "@odata.id": "/redfish/v1/Chassis/System.Embedded.1"
    }
  ],
  "Members@odata.count": 1,
  "Name": "Chassis Collection"
}
```

If you use this feature, prepare certificate files and specify command line options `-bmc-cert` and `-bmc-key`.

### Supported Action

Placemat V2 supports Reset Action for Chassis resource. You can confirm the supported actions in the Actions field of the Chassis Resource.

```json
{
  "@odata.context": "/redfish/v1/$metadata#Chassis.Chassis",
  "@odata.id": "/redfish/v1/Chassis/System.Embedded.1",
  "@odata.type": "#Chassis.v1_6_0.Chassis",
  "Actions": {
    "#Chassis.Reset": {
      "ResetType@Redfish.AllowableValues": [
        "On",
        "ForceOff"
      ],
      "target": "/redfish/v1/Chassis/System.Embedded.1/Actions/Chassis.Reset"
    }
  },
}
```
