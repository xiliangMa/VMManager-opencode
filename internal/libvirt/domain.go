package libvirt

import (
	"fmt"
	"strings"
	"time"

	"libvirt.org/go/libvirt"
)

type DomainManager struct {
	client *Client
}

func NewDomainManager(client *Client) *DomainManager {
	return &DomainManager{client: client}
}

func (dm *DomainManager) CreateVM(config VMConfig) (*libvirt.Domain, error) {
	xml := dm.generateDomainXML(config)

	dm.client.mu.Lock()
	defer dm.client.mu.Unlock()

	domain, err := dm.client.conn.DomainDefineXML(xml)
	if err != nil {
		return nil, fmt.Errorf("failed to define domain: %w", err)
	}

	if err := domain.Create(); err != nil {
		domain.Free()
		return nil, fmt.Errorf("failed to start domain: %w", err)
	}

	return domain, nil
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

func (dm *DomainManager) Start(domain *libvirt.Domain) error {
	return domain.Create()
}

func (dm *DomainManager) Stop(domain *libvirt.Domain) error {
	return domain.Shutdown()
}

func (dm *DomainManager) ForceStop(domain *libvirt.Domain) error {
	return domain.Destroy()
}

func (dm *DomainManager) Reboot(domain *libvirt.Domain) error {
	return domain.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT)
}

func (dm *DomainManager) Suspend(domain *libvirt.Domain) error {
	return domain.Suspend()
}

func (dm *DomainManager) Resume(domain *libvirt.Domain) error {
	return domain.Resume()
}

func (dm *DomainManager) Delete(domain *libvirt.Domain) error {
	if err := domain.Destroy(); err != nil {
		return fmt.Errorf("failed to destroy domain: %w", err)
	}
	return domain.Undefine()
}

func (dm *DomainManager) GetState(domain *libvirt.Domain) (libvirt.DomainState, error) {
	state, _, err := domain.GetState()
	return state, err
}

func (dm *DomainManager) GetInfo(domain *libvirt.Domain) (*DomainInfo, error) {
	info, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	return &DomainInfo{
		CPU:        int(info.NrVirtCpu),
		Memory:     int(info.MaxMem / 1024),
		UsedMemory: int(info.Memory / 1024),
	}, nil
}

func (dm *DomainManager) GetXMLDesc(domain *libvirt.Domain) (string, error) {
	return domain.GetXMLDesc(0)
}

func (dm *DomainManager) GetStats(domain *libvirt.Domain) (*DomainStats, error) {
	stats, err := domain.GetStats(0)
	if err != nil {
		return nil, err
	}

	return &DomainStats{
		CPUTime:   stats.CpuTime,
		MaxMem:    stats.MaxMem,
		UsedMem:   stats.MemUsed,
		NrVirtCpu: stats.NrVirtCpu,
		State:     stats.State,
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

type DomainStats struct {
	CPUTime   uint64
	MaxMem    uint64
	UsedMem   uint64
	NrVirtCpu uint64
	State     libvirt.DomainState
}

func GetDomainStateString(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_NOSTATE:
		return "nostate"
	case libvirt.DOMAIN_RUNNING:
		return "running"
	case libvirt.DOMAIN_PAUSED:
		return "paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "shutdown"
	case libvirt.DOMAIN_CRASHED:
		return "crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "suspended"
	case libvirt.DOMAIN_SHUTOFF:
		return "shutoff"
	default:
		return "unknown"
	}
}

func (dm *DomainManager) WaitForState(domain *libvirt.Domain, desiredState libvirt.DomainState, timeout time.Duration) error {
	states := make(chan libvirt.DomainState, 1)

	go func() {
		for {
			state, _, err := domain.GetState()
			if err != nil {
				states <- libvirt.DOMAIN_NOSTATE
				return
			}
			states <- state
			time.Sleep(500 * time.Millisecond)
		}
	}()

	select {
	case state := <-states:
		if state == desiredState {
			return nil
		}
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for desired state")
	}

	return nil
}
