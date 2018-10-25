package placemat

import (
	"net/http"
	"strings"
)

type Server struct {
	cluster *Cluster
	vms     map[string]*NodeVM
	runtime *Runtime
}

type NodeStatus struct {
	Name       string            `json:"name"`
	Taps       map[string]string `json:"taps"`
	Volumes    []string          `json:"volumes"`
	CPU        int               `json:"cpu"`
	Memory     string            `json:"memory"`
	UEFI       bool              `json:"uefi"`
	SMBIOS     SMBIOSConfig      `json:"smbios"`
	IsRunning  bool              `json:"is_running"`
	SocketPath string            `json:"socket_path"`
}

type PodStatus struct {
	Name    string            `json:"name"`
	UUID    string            `json:"uuid"`
	Veths   map[string]string `json:"veths"`
	Volumes []string          `json:"volumes"`
	Apps    []string          `json:"apps"`
}

func NewServer(cluster *Cluster, vms map[string]*NodeVM, r *Runtime) *Server {
	return &Server{
		cluster: cluster,
		vms:     vms,
		runtime: r,
	}
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/nodes") {
		s.handleNodes(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/pods") {
		s.handlePods(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/networks") {
		s.handleNetworks(w, r)
		return
	}
	renderError(r.Context(), w, APIErrBadRequest)
}

func (s Server) newNodeStatus(node *Node, vm *NodeVM) *NodeStatus {
	status := &NodeStatus{
		Name:      node.Name,
		Taps:      node.taps,
		CPU:       node.CPU,
		Memory:    node.Memory,
		UEFI:      node.UEFI,
		SMBIOS:    node.SMBIOS,
		IsRunning: vm.IsRunning(),
	}
	if !s.runtime.graphic {
		status.SocketPath = s.runtime.socketPath(node.Name)
	}
	status.Volumes = make([]string, len(node.Volumes))
	for i, v := range node.Volumes {
		status.Volumes[i] = v.Name
	}
	return status
}

func (s Server) newPodStatus(pod *Pod) *PodStatus {
	status := &PodStatus{
		Name:  pod.Name,
		Veths: pod.veths,
		UUID:  pod.uuid,
	}
	status.Volumes = make([]string, len(pod.Volumes))
	for i, v := range pod.Volumes {
		status.Volumes[i] = v.Name
	}
	status.Apps = make([]string, len(pod.Apps))
	for i, v := range pod.Apps {
		status.Apps[i] = v.Name
	}
	return status
}

func splitParams(path string) []string {
	paths := strings.Split(path, "/")
	var params []string
	for _, str := range paths {
		if str != "" {
			params = append(params, str)
		}
	}
	return params
}

func (s Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	params := splitParams(r.URL.Path)
	if r.Method == "GET" && len(params) == 1 {
		statuses := make([]*NodeStatus, len(s.cluster.Nodes))
		for i, node := range s.cluster.Nodes {
			statuses[i] = s.newNodeStatus(node, s.vms[node.SMBIOS.Serial])
		}
		renderJSON(w, statuses, http.StatusOK)
	} else if r.Method == "GET" && len(params) == 2 {
		if node, ok := s.cluster.nodeMap[params[1]]; ok {
			status := s.newNodeStatus(node, s.vms[node.SMBIOS.Serial])
			renderJSON(w, status, http.StatusOK)
		} else {
			renderError(r.Context(), w, APIErrNotFound)
		}
	} else if r.Method == "POST" && len(params) == 3 {
		if node, ok := s.cluster.nodeMap[params[1]]; ok {
			switch params[2] {
			case "start":
				s.vms[node.SMBIOS.Serial].PowerOn()
			case "stop":
				s.vms[node.SMBIOS.Serial].PowerOff()
			case "restart":
				s.vms[node.SMBIOS.Serial].PowerOff()
				s.vms[node.SMBIOS.Serial].PowerOn()
			default:
				renderError(r.Context(), w, APIErrBadRequest)
				return
			}
			renderJSON(w, s.vms[node.SMBIOS.Serial].IsRunning(), http.StatusOK)
		} else {
			renderError(r.Context(), w, APIErrNotFound)
		}
	} else {
		renderError(r.Context(), w, APIErrBadRequest)
	}
}

func (s Server) handlePods(w http.ResponseWriter, r *http.Request) {
	params := splitParams(r.URL.Path)
	if r.Method == "GET" && len(params) == 1 {
		statuses := make([]*PodStatus, len(s.cluster.Pods))
		for i, pod := range s.cluster.Pods {
			statuses[i] = s.newPodStatus(pod)
		}
		renderJSON(w, statuses, http.StatusOK)
	} else if r.Method == "GET" && len(params) == 2 {
		if pod, ok := s.cluster.podMap[params[1]]; ok {
			status := s.newPodStatus(pod)
			renderJSON(w, status, http.StatusOK)
		} else {
			renderError(r.Context(), w, APIErrNotFound)
		}
		/* not working
		} else if r.Method == "POST" && len(params) == 3 {
			if pod, ok := s.cluster.podMap[params[1]]; ok {
				var cmds [][]string
				switch params[2] {
				case "start":
					cmds = append(cmds, []string{"ip", "netns", "exec", "pm_" + pod.Name, "rkt", "start", pod.uuid})
				case "stop":
					cmds = append(cmds, []string{"ip", "netns", "exec", "pm_" + pod.Name, "rkt", "stop", pod.uuid})
				case "restart":
				default:
					renderError(r.Context(), w, APIErrBadRequest)
					return
				}
				err := execCommands(r.Context(), cmds)
				if err != nil {
					renderError(r.Context(), w, InternalServerError(err))
				} else {
					renderJSON(w, "ok", http.StatusOK)
				}
			} else {
				renderError(r.Context(), w, APIErrNotFound)
			}
		*/
	} else {
		renderError(r.Context(), w, APIErrBadRequest)
	}
}

func (s Server) handleNetworks(w http.ResponseWriter, r *http.Request) {
	params := splitParams(r.URL.Path)

	if r.Method == "POST" && len(params) == 3 {
		var cmds [][]string
		switch params[2] {
		case "up":
			cmds = append(cmds, []string{"ip", "link", "set", "dev", params[1], "up"})
		case "down":
			cmds = append(cmds, []string{"ip", "link", "set", "dev", params[1], "down"})
		case "delay":
			cmds = append(cmds, []string{"tc", "qdisc", "add", "dev", params[1], "root", "netem", "delay", "100ms"})
		case "loss":
			cmds = append(cmds, []string{"tc", "qdisc", "add", "dev", params[1], "root", "netem", "loss", "50%"})
		case "clear":
			cmds = append(cmds, []string{"tc", "qdisc", "del", "dev", params[1], "root"})
		default:
			renderError(r.Context(), w, APIErrBadRequest)
			return
		}

		err := execCommands(r.Context(), cmds)
		if err != nil {
			renderError(r.Context(), w, InternalServerError(err))
		} else {
			renderJSON(w, "ok", http.StatusOK)
		}
	} else {
		renderError(r.Context(), w, APIErrBadRequest)
	}
}
