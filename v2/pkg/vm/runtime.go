package vm

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/util"
)

var vhostNetSupported bool

// LoadModules loads vhost-net kernel module
func LoadModules() {
	err := exec.Command("modprobe", "vhost-net").Run()
	if err != nil {
		log.Error("failed to modprobe vhost-net", map[string]interface{}{
			"error": err,
		})
	}

	f, err := os.Open("/proc/modules")
	if err != nil {
		log.Error("failed to open /proc/modules", map[string]interface{}{
			"error": err,
		})
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.Contains(s.Text(), "vhost_net") {
			vhostNetSupported = true
			return
		}
	}
}

// Runtime contains the runtime information to run Cluster.
type Runtime struct {
	Force      bool
	Graphic    bool
	RunDir     string
	DataDir    string
	TempDir    string
	ListenAddr string
	ImageCache *util.Cache
}

// NewRuntime initializes a new Runtime.
func NewRuntime(force, graphic bool, runDir, dataDir, cacheDir, listenAddr string) (*Runtime, error) {
	r := &Runtime{
		Force:      force,
		Graphic:    graphic,
		RunDir:     runDir,
		DataDir:    dataDir,
		ListenAddr: listenAddr,
	}

	fi, err := os.Stat(cacheDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			return nil, errors.New(cacheDir + " is not a directory")
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	imageCacheDir := filepath.Join(cacheDir, "image_cache")
	err = os.MkdirAll(imageCacheDir, 0755)
	if err != nil {
		return nil, err
	}

	r.ImageCache = util.NewCache(imageCacheDir)

	fi, err = os.Stat(dataDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			return nil, errors.New(dataDir + " is not a directory")
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	volumeDir := filepath.Join(dataDir, "volumes")
	err = os.MkdirAll(volumeDir, 0755)
	if err != nil {
		return nil, err
	}

	nvramDir := filepath.Join(dataDir, "nvram")
	err = os.MkdirAll(nvramDir, 0755)
	if err != nil {
		return nil, err
	}

	tempDir := filepath.Join(dataDir, "temp")
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		return nil, err
	}
	myTempDir, err := ioutil.TempDir(tempDir, "")
	if err != nil {
		return nil, err
	}
	r.TempDir = myTempDir

	return r, nil
}

func (r *Runtime) socketPath(host string) string {
	return filepath.Join(r.RunDir, host+".socket")
}

func (r *Runtime) qmpSocketPath(host string) string {
	return filepath.Join(r.RunDir, host+".qmp")
}

func (r *Runtime) guestSocketPath(host string) string {
	return filepath.Join(r.RunDir, host+".guest")
}

func (r *Runtime) nvramPath(host string) string {
	return filepath.Join(r.DataDir, "nvram", host+".fd")
}

func (r *Runtime) swtpmSocketDirPath(host string) string {
	return filepath.Join(r.RunDir, host)
}

func (r *Runtime) swtpmSocketPath(host string) string {
	return filepath.Join(r.RunDir, host, "swtpm.socket")
}
