package virtualbmc

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ComputerSystem represents a ComputerSystem resource
type ComputerSystem struct {
	OdataContext            string                `json:"@odata.context"`
	OdataID                 string                `json:"@odata.id"`
	OdataType               string                `json:"@odata.type"`
	Actions                 ComputerSystemActions `json:"Actions"`
	AssetTag                string                `json:"AssetTag"`
	Bios                    OdataID               `json:"Bios"`
	BiosVersion             string                `json:"BiosVersion"`
	Boot                    Boot                  `json:"Boot"`
	Description             string                `json:"Description"`
	EthernetInterfaces      OdataID               `json:"EthernetInterfaces"`
	HostName                string                `json:"HostName"`
	HostWatchdogTimer       HostWatchdogTimer     `json:"HostWatchdogTimer"`
	HostingRoles            []interface{}         `json:"HostingRoles"`
	HostingRolesOdataCount  int                   `json:"HostingRoles@odata.count"`
	ID                      string                `json:"Id"`
	IndicatorLED            string                `json:"IndicatorLED"`
	Links                   ComputerSystemLinks   `json:"Links"`
	Manufacturer            string                `json:"Manufacturer"`
	Memory                  OdataID               `json:"Memory"`
	MemorySummary           MemorySummary         `json:"MemorySummary"`
	Model                   string                `json:"Model"`
	Name                    string                `json:"Name"`
	NetworkInterfaces       OdataID               `json:"NetworkInterfaces"`
	Oem                     ComputerSystemOem     `json:"Oem"`
	PCIeDevices             []OdataID             `json:"PCIeDevices"`
	PCIeDevicesOdataCount   int                   `json:"PCIeDevices@odata.count"`
	PCIeFunctions           []OdataID             `json:"PCIeFunctions"`
	PCIeFunctionsOdataCount int                   `json:"PCIeFunctions@odata.count"`
	PartNumber              string                `json:"PartNumber"`
	PowerState              PowerStatus           `json:"PowerState"`
	ProcessorSummary        ProcessorSummary      `json:"ProcessorSummary"`
	Processors              OdataID               `json:"Processors"`
	SKU                     string                `json:"SKU"`
	SecureBoot              OdataID               `json:"SecureBoot"`
	SerialNumber            string                `json:"SerialNumber"`
	SimpleStorage           OdataID               `json:"SimpleStorage"`
	Status                  MachineStatus         `json:"Status"`
	Storage                 OdataID               `json:"Storage"`
	SystemType              string                `json:"SystemType"`
	TrustedModules          []TrustedModule       `json:"TrustedModules"`
	UUID                    string                `json:"UUID"`
}

// ComputerSystemActions represents ComputerSystem resource's Actions field
type ComputerSystemActions struct {
	ComputerSystemReset ComputerSystemReset `json:"#ComputerSystem.Reset"`
}

// ComputerSystemReset represents ComputerSystem resource's Reset field
type ComputerSystemReset struct {
	ResetTypeRedfishAllowableValues []ResetType `json:"ResetType@Redfish.AllowableValues"`
	Target                          string      `json:"target"`
}

// Boot represents ComputerSystem resource's Boot field
type Boot struct {
	BootOptions                                    OdataID  `json:"BootOptions"`
	BootOrder                                      []string `json:"BootOrder"`
	BootOrderOdataCount                            int      `json:"BootOrder@odata.count"`
	BootSourceOverrideEnabled                      string   `json:"BootSourceOverrideEnabled"`
	BootSourceOverrideMode                         string   `json:"BootSourceOverrideMode"`
	BootSourceOverrideTarget                       string   `json:"BootSourceOverrideTarget"`
	BootSourceOverrideTargetRedfishAllowableValues []string `json:"BootSourceOverrideTarget@Redfish.AllowableValues"`
	UefiTargetBootSourceOverride                   string   `json:"UefiTargetBootSourceOverride"`
}

// HostWatchdogTimer represents ComputerSystem resource's HostWatchdogTimer field
type HostWatchdogTimer struct {
	FunctionEnabled bool   `json:"FunctionEnabled"`
	Status          Status `json:"Status"`
	TimeoutAction   string `json:"TimeoutAction"`
}

// Status represents ComputerSystem resource's Status field
type Status struct {
	State string `json:"State"`
}

// ComputerSystemLinks represents ComputerSystem resource's Links field
type ComputerSystemLinks struct {
	Chassis             []OdataID              `json:"Chassis"`
	ChassisOdataCount   int                    `json:"Chassis@odata.count"`
	CooledBy            []OdataID              `json:"CooledBy"`
	CooledByOdataCount  int                    `json:"CooledBy@odata.count"`
	ManagedBy           []OdataID              `json:"ManagedBy"`
	ManagedByOdataCount int                    `json:"ManagedBy@odata.count"`
	Oem                 ComputerSystemLinksOem `json:"Oem"`
	PoweredBy           []OdataID              `json:"PoweredBy"`
	PoweredByOdataCount int                    `json:"PoweredBy@odata.count"`
}

