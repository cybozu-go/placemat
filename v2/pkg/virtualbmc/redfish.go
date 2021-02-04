package virtualbmc

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type redfishServer struct {
	machine   Machine
	systemIDs map[string]struct{}
}

// OdataID represents the unique identifier for a resource
type OdataID struct {
	OdataID string `json:"@odata.id"`
}

// ServiceRoot represents Redfish service root(/redfish/v1)
type ServiceRoot struct {
	OdataContext              string                    `json:"@odata.context"`
	OdataID                   string                    `json:"@odata.id"`
	OdataType                 string                    `json:"@odata.type"`
	AccountService            OdataID                   `json:"AccountService"`
	Chassis                   OdataID                   `json:"Chassis"`
	Description               string                    `json:"Description"`
	EventService              OdataID                   `json:"EventService"`
	Fabrics                   OdataID                   `json:"Fabrics"`
	ID                        string                    `json:"Id"`
	JSONSchemas               OdataID                   `json:"JsonSchemas"`
	Links                     ServiceRootLinks          `json:"Links"`
	Managers                  OdataID                   `json:"Managers"`
	Name                      string                    `json:"Name"`
	Oem                       ServiceRootOem            `json:"Oem"`
	Product                   string                    `json:"Product"`
	ProtocolFeaturesSupported ProtocolFeaturesSupported `json:"ProtocolFeaturesSupported"`
	RedfishVersion            string                    `json:"RedfishVersion"`
	Registries                OdataID                   `json:"Registries"`
	SessionService            OdataID                   `json:"SessionService"`
	Systems                   OdataID                   `json:"Systems"`
	Tasks                     OdataID                   `json:"Tasks"`
	UpdateService             OdataID                   `json:"UpdateService"`
}

// ServiceRootLinks represents ServiceRoot's Links field
type ServiceRootLinks struct {
	Sessions OdataID `json:"Sessions"`
}

// ServiceRootOem represents ServiceRoot's Oem field
type ServiceRootOem struct {
}

// ProtocolFeaturesSupported represents ServiceRoot's ProtocolFeaturesSupported field
type ProtocolFeaturesSupported struct {
	ExpandQuery ExpandQuery `json:"ExpandQuery"`
	FilterQuery bool        `json:"FilterQuery"`
	SelectQuery bool        `json:"SelectQuery"`
}

// ExpandQuery represents ServiceRoot's ExpandQuery field
type ExpandQuery struct {
	ExpandAll bool `json:"ExpandAll"`
	Levels    bool `json:"Levels"`
	Links     bool `json:"Links"`
	MaxLevels int  `json:"MaxLevels"`
	NoLinks   bool `json:"NoLinks"`
}

// ResourceCollection represents the collection of resource instances
type ResourceCollection struct {
	OdataContext      string    `json:"@odata.context"`
	OdataID           string    `json:"@odata.id"`
	OdataType         string    `json:"@odata.type"`
	Description       string    `json:"Description"`
	Members           []OdataID `json:"Members"`
	MembersOdataCount int       `json:"Members@odata.count"`
	Name              string    `json:"Name"`
}

// RequestBody represents Post request body
type RequestBody struct {
	ResetType ResetType `json:"ResetType"`
}

// ErrorResponse represents Error response
type ErrorResponse struct {
	Error Error `json:"error"`
}

// Error represents ErrorResponse's Error field
type Error struct {
	MessageExtendedInfo []MessageExtendedInfo `json:"@Message.ExtendedInfo"`
	Code                string                `json:"code"`
	Message             string                `json:"message"`
}

// MessageExtendedInfo represents ErrorResponse's MessageExtendedInfo field
type MessageExtendedInfo struct {
	Message                     string        `json:"Message"`
	MessageArgs                 []string      `json:"MessageArgs"`
	MessageArgsOdataCount       int           `json:"MessageArgs@odata.count"`
	MessageID                   string        `json:"MessageId"`
	RelatedProperties           []interface{} `json:"RelatedProperties"`
	RelatedPropertiesOdataCount int           `json:"RelatedProperties@odata.count"`
	Resolution                  string        `json:"Resolution"`
	Severity                    string        `json:"Severity"`
}

const systemID = "System.Embedded.1"

type ResetType string

const (
	ResetTypeOn               = ResetType("On")
	ResetTypeForceOff         = ResetType("ForceOff")
	ResetTypeForceRestart     = ResetType("ForceRestart")
	ResetTypeGracefulShutdown = ResetType("GracefulShutdown")
	ResetTypePushPowerButton  = ResetType("PushPowerButton")
	ResetTypeNmi              = ResetType("Nmi")
)

var serviceRootResponse = ServiceRoot{
	OdataContext: "/redfish/v1/$metadata#ServiceRoot.ServiceRoot",
	OdataID:      "/redfish/v1",
	OdataType:    "#ServiceRoot.v1_3_0.ServiceRoot",
	AccountService: OdataID{
		OdataID: "/redfish/v1/Managers/Embedded.1/AccountService",
	},
	Chassis: OdataID{
		OdataID: "/redfish/v1/Chassis",
	},
	Description: "Root Service",
	EventService: OdataID{
		OdataID: "/redfish/v1/EventService",
	},
	Fabrics: OdataID{
		OdataID: "/redfish/v1/Fabrics",
	},
	ID: "RootService",
	JSONSchemas: OdataID{
		OdataID: "/redfish/v1/JSONSchemas",
	},
	Links: ServiceRootLinks{
		Sessions: OdataID{
			OdataID: "/redfish/v1/Sessions",
		},
	},
	Managers: OdataID{
		OdataID: "/redfish/v1/Managers",
	},
	Name:    "Root Service",
	Oem:     ServiceRootOem{},
	Product: "Placemat",
	ProtocolFeaturesSupported: ProtocolFeaturesSupported{
		ExpandQuery: ExpandQuery{
			ExpandAll: true,
			Levels:    true,
			Links:     true,
			MaxLevels: 1,
			NoLinks:   true,
		},
		FilterQuery: true,
		SelectQuery: true,
	},
	RedfishVersion: "1.4.0",
	Registries: OdataID{
		OdataID: "/redfish/v1/Registries",
	},
	SessionService: OdataID{
		OdataID: "/redfish/v1/SessionService",
	},
	Systems: OdataID{
		OdataID: "/redfish/v1/Systems",
	},
	Tasks: OdataID{
		OdataID: "/redfish/v1/TaskService",
	},
	UpdateService: OdataID{
		OdataID: "/redfish/v1/UpdateService",
	},
}

func newRedfishServer(machine Machine) *redfishServer {
	return &redfishServer{
		machine:   machine,
		systemIDs: map[string]struct{}{systemID: {}},
	}
}

func handleServiceRoot(c *gin.Context) {
	c.JSON(http.StatusOK, serviceRootResponse)
}
