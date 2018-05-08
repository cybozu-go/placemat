package placemat

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/log"
)

func umount(mp string) error {
	return exec.Command("umount", mp).Run()
}

func bindMount(src, dest string) error {
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}
	log.Info("bind mount", map[string]interface{}{
		"src":  src,
		"dest": dest,
	})
	c := exec.Command("mount", "--bind", src, dest)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return c.Run()
}

func mount(fs, dest, options string) error {
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}
	log.Info("mount", map[string]interface{}{
		"fs":   fs,
		"dest": dest,
	})
	c := exec.Command("mount", "-t", fs, "-o", options, fs, dest)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return c.Run()
}

// Rootfs is a fake root filesystem in order to fool rkt
// into believing that the system is running without systemd
// by hiding /run/systemd/system directory.
type Rootfs struct {
	root        string
	mountPoints []string
}

// Path returns the absolute filesystem path to the fake rootfs.
func (r *Rootfs) Path() string {
	return r.root
}

// Destroy unmounts the root filesystem and remove the mount point directory.
func (r *Rootfs) Destroy() error {
	var err error

	l := len(r.mountPoints)
	for i := 0; i < l; i++ {
		e := umount(r.mountPoints[l-i-1])
		if e != nil {
			err = e
		}
	}

	if err != nil {
		log.Error("failed to umount", map[string]interface{}{
			"root":      r.root,
			log.FnError: err,
		})
		return err
	}

	return os.RemoveAll(r.root)
}

// NewRootfs creates a new root filesystem.
func NewRootfs() (*Rootfs, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	root, err := ioutil.TempDir("/", "placemat-root")
	if err != nil {
		return nil, err
	}

	err = bindMount("/", root)
	if err != nil {
		return nil, err
	}

	mountPoints := []string{root}
	defer func() {
		l := len(mountPoints)
		for i := 0; i < l; i++ {
			umount(mountPoints[l-i-1])
		}
	}()

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		mp := fields[1]
		fs := fields[2]
		options := fields[3]
		opts := strings.Split(options, ",")
		dest := filepath.Join(root, mp)

		switch {
		case mp == "/":
			continue
		case strings.HasPrefix(mp, "/placemat-root"):
			continue
		case strings.HasPrefix(mp, "/boot"):
			continue
		}

		readonly := false
		for _, opt := range opts {
			if opt == "ro" {
				readonly = true
				break
			}
		}
		if fs == "tmpfs" && readonly {
			for i := range opts {
				if opts[i] == "ro" {
					opts[i] = "rw"
					break
				}
			}
			options = strings.Join(opts, ",")
		}

		switch fs {
		case "tmpfs", "proc", "sysfs", "securityfs", "cgroup", "debugfs", "fusectl", "configfs":
			err = mount(fs, dest, options)
			if err != nil {
				return nil, err
			}
			mountPoints = append(mountPoints, dest)

		case "autofs", "pstore", "efivarfs", "fuse.lxcfs":
			// ignore

		default:
			err = bindMount(mp, dest)
			if err != nil {
				return nil, err
			}
			mountPoints = append(mountPoints, dest)
		}
	}

	ret := &Rootfs{root, mountPoints}
	mountPoints = nil
	return ret, nil
}