// ComputerSystemLinksOem represents ComputerSystem resource's Links Oem field
type ComputerSystemLinksOem struct {
}

// MemorySummary represents ComputerSystem resource's MemorySummary field
type MemorySummary struct {
	MemoryMirroring      string        `json:"MemoryMirroring"`
	Status               MachineStatus `json:"Status"`
	TotalSystemMemoryGiB float64       `json:"TotalSystemMemoryGiB"`
}

// MachineStatus represents ComputerSystem resource's Status field
type MachineStatus struct {
	Health       string `json:"Health"`
	HealthRollup string `json:"HealthRollup"`
	State        string `json:"State"`
}

// ComputerSystemOem represents ComputerSystem resource's Oem field
type ComputerSystemOem struct {
}

// ProcessorSummary represents ComputerSystem resource's ProcessorSummary field
type ProcessorSummary struct {
	Count                 int           `json:"Count"`
	LogicalProcessorCount int           `json:"LogicalProcessorCount"`
	Model                 string        `json:"Model"`
	Status                MachineStatus `json:"Status"`
}

// TrustedModule represents ComputerSystem resource's TrustedModule field
type TrustedModule struct {
	FirmwareVersion string `json:"FirmwareVersion"`
	InterfaceType   string `json:"InterfaceType"`
	Status          Status `json:"Status"`
}

var computerSystemCollectionResponse = ResourceCollection{
	OdataContext: "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection",
	OdataID:      "/redfish/v1/Systems",
	OdataType:    "#ComputerSystemCollection.ComputerSystemCollection",
	Description:  "Collection of Computer Systems",
	Members: []OdataID{
		{OdataID: fmt.Sprintf("/redfish/v1/Systems/%s", systemID)},
	},
	MembersOdataCount: 1,
	Name:              "Computer System Collection",
}

var serverIsAlreadyPoweredOffResponse = ErrorResponse{
	Error: Error{
		MessageExtendedInfo: []MessageExtendedInfo{
			{
				Message:                     "Server is already powered OFF",
				MessageArgs:                 []string{},
				MessageArgsOdataCount:       0,
				MessageID:                   "X.X.X",
				RelatedProperties:           []interface{}{},
				RelatedPropertiesOdataCount: 0,
				Resolution:                  "No response action is required.",
				Severity:                    "Informational",
			},
		},
		Code:    "Base.X.X.GeneralError",
		Message: "A general error has occurred. See ExtendedInfo for more information",
	},
}

var serverIsAlreadyPoweredOnResponse = ErrorResponse{
	Error: Error{
		MessageExtendedInfo: []MessageExtendedInfo{
			{
				Message:                     "Server is already powered ON",
				MessageArgs:                 []string{},
				MessageArgsOdataCount:       0,
				MessageID:                   "X.X.X",
				RelatedProperties:           []interface{}{},
				RelatedPropertiesOdataCount: 0,
				Resolution:                  "No response action is required.",
				Severity:                    "Informational",
			},
		},
		Code:    "Base.X.X.GeneralError",
		Message: "A general error has occurred. See ExtendedInfo for more information",
	},
}

func handleComputerSystemCollection(c *gin.Context) {
	c.JSON(http.StatusOK, computerSystemCollectionResponse)
}

func (r Redfish) handleComputerSystem(c *gin.Context) {
	id := c.Param("id")
	_, ok := r.systemIDs[id]
	if !ok {
		c.JSON(http.StatusNotFound, nil)
	}

	c.JSON(http.StatusOK, createComputerSystemResponse(id, r.machine.PowerStatus()))
}

