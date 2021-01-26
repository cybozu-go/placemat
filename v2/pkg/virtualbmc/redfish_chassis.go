package virtualbmc

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChassisCollection represents the collection of Chassis resource instances
type ChassisCollection struct {
	OdataContext      string    `json:"@odata.context"`
	OdataID           string    `json:"@odata.id"`
	OdataType         string    `json:"@odata.type"`
	Description       string    `json:"Description"`
	Members           []OdataID `json:"Members"`
	MembersOdataCount int       `json:"Members@odata.count"`
	Name              string    `json:"Name"`
}

// Chassis represents a Chassis resource
type Chassis struct {
	OdataContext     string           `json:"@odata.context"`
	OdataID          string           `json:"@odata.id"`
	OdataType        string           `json:"@odata.type"`
	Actions          ChassisActions   `json:"Actions"`
	Assembly         OdataID          `json:"Assembly"`
	AssetTag         interface{}      `json:"AssetTag"`
	ChassisType      string           `json:"ChassisType"`
	Description      string           `json:"Description"`
	ID               string           `json:"Id"`
	IndicatorLED     string           `json:"IndicatorLED"`
	Links            ChassisLinks     `json:"Links"`
	Location         Location         `json:"Location"`
	Manufacturer     string           `json:"Manufacturer"`
	Model            string           `json:"Model"`
	Name             string           `json:"Name"`
	NetworkAdapters  OdataID          `json:"NetworkAdapters"`
	PartNumber       string           `json:"PartNumber"`
	PhysicalSecurity PhysicalSecurity `json:"PhysicalSecurity"`
	Power            OdataID          `json:"Power"`
	PowerState       PowerStatus      `json:"PowerState"`
	SKU              string           `json:"SKU"`
	SerialNumber     string           `json:"SerialNumber"`
	Status           MachineStatus    `json:"Status"`
	Thermal          OdataID          `json:"Thermal"`
}

// ChassisActions represents Chassis's Actions field
type ChassisActions struct {
	ChassisReset ChassisReset `json:"#Chassis.Reset"`
}

// ChassisReset represents Chassis's ChassisReset field
type ChassisReset struct {
	ResetTypeRedfishAllowableValues []ResetType `json:"ResetType@Redfish.AllowableValues"`
	Target                          string      `json:"target"`
}

// ChassisLinks represents Chassis's Links field
type ChassisLinks struct {
	ComputerSystems             []OdataID     `json:"ComputerSystems"`
	ComputerSystemsOdataCount   int           `json:"ComputerSystems@odata.count"`
	Contains                    []OdataID     `json:"Contains"`
	ContainsOdataCount          int           `json:"Contains@odata.count"`
	CooledBy                    []OdataID     `json:"CooledBy"`
	CooledByOdataCount          int           `json:"CooledBy@odata.count"`
	Drives                      []interface{} `json:"Drives"`
	DrivesOdataCount            int           `json:"Drives@odata.count"`
	ManagedBy                   []OdataID     `json:"ManagedBy"`
	ManagedByOdataCount         int           `json:"ManagedBy@odata.count"`
	ManagersInChassis           []OdataID     `json:"ManagersInChassis"`
	ManagersInChassisOdataCount int           `json:"ManagersInChassis@odata.count"`
	PCIeDevices                 []OdataID     `json:"PCIeDevices"`
	PCIeDevicesOdataCount       int           `json:"PCIeDevices@odata.count"`
	PoweredBy                   []OdataID     `json:"PoweredBy"`
	PoweredByOdataCount         int           `json:"PoweredBy@odata.count"`
	Storage                     []OdataID     `json:"Storage"`
	StorageOdataCount           int           `json:"Storage@odata.count"`
}

// Location represents Chassis's Location field
type Location struct {
	Info          string        `json:"Info"`
	InfoFormat    string        `json:"InfoFormat"`
	Placement     Placemat      `json:"Placement"`
	PostalAddress PostalAddress `json:"PostalAddress"`
}

// Placemat represents Chassis's Placemat field
type Placemat struct {
	Rack string `json:"Rack"`
	Row  string `json:"Row"`
}

// PostalAddress represents Chassis's PostalAddress field
type PostalAddress struct {
	Building string `json:"Building"`
	Room     string `json:"Room"`
}

// PhysicalSecurity represents Chassis's PhysicalSecurity field
type PhysicalSecurity struct {
	IntrusionSensor       string `json:"IntrusionSensor"`
	IntrusionSensorNumber int    `json:"IntrusionSensorNumber"`
	IntrusionSensorReArm  string `json:"IntrusionSensorReArm"`
}

