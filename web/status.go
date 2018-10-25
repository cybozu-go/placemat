package web

type NodeStatus struct {
	Name    string            `json:"name"`
	Taps    map[string]string `json:"taps"`
	Volumes []string          `json:"volumes"`
	CPU     int               `json:"cpu"`
	Memory  string            `json:"memory"`
	UEFI    bool              `json:"uefi"`
	//SMBIOS     SMBIOSConfig      `json:"smbios"`
	IsRunning  bool   `json:"is_running"`
	SocketPath string `json:"socket_path"`
}

type PodStatus struct {
	Name    string            `json:"name"`
	UUID    string            `json:"uuid"`
	Veths   map[string]string `json:"veths"`
	Volumes []string          `json:"volumes"`
	Apps    []string          `json:"apps"`
}