func createComputerSystemResponse(systemID string, powerState PowerStatus) ComputerSystem {
	return ComputerSystem{
		OdataContext: "/redfish/v1/$metadata#ComputerSystem.ComputerSystem",
		OdataID:      fmt.Sprintf("/redfish/v1/Systems/%s", systemID),
		OdataType:    "#ComputerSystem.v1_5_0.ComputerSystem",
		Actions: ComputerSystemActions{
			ComputerSystemReset: ComputerSystemReset{
				ResetTypeRedfishAllowableValues: []ResetType{
					ResetTypeOn,
					ResetTypeForceOff,
					ResetTypeForceRestart,
					ResetTypeGracefulShutdown,
					ResetTypePushPowerButton,
					ResetTypeNmi,
				},
				Target: fmt.Sprintf("/redfish/v1/Systems/%s/Actions/ComputerSystem.Reset", systemID),
			},
		},
		AssetTag: "",
		Bios: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/Bios", systemID),
		},
		BiosVersion: "X.X.X",
		Boot: Boot{
			BootOptions: OdataID{
				OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/BootOptions", systemID),
			},
			BootOrder: []string{
				"Boot0000",
				"Boot0001",
			},
			BootOrderOdataCount:       2,
			BootSourceOverrideEnabled: "Once",
			BootSourceOverrideMode:    "UEFI",
			BootSourceOverrideTarget:  "None",
			BootSourceOverrideTargetRedfishAllowableValues: []string{
				"None",
				"Pxe",
				"Floppy",
				"Cd",
				"Hdd",
				"BiosSetup",
				"Utilities",
				"UefiTarget",
				"SDCard",
				"UefiHttp",
			},
			UefiTargetBootSourceOverride: "",
		},
		Description: "Computer System which represents a machine (physical or virtual) and the local resources such as memory, cpu and other devices that can be accessed from that machine.",
		EthernetInterfaces: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/EthernetInterfaces", systemID),
		},
		HostName: "",
		HostWatchdogTimer: HostWatchdogTimer{
			FunctionEnabled: false,
			Status: Status{
				State: "Disabled",
			},
			TimeoutAction: "None",
		},
		HostingRoles:           []interface{}{},
		HostingRolesOdataCount: 0,
		ID:                     systemID,
		IndicatorLED:           "Off",
		Links: ComputerSystemLinks{
			Chassis: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s", systemID),
				},
			},
			ChassisOdataCount: 1,
			CooledBy: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/Fans/Fan.Embedded.1A", systemID),
				},
				{
					OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/Fans/Fan.Embedded.1B", systemID),
				},
			},
			CooledByOdataCount: 2,
			ManagedBy: []OdataID{
				{
					OdataID: "/redfish/v1/Managers/1",
				},
			},
			ManagedByOdataCount: 1,
			Oem:                 ComputerSystemLinksOem{},
			PoweredBy: []OdataID{
				{
					OdataID: fmt.Sprintf("/redfish/v1/Chassis/%s/Power/PowerSupplies/PSU.Slot.1", systemID),
				},
			},
			PoweredByOdataCount: 1,
		},
		Manufacturer: "Placemat",
		Memory: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/Memory", systemID),
		},
		MemorySummary: MemorySummary{
			MemoryMirroring: "System",
			Status: MachineStatus{
				Health:       "OK",
				HealthRollup: "OK",
				State:        "Enabled",
			},
			TotalSystemMemoryGiB: 300,
		},
		Model: "XXX",
		Name:  "System",
		NetworkInterfaces: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/NetworkInterfaces", systemID),
		},
		Oem: ComputerSystemOem{},
		PCIeDevices: []OdataID{
			{
				OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/PCIeDevice/0-31", systemID),
			},
		},
		PCIeDevicesOdataCount: 1,
		PCIeFunctions: []OdataID{
			{
				OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/PCIeFunction/0-31-4", systemID),
			},
		},
		PCIeFunctionsOdataCount: 1,
		PartNumber:              "XXXX",
		PowerState:              powerState,
		ProcessorSummary: ProcessorSummary{
			Count:                 2,
			LogicalProcessorCount: 28,
			Model:                 "XXXX",
			Status: MachineStatus{
				Health:       "OK",
				HealthRollup: "OK",
				State:        "Enabled",
			},
		},
		Processors: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/Processors", systemID),
		},
		SKU: "XXXX",
		SecureBoot: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/SecureBoot", systemID),
		},
		SerialNumber: "XXXX",
		SimpleStorage: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/SimpleStorage/Controllers", systemID),
		},
		Status: MachineStatus{
			Health:       "OK",
			HealthRollup: "OK",
			State:        "Enabled",
		},
		Storage: OdataID{
			OdataID: fmt.Sprintf("/redfish/v1/Systems/%s/Storage", systemID),
		},
		SystemType: "Virtual",
		TrustedModules: []TrustedModule{
			{
				FirmwareVersion: "X.X.X.X",
				InterfaceType:   "TPM2_0",
				Status: Status{
					State: "Enabled",
				},
			},
		},
		UUID: "XXXX",
	}
}

func (r Redfish) handleComputerSystemActionsReset(c *gin.Context) {
	var json RequestBody
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch json.ResetType {
	case ResetTypeOn:
		powerStatus := r.machine.PowerStatus()
		if powerStatus == PowerStatusOn || powerStatus == PowerStatusPoweringOn {
			c.JSON(http.StatusConflict, serverIsAlreadyPoweredOnResponse)
			return
		}
		if err := r.machine.PowerOn(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case ResetTypeForceOff:
		powerStatus := r.machine.PowerStatus()
		if powerStatus == PowerStatusOff || powerStatus == PowerStatusPoweringOff {
			c.JSON(http.StatusConflict, serverIsAlreadyPoweredOffResponse)
			return
		}
		if err := r.machine.PowerOff(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case ResetTypeGracefulShutdown:
		powerStatus := r.machine.PowerStatus()
		if powerStatus == PowerStatusOff || powerStatus == PowerStatusPoweringOff {
			c.JSON(http.StatusConflict, serverIsAlreadyPoweredOffResponse)
			return
		}
		if err := r.machine.PowerOff(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case ResetTypeForceRestart:
		powerStatus := r.machine.PowerStatus()
		if powerStatus == PowerStatusOff || powerStatus == PowerStatusPoweringOff {
			c.JSON(http.StatusConflict, serverIsAlreadyPoweredOffResponse)
			return
		}
		if err := r.machine.PowerOff(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := r.machine.PowerOn(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusNoContent, nil)
}