var chassisCollectionResponse = ResourceCollection{
	OdataContext: "/redfish/v1/$metadata#ChassisCollection.ChassisCollection",
	OdataID:      "/redfish/v1/Chassis/",
	OdataType:    "#ChassisCollection.ChassisCollection",
	Description:  "Collection of Chassis",
	Members: []OdataID{
		{OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s", systemID)},
	},
	MembersOdataCount: 1,
	Name:              "Chassis Collection",
}

func handleChassisCollection(c *gin.Context) {
	c.JSON(http.StatusOK, chassisCollectionResponse)
}

func (r *redfishServer) handleChassis(c *gin.Context) {
	id := c.Param("id")
	_, ok := r.systemIDs[id]
	if !ok {
		c.JSON(http.StatusNotFound, createChassisNotFoundErrorResponse(id))
	}

	c.JSON(http.StatusOK, createChassisResponse(id, r.machine.PowerStatus()))
}

func createChassisResponse(chassisID string, powerState PowerStatus) Chassis {
	return Chassis{
		OdataContext: "/redfish/v1/$metadata#Chassis.Chassis",
		OdataID:      fmt.Sprintf("/redfish/v1/Chassis/%s", chassisID),
		OdataType:    "#Chassis.v1_6_0.Chassis",
		Actions: ChassisActions{
			ChassisReset: ChassisReset{
				ResetTypeRedfishAllowableValues: []ResetType{
					ResetTypeOn,
					ResetTypeForceOff,
				},
				Target: fmt.Sprintf("/redfish/v1/Chassis/%s/Actions/Chassis.Reset", chassisID),
			},
		},
		Assembly: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Assembly", chassisID),
		},
		AssetTag:     nil,
		ChassisType:  "RackMount",
		Description:  "It represents the properties for physical components for any system.It represent racks, rackmount servers, blades, standalone, modular systems,enclosures, and all other containers.The non-cpu/device centric parts of the schema are all accessed either directly or indirectly through this resource.",
		ID:           chassisID,
		IndicatorLED: "Off",
		Links: ChassisLinks{
			ComputerSystems: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Systems/%s", chassisID),
				},
			},
			ComputerSystemsOdataCount: 1,
			Contains:                  []OdataID{},
			ContainsOdataCount:        0,
			CooledBy: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/Fans/1A", chassisID),
				},
				{
					OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/Fans/1B", chassisID),
				},
			},
			CooledByOdataCount: 2,
			Drives:             []interface{}{},
			DrivesOdataCount:   0,
			ManagedBy: []OdataID{
				{
					OdataID: "/redfish/v1/Managers/1",
				},
			},
			ManagedByOdataCount: 1,
			ManagersInChassis: []OdataID{
				{
					OdataID: "/redfish/v1/Managers/1",
				},
			},
			ManagersInChassisOdataCount: 1,
			PCIeDevices: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/PCIeDevice/0-31", chassisID),
				},
			},
			PCIeDevicesOdataCount: 1,
			PoweredBy: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Power/PowerSupplies/PSU.Slot.1", chassisID),
				},
			},
			PoweredByOdataCount: 1,
			Storage: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/Storage/CPU.1", chassisID),
				},
			},
			StorageOdataCount: 1,
		},
		Location: Location{
			Info:       ";;;;1",
			InfoFormat: "DataCenter;RoomName;Aisle;RackName;RackSlot",
			Placement: Placemat{
				Rack: "",
				Row:  "",
			},
			PostalAddress: PostalAddress{
				Building: "",
				Room:     "",
			},
		},
		Manufacturer: "Placemat Inc.",
		Model:        "Placemat XXX",
		Name:         "Computer System Chassis",
		NetworkAdapters: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/NetworkAdapters", chassisID),
		},
		PartNumber: "0000",
		PhysicalSecurity: PhysicalSecurity{
			IntrusionSensor:       "XXX",
			IntrusionSensorNumber: 111,
			IntrusionSensorReArm:  "XXX",
		},
		Power: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Power", chassisID),
		},
		PowerState:   powerState,
		SKU:          "XXXXXX",
		SerialNumber: "XXXXXX",
		Status: MachineStatus{
			Health:       "OK",
			HealthRollup: "OK",
			State:        "Enabled",
		},
		Thermal: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Thermal", chassisID),
		},
	}
}

func createChassisNotFoundErrorResponse(chassisID string) ErrorResponse {
	return ErrorResponse{
		Error: Error{
			MessageExtendedInfo: []MessageExtendedInfo{
				{
					Message:                     fmt.Sprintf("Unable to complete the operation because the resource %s entered in not found.", chassisID),
					MessageArgs:                 []string{chassisID},
					MessageArgsOdataCount:       1,
					MessageID:                   "X.X.X",
					RelatedProperties:           []interface{}{},
					RelatedPropertiesOdataCount: 0,
					Resolution:                  "Enter the correct resource and retry the operation. For information about valid resource, see the Redfish Users Guide available on the support site.",
					Severity:                    "Critical",
				},
				{
					Message:                     fmt.Sprintf("The resource at the URI %s was not found.", chassisID),
					MessageArgs:                 []string{chassisID},
					MessageArgsOdataCount:       1,
					MessageID:                   "Base.X.X.ResourceMissingAtURI",
					RelatedProperties:           []interface{}{""},
					RelatedPropertiesOdataCount: 1,
					Resolution:                  "Place a valid resource at the URI or correct the URI and resubmit the request.",
					Severity:                    "Critical",
				},
			},
			Code:    "Base.X.X.GeneralError",
			Message: "A general error has occurred. See ExtendedInfo for more information",
		}}
}

func (r *redfishServer) handleChassisActionsReset(c *gin.Context) {
	var json RequestBody
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resetType := json.ResetType
	if resetType != ResetTypeOn && resetType != ResetTypeForceOff {
		c.JSON(http.StatusBadRequest, gin.H{"error": ""})
		return
	}

	switch resetType {
	case ResetTypeOn:
		powerStatus := r.machine.PowerStatus()
		if powerStatus == PowerStatusOn || powerStatus == PowerStatusPoweringOn {
			c.JSON(http.StatusConflict, nil)
		}
		if err := r.machine.PowerOn(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case ResetTypeForceOff:
		powerStatus := r.machine.PowerStatus()
		if powerStatus == PowerStatusOff || powerStatus == PowerStatusPoweringOff {
			c.JSON(http.StatusConflict, nil)
		}
		if err := r.machine.PowerOff(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusNoContent, nil)
}
