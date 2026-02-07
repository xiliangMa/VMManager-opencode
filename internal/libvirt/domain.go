package libvirt

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

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

type DomainManager struct {
	client *Client
}

func NewDomainManager(client *Client) *DomainManager {
	return &DomainManager{client: client}
}

func (dm *DomainManager) CreateVM(config VMConfig) (*MockDomain, error) {
	domain := &MockDomain{
		Name:         config.Name,
		UUID:         config.UUID,
		State:        0,
		CPUTime:      0,
		MaxMem:       uint64(config.Memory * 1024),
		MemUsed:      0,
		DiskPath:     config.Disks[0].Path,
		VNCPort:      0,
		SnapshotList: make([]*MockSnapshot, 0),
		Autostart:    false,
	}

	if dm.client != nil {
		dm.client.mu.Lock()
		dm.client.Domains[config.Name] = domain
		dm.client.mu.Unlock()
	}

	return domain, nil
}

func (dm *DomainManager) generateDomainXML(config VMConfig) string {
	var xml strings.Builder

	xml.WriteString(fmt.Sprintf(`<domain type='kvm'>
 </name>
  <name>%s <uuid>%s</uuid>
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
	if dm.client != nil {
		dm.client.mu.Lock()
		delete(dm.client.Domains, domain.Name)
		dm.client.mu.Unlock()
	}
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
	return domain.GetXMLDesc(0)
}

func (dm *DomainManager) GetStats(domain *MockDomain) (*DomainStats, error) {
	return &DomainStats{
		State:       domain.State,
		CPUTime:     int64(domain.CPUTime),
		MemoryUsage: int64(domain.MemUsed),
		MemoryTotal: int64(domain.MaxMem),
		DiskRead:    1024,
		DiskWrite:   2048,
		NetworkRX:   10240,
		NetworkTX:   20480,
	}, nil
}

func (dm *DomainManager) CreateSnapshot(domain *MockDomain, name, description string) (*SnapshotInfo, error) {
	snapshot := &MockSnapshot{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		State:       domain.State,
		MemoryFile:  fmt.Sprintf("/var/lib/libvirt/images/%s-%s.mem", domain.UUID, name),
	}

	domain.SnapshotList = append(domain.SnapshotList, snapshot)

	return &SnapshotInfo{
		Name:        snapshot.Name,
		Description: snapshot.Description,
		CreatedAt:   snapshot.CreatedAt,
		State:       GetDomainStateString(snapshot.State),
	}, nil
}

func (dm *DomainManager) ListSnapshots(domain *MockDomain) ([]SnapshotInfo, error) {
	snapshots := make([]SnapshotInfo, 0, len(domain.SnapshotList))
	for _, s := range domain.SnapshotList {
		snapshots = append(snapshots, SnapshotInfo{
			Name:        s.Name,
			Description: s.Description,
			CreatedAt:   s.CreatedAt,
			State:       GetDomainStateString(s.State),
		})
	}
	return snapshots, nil
}

func (dm *DomainManager) GetSnapshot(domain *MockDomain, name string) (*SnapshotInfo, error) {
	for _, s := range domain.SnapshotList {
		if s.Name == name {
			return &SnapshotInfo{
				Name:        s.Name,
				Description: s.Description,
				CreatedAt:   s.CreatedAt,
				State:       GetDomainStateString(s.State),
			}, nil
		}
	}
	return nil, fmt.Errorf("snapshot not found: %s", name)
}

func (dm *DomainManager) DeleteSnapshot(domain *MockDomain, name string) error {
	for i, s := range domain.SnapshotList {
		if s.Name == name {
			domain.SnapshotList = append(domain.SnapshotList[:i], domain.SnapshotList[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("snapshot not found: %s", name)
}

func (dm *DomainManager) RestoreSnapshot(domain *MockDomain, name string) error {
	for _, s := range domain.SnapshotList {
		if s.Name == name {
			domain.State = s.State
			return nil
		}
	}
	return fmt.Errorf("snapshot not found: %s", name)
}

func (dm *DomainManager) SetMemory(domain *MockDomain, memory int) error {
	domain.MaxMem = uint64(memory * 1024)
	return nil
}

func (dm *DomainManager) SetVCPUs(domain *MockDomain, vcpus int) error {
	return nil
}

func (dm *DomainManager) ResizeDisk(domain *MockDomain, path string, size uint64) error {
	return nil
}

func (dm *DomainManager) AttachDisk(domain *MockDomain, disk DiskConfig) error {
	return nil
}

func (dm *DomainManager) DetachDisk(domain *MockDomain, device string) error {
	return nil
}

func (dm *DomainManager) AttachNetwork(domain *MockDomain, network NetworkConfig) error {
	return nil
}

func (dm *DomainManager) DetachNetwork(domain *MockDomain, mac string) error {
	return nil
}

func (dm *DomainManager) GetVNCPort(domain *MockDomain) (int, error) {
	if domain.VNCPort == 0 {
		domain.VNCPort = 5900 + len(domain.SnapshotList)%100
	}
	return domain.VNCPort, nil
}

func (dm *DomainManager) UpdateVNCPassword(domain *MockDomain, password string) error {
	return nil
}

func (dm *DomainManager) WaitForState(domain *MockDomain, desiredState int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if domain.State == desiredState {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for state %d", desiredState)
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

type SnapshotInfo struct {
	Name        string
	Description string
	CreatedAt   time.Time
	State       string
}

func GenerateVMConfig(name string, cpu, memory int) *VMConfig {
	return &VMConfig{
		ID:           uuid.New().String(),
		Name:         name,
		UUID:         uuid.New().String(),
		Architecture: "x86_64",
		VCPU:         cpu,
		Memory:       memory,
		Disks: []DiskConfig{
			{
				Path:   fmt.Sprintf("/var/lib/libvirt/images/%s.qcow2", name),
				Format: "qcow2",
				Size:   20,
			},
		},
		Networks: []NetworkConfig{
			{
				MACAddress: "",
				Bridge:     "br0",
			},
		},
	}
}

func GenerateMACAddress() (string, error) {
	bytes := make([]byte, 3)
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", bytes[0], bytes[1], bytes[2]), nil
}
