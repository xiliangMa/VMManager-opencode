//go:build !linux || mock
// +build !linux mock

package libvirt

import (
	"fmt"
	"strings"
	"time"
)

type DomainManager struct {
	client *Client
}

func NewDomainManager(client *Client) *DomainManager {
	return &DomainManager{client: client}
}

func (dm *DomainManager) CreateVM(config VMConfig) (*MockDomain, error) {
	return &MockDomain{
		Name:    config.Name,
		UUID:    config.UUID,
		State:   1,
		CPUTime: 0,
		MaxMem:  uint64(config.Memory * 1024),
		MemUsed: 0,
	}, nil
}

func (dm *DomainManager) generateDomainXML(config VMConfig) string {
	var xml strings.Builder

	xml.WriteString(fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit='MiB'>%d</memory>
  <vcpu placement='static' current='%d'>%d</vcpu>
  <os>
    <type arch='%s' machine='pc'>hvm</type>
    <boot dev='hd'/>
  </os>
`, config.Name, config.UUID, config.Memory, config.VCPU, config.VCPU, config.Architecture))

	for i, disk := range config.Disks {
		xml.WriteString(fmt.Sprintf(`
  <disk type='file' device='disk'>
    <driver name='qemu' type='%s'/>
    <source file='%s'/>
    <target dev='vd%c' bus='virtio'/>
  </disk>
`, disk.Format, disk.Path, 'a'+rune(i)))
	}

	for _, net := range config.Networks {
		xml.WriteString(fmt.Sprintf(`
  <interface type='bridge'>
    <mac address='%s'/>
    <source bridge='%s'/>
    <model type='virtio'/>
  </interface>
`, net.MACAddress, net.Bridge))
	}

	xml.WriteString(`
  <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'>
    <listen type='address' address='0.0.0.0'/>
  </graphics>
`)

	xml.WriteString(`</domain>`)

	return xml.String()
}

func (dm *DomainManager) Start(domain *MockDomain) error {
	domain.State = 1
	return nil
}

func (dm *DomainManager) Stop(domain *MockDomain) error {
	domain.State = 0
	return nil
}

func (dm *DomainManager) ForceStop(domain *MockDomain) error {
	domain.State = 0
	return nil
}

func (dm *DomainManager) Reboot(domain *MockDomain) error {
	domain.State = 1
	return nil
}

func (dm *DomainManager) Suspend(domain *MockDomain) error {
	domain.State = 3
	return nil
}

func (dm *DomainManager) Resume(domain *MockDomain) error {
	domain.State = 1
	return nil
}

func (dm *DomainManager) Delete(domain *MockDomain) error {
	return nil
}

func (dm *DomainManager) GetState(domain *MockDomain) (int, error) {
	return domain.State, nil
}

func (dm *DomainManager) GetInfo(domain *MockDomain) (*DomainInfo, error) {
	return &DomainInfo{
		CPU:        4,
		Memory:     int(domain.MaxMem / 1024),
		UsedMemory: int(domain.MemUsed / 1024),
	}, nil
}

func (dm *DomainManager) GetXMLDesc(domain *MockDomain) (string, error) {
	return "<domain><name>" + domain.Name + "</name></domain>", nil
}

func (dm *DomainManager) GetStats(domain *MockDomain) (*DomainStats, error) {
	return &DomainStats{
		CPUTime:     int64(domain.CPUTime),
		MemoryUsage: int64(domain.MemUsed),
		MemoryTotal: int64(domain.MaxMem),
	}, nil
}

type VMConfig struct {
	ID           string
	Name         string
	UUID         string
	Architecture string
	VCPU         int
	Memory       int
	Disks        []DiskConfig
	Networks     []NetworkConfig
}

type DiskConfig struct {
	Path   string
	Format string
	Size   int
}

type NetworkConfig struct {
	MACAddress string
	Bridge     string
}

type DomainInfo struct {
	CPU        int
	Memory     int
	UsedMemory int
}

func GetDomainStateString(state int) string {
	switch state {
	case 0:
		return "nostate"
	case 1:
		return "running"
	case 2:
		return "paused"
	case 3:
		return "shutdown"
	case 4:
		return "crashed"
	case 5:
		return "suspended"
	case 6:
		return "shutoff"
	default:
		return "unknown"
	}
}

func (dm *DomainManager) WaitForState(domain *MockDomain, desiredState int, timeout time.Duration) error {
	return nil
}
