package vm

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cybozu-go/placemat/v2/pkg/types"
)

const (
	defaultOVMFCodePath  = "/usr/share/OVMF/OVMF_CODE.fd"
	defaultOVMFVarsPath  = "/usr/share/OVMF/OVMF_VARS.fd"
	defaultRebootTimeout = 30 * time.Second
)

type Qemu struct {
	name         string
	taps         []*TapInfo
	volumes      []VolumeArgs
	ignitionFile string
	cpu          int
	memory       string
	uefi         bool
	tpm          bool
	smbios       SMBIOSConfig
	MACGenerator
}

func NewQemu(nodeName string, taps []*TapInfo, volumes []VolumeArgs, ignitionFile string, cpu int,
	memory string, uefi bool, tpm bool, smbios SMBIOSConfig) *Qemu {
	return &Qemu{
		name:         nodeName,
		taps:         taps,
		volumes:      volumes,
		ignitionFile: ignitionFile,
		cpu:          cpu,
		memory:       memory,
		uefi:         uefi,
		tpm:          tpm,
		smbios:       smbios,
		MACGenerator: &MACGeneratorForKVM{},
	}
}

func (c *Qemu) Command(r *Runtime) []string {
	params := c.qemuParams(r)

	for _, v := range c.volumes {
		params = append(params, v.Args()...)
	}

	for _, t := range c.taps {
		netdev := fmt.Sprintf("tap,id=%s,ifname=%s,script=no,downscript=no", t.bridge, t.tap)
		if vhostNetSupported {
			netdev += ",vhost=on"
		}

		params = append(params, "-netdev", netdev)

		devParams := []string{
			"virtio-net-pci",
			fmt.Sprintf("host_mtu=%d", t.mtu),
			fmt.Sprintf("netdev=%s", t.bridge),
			fmt.Sprintf("mac=%s", c.Generate()),
		}
		if c.uefi {
			// disable iPXE boot
			devParams = append(devParams, "romfile=")
		}
		params = append(params, "-device", strings.Join(devParams, ","))
	}

	if c.tpm {
		params = append(params, "-chardev", fmt.Sprintf("socket,id=chrtpm,path=%s", r.swtpmSocketPath(c.name)))
		params = append(params, "-tpmdev", "emulator,id=tpm0,chardev=chrtpm")
		params = append(params, "-device", "tpm-tis,tpmdev=tpm0")
	}

	params = append(params, "-boot", fmt.Sprintf("reboot-timeout=%d", int64(defaultRebootTimeout/time.Millisecond)))

	guest := r.guestSocketPath(c.name)
	params = append(params, "-chardev", fmt.Sprintf("socket,id=char0,path=%s,server,nowait", guest))
	params = append(params, "-device", "virtio-serial")
	params = append(params, "-device", "virtserialport,chardev=char0,name=placemat")

	monitor := r.monitorSocketPath(c.name)
	params = append(params, "-monitor", fmt.Sprintf("unix:%s,server,nowait", monitor))

	// Random generator passthrough for fast boot
	params = append(params, "-object", "rng-random,id=rng0,filename=/dev/urandom")
	params = append(params, "-device", "virtio-rng-pci,rng=rng0")

	// Use host CPU flags for stability
	params = append(params, "-cpu", "host")

	return append([]string{"qemu-system-x86_64"}, params...)
}

func (c *Qemu) qemuParams(r *Runtime) []string {
	params := []string{"-enable-kvm"}

	if c.ignitionFile != "" {
		params = append(params, "-fw_cfg")
		params = append(params, fmt.Sprintf("opt/com.coreos/config,file=%s", c.ignitionFile))
	}

	if c.cpu != 0 {
		params = append(params, "-smp", strconv.Itoa(c.cpu))
	}
	if c.memory != "" {
		params = append(params, "-m", c.memory)
	}
	if !r.graphic {
		p := r.socketPath(c.name)
		params = append(params, "-nographic")
		params = append(params, "-serial", fmt.Sprintf("unix:%s,server,nowait", p))
	}
	if c.uefi {
		p := r.nvramPath(c.name)
		params = append(params, "-drive", fmt.Sprintf("if=pflash,file=%s,format=raw,readonly", defaultOVMFCodePath))
		params = append(params, "-drive", fmt.Sprintf("if=pflash,file=%s,format=raw", p))
	}

	smbios := "type=1"
	if c.smbios.manufacturer != "" {
		smbios += ",manufacturer=" + c.smbios.manufacturer
	}
	if c.smbios.product != "" {
		smbios += ",product=" + c.smbios.product
	}
	if c.smbios.serial == "" {
		c.smbios.serial = fmt.Sprintf("%x", sha1.Sum([]byte(c.name)))
	}
	smbios += ",serial=" + c.smbios.serial
	params = append(params, "-smbios", smbios)
	return params
}

type MACGenerator interface {
	Generate() string
}

type MACGeneratorForKVM struct {
}

func (m *MACGeneratorForKVM) Generate() string {
	vendorPrefix := "52:54:00" // QEMU's vendor prefix
	buf := make([]byte, 3)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s:%02x:%02x:%02x", vendorPrefix, buf[0], buf[1], buf[2])
}

type VolumeArgs interface {
	Args() []string
}

type ImageVolumeArgs struct {
	volumePath string
	cache      string
}

func (v *ImageVolumeArgs) Args() []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=%s,file=%s", v.cache, selectAIOforCache(v.cache), v.volumePath),
	}
}

func selectAIOforCache(cache string) string {
	if cache == types.NodeVolumeCacheNone {
		return "native"
	}
	return "threads"
}

type LocalDSVolumeArgs struct {
	volumePath string
	cache      string
}

func (v *LocalDSVolumeArgs) Args() []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=%s,format=raw,file=%s", v.cache, selectAIOforCache(v.cache), v.volumePath),
	}
}

type RawVolumeArgs struct {
	volumePath string
	cache      string
	format     string
}

func (v *RawVolumeArgs) Args() []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=%s,format=%s,file=%s", v.cache, selectAIOforCache(v.cache), v.format, v.volumePath),
	}
}

type LVVolumeArgs struct {
	volumePath string
	cache      string
}

func (v *LVVolumeArgs) Args() []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=%s,format=raw,file=%s", v.cache, selectAIOforCache(v.cache), v.volumePath),
	}
}

type Qemu9pVolumeArgs struct {
	volumePath string
	cache      string
	writable   bool
	mountTag   string
}

func (v *Qemu9pVolumeArgs) Args() []string {
	readonly := ""
	if !v.writable {
		readonly = ",readonly"
	}
	return []string{
		"-virtfs",
		fmt.Sprintf("local,path=%s,mount_tag=%s,security_model=none%s", v.volumePath, v.mountTag, readonly),
	}
}
