package placemat

import (
	"errors"
	"fmt"
	"path/filepath"
)

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
func NewPodVolume(name, kind, folder, mode, uid, gid string, readOnly bool) (PodVolume, error) {
	if len(name) == 0 {
		return nil, errors.New("invalid pod volume name")
	}
	switch kind {
	case "host":
		return newHostPodVolume(name, folder, readOnly), nil
	case "empty":
		return newEmptyPodVolume(name, mode, uid, gid), nil
	}

	return nil, errors.New("invalid kind of pod volume: " + kind)
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
	return fmt.Sprintf("%s,kind=host,folder=%s,readOnly=%v", v.name, v.folderName, v.readOnly)
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
	buf = append(buf, ",kind=empty"...)
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

// PodApp represents an app for Pod.
type PodApp struct {
	Name           string
	Image          string
	ReadOnlyRootfs bool
	User           string
	Group          string
	Exec           string
	Args           []string
	Env            map[string]string
	CapsRetain     []string
	MountPoints    []struct {
		VolumeName string
		Target     string
		volume     PodVolume
	}
}

// AddMountPoint adds a mount point to app.
func (a *PodApp) AddMountPoint(volname, target string) error {
	if len(volname) == 0 {
		return errors.New("empty volume for app " + a.Name)
	}

	if !filepath.IsAbs(target) {
		return errors.New("bad mount target for app " + a.Name)
	}

	a.MountPoints = append(a.MountPoints, struct {
		VolumeName string
		Target     string
		volume     PodVolume
	}{
		VolumeName: volname,
		Target:     target,
	})
	return nil
}

func (a *PodApp) resolve(vm map[string]PodVolume) error {
	for i := range a.MountPoints {
		vn := a.MountPoints[i].VolumeName
		v := vm[vn]
		if v == nil {
			return errors.New("no such pod volume: " + vn)
		}
		a.MountPoints[i].volume = v
	}
	return nil
}

// Pod represents a pod resource.
type Pod struct {
	Name        string
	InitScripts []string
	Interfaces  []struct {
		NetworkName string
		Addresses   []string
	}
	Volumes []PodVolume
	Apps    []*PodApp
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

	vm := make(map[string]PodVolume)
	for _, v := range p.Volumes {
		err := v.Resolve(c)
		if err != nil {
			return err
		}
		vm[v.Name()] = v
	}

	for _, a := range p.Apps {
		err := a.resolve(vm)
		if err != nil {
			return err
		}
	}

	return nil
}
