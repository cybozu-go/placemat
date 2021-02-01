package placemat

import (
	"context"
	"net"
	"net/http"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/virtualbmc"
	"github.com/cybozu-go/placemat/v2/pkg/vm"
	"github.com/cybozu-go/well"
	"github.com/gin-gonic/gin"
)

// NodeStatus represents status of a Node
type NodeStatus struct {
	Name        string                 `json:"name"`
	Taps        map[string]string      `json:"taps"`
	Volumes     []string               `json:"volumes"`
	CPU         int                    `json:"cpu"`
	Memory      string                 `json:"memory"`
	UEFI        bool                   `json:"uefi"`
	TPM         bool                   `json:"tpm"`
	SMBIOS      SMBIOSStatus           `json:"smbios"`
	PowerStatus virtualbmc.PowerStatus `json:"power_status"`
	SocketPath  string                 `json:"socket_path"`
}

// SMBIOSStatus represents SMBIOS of a Node
type SMBIOSStatus struct {
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
	Serial       string `json:"serial"`
}

type apiServer struct {
	cluster *cluster
	runtime *vm.Runtime
}

func newAPIServer(cluster *cluster, r *vm.Runtime) *apiServer {
	return &apiServer{
		cluster: cluster,
		runtime: r,
	}
}

func (s *apiServer) start(ctx context.Context, listener net.Listener) error {
	serv := &well.HTTPServer{
		Server: &http.Server{
			Handler: s.prepareRouter(),
		},
	}

	go func() {
		<-ctx.Done()
		listener.Close()
		serv.Close()
	}()

	log.Info("Start Placemat API server", map[string]interface{}{"address": listener.Addr()})

	if err := serv.Serve(listener); err != nil {
		return err
	}

	return nil
}

func (s *apiServer) prepareRouter() http.Handler {
	router := gin.Default()
	router.GET("/nodes", s.handleNodes)
	router.GET("/nodes/:name", s.handleNode)
	router.POST("/nodes/:name/:action", s.handleNodeAction)

	return router
}

func (s *apiServer) handleNode(c *gin.Context) {
	name := c.Param("name")
	spec, ok := s.cluster.nodeSpecMap[name]
	if !ok {
		c.JSON(http.StatusNotFound, nil)
		return
	}
	status, err := newNodeStatus(spec, s.cluster.nodeMap[name], s.cluster.vms[spec.SMBIOS.Serial], s.runtime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	c.JSON(http.StatusOK, status)
}

func (s *apiServer) handleNodes(c *gin.Context) {
	statuses := make([]*NodeStatus, len(s.cluster.nodeSpecs))
	for i, spec := range s.cluster.nodeSpecs {
		status, err := newNodeStatus(spec, s.cluster.nodeMap[spec.Name], s.cluster.vms[spec.SMBIOS.Serial], s.runtime)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		statuses[i] = status
	}
	c.JSON(http.StatusOK, statuses)
}

func (s *apiServer) handleNodeAction(c *gin.Context) {
	name := c.Param("name")
	action := c.Param("action")

	spec, ok := s.cluster.nodeSpecMap[name]
	if !ok {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	vm := s.cluster.vms[spec.SMBIOS.Serial]
	switch action {
	case "start":
		if err := vm.PowerOn(); err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
	case "stop":
		if err := vm.PowerOff(); err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
	case "restart":
		if err := vm.PowerOff(); err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		if err := vm.PowerOn(); err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
	default:
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	c.JSON(http.StatusOK, nil)
}

func newNodeStatus(spec *types.NodeSpec, node vm.Node, vm vm.VM, runtime *vm.Runtime) (*NodeStatus, error) {
	powerStatus, err := vm.PowerStatus()
	if err != nil {
		return nil, err
	}

	status := &NodeStatus{
		Name:        spec.Name,
		Taps:        node.Taps(),
		CPU:         spec.CPU,
		Memory:      spec.Memory,
		UEFI:        spec.UEFI,
		TPM:         spec.TPM,
		PowerStatus: powerStatus,
	}
	status.SMBIOS.Serial = spec.SMBIOS.Serial
	status.SMBIOS.Manufacturer = spec.SMBIOS.Manufacturer
	status.SMBIOS.Product = spec.SMBIOS.Product
	if !runtime.Graphic {
		status.SocketPath = vm.SocketPath()
	}
	status.Volumes = make([]string, len(spec.Volumes))
	for i, v := range spec.Volumes {
		status.Volumes[i] = v.Name
	}
	return status, nil
}
