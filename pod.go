package placemat

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PodInterfaceSpec represents a Pod's Interface definition in YAML
type PodInterfaceSpec struct {
	Network   string   `yaml:"network"`
	Addresses []string `yaml:"addresses,omitempty"`
}

// PodVolumeSpec represents a Pod's Volume definition in YAML
type PodVolumeSpec struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Folder   string `yaml:"folder,omitempty"`
	ReadOnly bool   `yaml:"readonly"`
	Mode     string `yaml:"mode,omitempty"`
	UID      string `yaml:"uid,omitempty"`
	GID      string `yaml:"gid,omitempty"`
}

// PodAppMountSpec represents a App's Mount definition in YAML
type PodAppMountSpec struct {
	Volume string `yaml:"volume"`
	Target string `yaml:"target"`
}

// PodAppSpec represents a Pod's App definition in YAML
type PodAppSpec struct {
	Name           string            `yaml:"name"`
	Image          string            `yaml:"image"`
	ReadOnlyRootfs bool              `yaml:"readonly-rootfs"`
	User           string            `yaml:"user,omitempty"`
	Group          string            `yaml:"group,omitempty"`
	Exec           string            `yaml:"exec,omitempty"`
	Args           []string          `yaml:"args,omitempty"`
	Env            map[string]string `yaml:"env,omitempty"`
	CapsRetain     []string          `yaml:"caps-retain,omitempty"`
	Mount          []PodAppMountSpec `yaml:"mount,omitempty"`
}

// PodSpec represents a Pod specification in YAML
type PodSpec struct {
	Name        string             `yaml:"name"`
	InitScripts []string           `yaml:"init-scripts,omitempty"`
	Interfaces  []PodInterfaceSpec `yaml:"interfaces,omitempty"`
	Volumes     []*PodVolumeSpec   `yaml:"volumes,omitempty"`
	Apps        []*PodAppSpec      `yaml:"apps"`
}

func NewPod(spec *PodSpec) (*Pod, error) {
	p = &Pod{
		PodSpec: spec,
	}

	if len(spec.Name) == 0 {
		return nil, errors.New("pod name is empty")
	}

	for _, script := range spec.InitScripts {
		script, err = filepath.Abs(script)
		if err != nil {
			return nil, err
		}
		_, err := os.Stat(script)
		if err != nil {
			return nil, err
		}
		pod.initScripts = append(pod.initScripts, script)
	}

	for _, vs := range spec.Volumes {
		vol, err := NewPodVolume(vs)
		if err != nil {
			return nil, err
		}
		p.volumes = append(p.volumes, vol)
	}

	if len(spec.Apps) == 0 {
		return nil, errors.New("no app for pod " + spec.Name)
	}

	return p, nil
}

// PodVolume is an interface of a volume for Pod.
type PodVolume interface {
	// Name returns the volume name.
	Name() string
	// Resolve resolves references in the volume definition.
	Resolve(*Cluster) error
	// Spec returns a command-line argument for the volume.
	Spec() string
}

// NewPodVolume makes a PodVolume, or returns an error.
func NewPodVolume(spec *PodVolumeSpec) (PodVolume, error) {
	if len(spec.Name) == 0 {
		return nil, errors.New("invalid pod volume name")
	}
	switch spec.Kind {
	case "host":
		return newHostPodVolume(spec.Name, spec.Folder, spec.ReadOnly), nil
	case "empty":
		return newEmptyPodVolume(spec.Name, spec.Mode, spec.UID, spec.GID), nil
	}

	return nil, errors.New("invalid kind of pod volume: " + spec.Kind)
}

type hostPodVolume struct {
	name       string
	folderName string
	folder     *DataFolder
	readOnly   bool
}

func (v *hostPodVolume) Name() string {
	return v.name
}

func (v *hostPodVolume) Resolve(c *Cluster) error {
	for _, df := range c.DataFolders {
		if v.folderName == df.Name {
			v.folder = df
			return nil
		}
	}
	return errors.New("folder is not found:" + v.folderName)
}

func (v *hostPodVolume) Spec() string {
	return fmt.Sprintf("%s,kind=host,source=%s,readOnly=%v", v.name, v.folder.Path(), v.readOnly)
}

func newHostPodVolume(name, folder string, readOnly bool) PodVolume {
	return &hostPodVolume{name, folder, nil, readOnly}
}

type emptyPodVolume struct {
	name string
	mode string
	uid  string
	gid  string
}

func (v *emptyPodVolume) Name() string {
	return v.name
}

func (v *emptyPodVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *emptyPodVolume) Spec() string {
	buf := make([]byte, 0, 32)
	buf = append(buf, v.name...)
	buf = append(buf, ",kind=empty,readOnly=false"...)
	if len(v.mode) > 0 {
		buf = append(buf, ",mode="...)
		buf = append(buf, v.mode...)
	}
	if len(v.uid) > 0 {
		buf = append(buf, ",uid="...)
		buf = append(buf, v.uid...)
	}
	if len(v.gid) > 0 {
		buf = append(buf, ",gid="...)
		buf = append(buf, v.gid...)
	}
	return string(buf)
}

func newEmptyPodVolume(name, mode, uid, gid string) PodVolume {
	return &emptyPodVolume{name, mode, uid, gid}
}

func (a *PodAppSpec) appendParams(params []string, podname string) []string {
	params = append(params, []string{
		a.Image,
		"--name", a.Name,
		"--user-label", "name=" + podname,
	}...)
	if a.ReadOnlyRootfs {
		params = append(params, "--readonly-rootfs=true")
	}
	if len(a.User) > 0 {
		params = append(params, "--user="+a.User)
	}
	if len(a.Group) > 0 {
		params = append(params, "--group="+a.Group)
	}
	if len(a.Exec) > 0 {
		params = append(params, []string{"--exec", a.Exec}...)
	}
	for k, v := range a.Env {
		params = append(params, fmt.Sprintf("--set-env=%s=%s", k, v))
	}
	if len(a.CapsRetain) > 0 {
		params = append(params, "--caps-retain="+strings.Join(a.CapsRetain, ","))
	}
	for _, mp := range a.Mount {
		t := fmt.Sprintf("volume=%s,target=%s", mp.Volume, mp.Target)
		params = append(params, []string{"--mount", t}...)
	}
	if len(a.Args) > 0 {
		params = append(params, "--")
		params = append(params, a.Args...)
	}

	return params
}

// Pod represents a pod resource.
type Pod struct {
	*PodSpec
	initScripts []string
	volumes     []PodVolume
}

func (p *Pod) resolve(c *Cluster) error {
	nm := make(map[string]*Network)
	for _, n := range c.Networks {
		nm[n.Name] = n
	}

	for i := range p.Interfaces {
		nn := p.Interfaces[i].NetworkName
		if _, ok := nm[nn]; !ok {
			return errors.New("no such network: " + nn)
		}
	}

	for _, v := range p.Volumes {
		err := v.Resolve(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pod) appendParams(params []string) []string {
	params = append(params, []string{"--hostname", p.Name}...)
	for _, v := range p.Volumes {
		params = append(params, []string{"--volume", v.Spec()}...)
	}

	addDDD := false
	for _, a := range p.Apps {
		if addDDD {
			params = append(params, "---")
		}
		params = a.appendParams(params, p.Name)
		addDDD = len(a.Args) > 0
	}
	return params
}
