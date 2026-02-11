package libvirt

import (
	"encoding/xml"
	"fmt"
	"log"
	"sync"
	"time"
)

type Client struct {
	conn    *MockConnection
	uri     string
	mu      sync.RWMutex
	Domains map[string]*MockDomain
}

type MockConnection struct{}

type MockDomain struct {
	Name         string
	UUID         string
	State        int
	CPUTime      uint64
	MaxMem       uint64
	MemUsed      uint64
	DiskPath     string
	VNCPort      int
	XMLDesc      string
	Autostart    bool
	SnapshotList []*MockSnapshot
}

type MockSnapshot struct {
	Name        string
	Description string
	CreatedAt   time.Time
	State       int
	MemoryFile  string
}

func NewClient(uri string) (*Client, error) {
	return &Client{
		conn:    &MockConnection{},
		uri:     uri,
		Domains: make(map[string]*MockDomain),
	}, nil
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) IsConnected() bool {
	return true
}

func (c *Client) GetLibVersion() (uint32, error) {
	return 1000000, nil
}

func (c *Client) GetHostInfo() (*HostInfo, error) {
	return &HostInfo{
		Model:          "Mock Host - QEMU/KVM",
		CPUs:           8,
		MHz:            2400,
		Nodes:          1,
		Sockets:        1,
		CoresPerSocket: 4,
		ThreadsPerCore: 2,
		MaxMemory:      16777216,
		FreeMemory:     8388608,
	}, nil
}

type HostInfo struct {
	Model          string
	CPUs           int
	MHz            int
	Nodes          int
	Sockets        int
	CoresPerSocket int
	ThreadsPerCore int
	MaxMemory      int64
	FreeMemory     int64
}

func (c *Client) ListAllDomains() ([]*MockDomain, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	domains := make([]*MockDomain, 0, len(c.Domains))
	for _, d := range c.Domains {
		domains = append(domains, d)
	}
	return domains, nil
}

func (c *Client) LookupByName(name string) (*MockDomain, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if d, ok := c.Domains[name]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("domain not found: %s", name)
}

func (c *Client) LookupByUUID(uuid string) (*MockDomain, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, d := range c.Domains {
		if d.UUID == uuid {
			return d, nil
		}
	}
	return nil, fmt.Errorf("domain not found: %s", uuid)
}

func (c *Client) LookupByVMID(vmID string) (*MockDomain, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, d := range c.Domains {
		if d.UUID == vmID || d.Name == vmID {
			return d, nil
		}
	}
	return nil, fmt.Errorf("domain not found: %s", vmID)
}

type domainXML struct {
	Name string `xml:"name"`
	UUID string `xml:"uuid"`
}

func (c *Client) DomainCreateXML(xmlData string, flags uint32) (*MockDomain, error) {
	var domainDef domainXML
	if err := xml.Unmarshal([]byte(xmlData), &domainDef); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	log.Printf("[LIBVIRT] Creating domain with Name=%s, UUID=%s", domainDef.Name, domainDef.UUID)

	domain := &MockDomain{
		Name:    domainDef.Name,
		UUID:    domainDef.UUID,
		State:   0,
		CPUTime: 0,
		MaxMem:  4194304,
		MemUsed: 0,
	}

	c.mu.Lock()
	c.Domains[domain.UUID] = domain
	c.mu.Unlock()

	log.Printf("[LIBVIRT] Domain registered: %s", domain.UUID)

	return domain, nil
}

func (c *Client) DefineXML(xmlData string, flags uint32) (*MockDomain, error) {
	var domainDef struct {
		Name string `xml:"name"`
		UUID string `xml:"uuid"`
	}
	if err := xml.Unmarshal([]byte(xmlData), &domainDef); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	domain := &MockDomain{
		Name:    domainDef.Name,
		UUID:    domainDef.UUID,
		State:   0,
		CPUTime: 0,
		MaxMem:  4194304,
		MemUsed: 0,
	}

	c.mu.Lock()
	c.Domains[domain.UUID] = domain
	c.mu.Unlock()

	return domain, nil
}

func (d *MockDomain) GetName() (string, error) {
	return d.Name, nil
}

func (d *MockDomain) GetUUIDString() (string, error) {
	return d.UUID, nil
}

func (d *MockDomain) GetState() (int, uint32, error) {
	return d.State, 0, nil
}

func (d *MockDomain) Create() error {
	d.State = 1
	return nil
}

func (d *MockDomain) Destroy() error {
	d.State = 0
	return nil
}

func (d *MockDomain) Shutdown() error {
	d.State = 0
	return nil
}

func (d *MockDomain) Reset() error {
	return nil
}

func (d *MockDomain) Suspend() error {
	d.State = 3
	return nil
}

func (d *MockDomain) Resume() error {
	d.State = 1
	return nil
}

func (d *MockDomain) Free() error {
	return nil
}

func (d *MockDomain) GetXMLDesc(flags uint32) (string, error) {
	if d.XMLDesc != "" {
		return d.XMLDesc, nil
	}
	return fmt.Sprintf("<domain><name>%s</name><uuid>%s</uuid></domain>", d.Name, d.UUID), nil
}

func (c *Client) DomainEventLifecycleRegister(cb interface{}) (int, error) {
	return 0, nil
}

func (c *Client) GetDomainStats(statsTypes []string, flags uint16) ([]*DomainStats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make([]*DomainStats, 0, len(c.Domains))
	for _, d := range c.Domains {
		stats = append(stats, &DomainStats{
			State:       d.State,
			CPUTime:     int64(d.CPUTime),
			MemoryUsage: int64(d.MemUsed),
			MemoryTotal: int64(d.MaxMem),
			DiskRead:    1024,
			DiskWrite:   2048,
			NetworkRX:   10240,
			NetworkTX:   20480,
		})
	}
	return stats, nil
}

type DomainStats struct {
	State       int
	CPUTime     int64
	MemoryUsage int64
	MemoryTotal int64
	DiskRead    int64
	DiskWrite   int64
	NetworkRX   int64
	NetworkTX   int64
}

func (c *Client) StoragePoolLookupByName(name string) error {
	return nil
}

func (c *Client) ListStoragePools() ([]string, error) {
	return []string{"default", "images"}, nil
}

func (c *Client) NodeGetCPUMap(flags uint32) ([]int, error) {
	return []int{0, 1, 2, 3, 4, 5, 6, 7}, nil
}

func (c *Client) NodeGetMemoryStats(cellNum int, flags uint32) (map[string]int64, error) {
	return map[string]int64{
		"total": 16777216,
		"free":  8388608,
		"used":  8388608,
	}, nil
}

func (c *Client) ListInterfaces() ([]string, error) {
	return []string{"eth0", "br0"}, nil
}

func (c *Client) InterfaceLookupByName(name string) error {
	return nil
}

func (c *Client) GetNetwork(name string) error {
	return nil
}

func (c *Client) ListNetworks() ([]string, error) {
	return []string{"default", "host-only"}, nil
}

func (c *Client) NetworkLookupByName(name string) error {
	return nil
}

func (c *Client) SecretLookupByUsage(usageType int, usageID string) error {
	return nil
}

func (c *Client) ListSecrets() ([]string, error) {
	return []string{}, nil
}

func (c *Client) DomainScreenshot(domain *MockDomain, screen uint32, flags uint32) (string, error) {
	return "", nil
}

func (c *Client) DomainOpenConsole(domain *MockDomain, stream interface{}, flags uint32) error {
	return nil
}

func (c *Client) DomainCreate(domain *MockDomain, flags uint32) error {
	domain.State = 1
	return nil
}

func (c *Client) DomainUndefine(domain *MockDomain) error {
	return nil
}

func (c *Client) DomainGetInfo(domain *MockDomain) (uint8, uint64, uint64, error) {
	return uint8(domain.State), domain.MaxMem, domain.CPUTime, nil
}

func (c *Client) DomainSetMemory(domain *MockDomain, memory uint, flags uint32) error {
	domain.MaxMem = uint64(memory)
	return nil
}

func (c *Client) DomainSetVcpus(domain *MockDomain, nvcpus uint, flags uint32) error {
	return nil
}

func (c *Client) DomainAttachDeviceFlags(domain *MockDomain, xml string, flags uint32) error {
	return nil
}

func (c *Client) DomainDetachDeviceFlags(domain *MockDomain, xml string, flags uint32) error {
	return nil
}

func (c *Client) DomainUpdateDeviceFlags(domain *MockDomain, xml string, flags uint32) error {
	return nil
}

func (c *Client) DomainCoreDumpWithFormat(domain *MockDomain, to string, format uint32, flags uint32) error {
	return nil
}

func (c *Client) DomainManagedSave(domain *MockDomain, flags uint32) error {
	return nil
}

func (c *Client) DomainHasManagedSaveImage(domain *MockDomain, flags uint32) (bool, error) {
	return false, nil
}

func (c *Client) DomainManagedSaveRemove(domain *MockDomain, flags uint32) error {
	return nil
}

func (c *Client) DomainGetMaxMemory(domain *MockDomain) (uint64, error) {
	return domain.MaxMem, nil
}

func (c *Client) DomainGetMaxVcpus(domain *MockDomain) (int, error) {
	return 4, nil
}

func (c *Client) DomainSetMetadata(domain *MockDomain, metadataType int, metadata string, key string, uri string, flags uint32) error {
	return nil
}

func (c *Client) DomainGetMetadata(domain *MockDomain, metadataType int, uri string, flags uint32) (string, error) {
	return "", nil
}

func (c *Client) DomainRename(domain *MockDomain, name string, flags uint32) error {
	domain.Name = name
	return nil
}

func (c *Client) DomainAbortAsyncJob(domain *MockDomain, flags uint32) error {
	return nil
}

func (c *Client) DomainMigratePrepare3(uri string, params *map[string]interface{}, cookie []byte, flags uint64) ([]byte, error) {
	return []byte{}, nil
}

func (c *Client) DomainMigratePerform3(domain *MockDomain, dname string, params []interface{}, flags uint64) error {
	return nil
}

func (c *Client) DomainMigrateConfirm3(domain *MockDomain, cookie []byte, flags uint32) error {
	return nil
}

func (c *Client) DomainGetJobInfo(domain *MockDomain) (int64, int64, int64, int64, int64, uint64, error) {
	return 0, 0, 0, 0, 0, 0, nil
}

func (c *Client) DomainGetAutostart(domain *MockDomain) (bool, error) {
	return domain.Autostart, nil
}

func (c *Client) DomainSetAutostart(domain *MockDomain, autostart int) error {
	domain.Autostart = autostart == 1
	return nil
}

func (c *Client) DomainGetSchedulerType(domain *MockDomain) (string, int, error) {
	return "fair", 1, nil
}

func (c *Client) DomainGetSchedulerParameters(domain *MockDomain) (map[string]int64, error) {
	return map[string]int64{
		"weight":     100,
		"cap":        100,
		"share_min":  0,
		"share":      1000,
		"share_max":  0,
		"share_hard": 0,
	}, nil
}

func (c *Client) DomainSetSchedulerParameters(domain *MockDomain, params map[string]int64, flags uint32) error {
	return nil
}

func (c *Client) DomainSetSchedulerParametersFlags(domain *MockDomain, params map[string]int64, flags uint32) error {
	return nil
}

func (c *Client) DomainInjectNmi(domain *MockDomain, flags uint32) error {
	return nil
}

func (c *Client) SendKey(domain *MockDomain, codeset int, holdtime uint32, keycodes []int, flags uint32) error {
	return nil
}

func (c *Client) DomainDestroyFlags(domain *MockDomain, flags uint32) error {
	domain.State = 0
	return nil
}

func (c *Client) DomainSave(domain *MockDomain, to string, flags uint32) error {
	domain.State = 0
	return nil
}

func (c *Client) DomainRestore(domain *MockDomain, from string, flags uint32) error {
	domain.State = 1
	return nil
}

func (c *Client) DomainSaveImageDefineXML(domain *MockDomain, xml string, flags uint32) error {
	return nil
}

func (c *Client) DomainSaveImageGetXMLDesc(domain *MockDomain, flags uint32) (string, error) {
	return "", nil
}

func (c *Client) DomainBlockJobAbort(domain *MockDomain, path string, flags uint32) error {
	return nil
}

func (c *Client) DomainBlockJobSetSpeed(domain *MockDomain, path string, speed uint64, flags uint32) error {
	return nil
}

func (c *Client) DomainBlockJobInfo(domain *MockDomain, path string, flags uint32) (uint32, int64, int64, error) {
	return 0, 0, 0, nil
}

func (c *Client) DomainBlockPull(domain *MockDomain, path string, base string, bandwidth uint64, flags uint32) error {
	return nil
}

func (c *Client) DomainBlockRebase(domain *MockDomain, path string, base string, bandwidth uint64, flags uint32) error {
	return nil
}

func (c *Client) DomainBlockCommit(domain *MockDomain, disk string, base string, top string, bandwidth uint64, flags uint32) error {
	return nil
}

func (c *Client) DomainBlockStats(domain *MockDomain, path string) (int64, int64, int64, int64, int64, error) {
	return 1024, 2048, 4096, 8192, 0, nil
}

func (c *Client) DomainInterfaceStats(domain *MockDomain, path string) (int64, int64, int64, int64, error) {
	return 10240, 100, 20480, 200, nil
}

func (c *Client) DomainMemoryStats(domain *MockDomain, flags uint32) (map[string]uint64, error) {
	return map[string]uint64{
		"total":     domain.MaxMem,
		"unused":    domain.MaxMem - domain.MemUsed,
		"available": domain.MaxMem - domain.MemUsed,
		"used":      domain.MemUsed,
	}, nil
}

func (c *Client) DomainBlockStatsFlags(domain *MockDomain, path string, flags uint32) (map[string]int64, error) {
	return map[string]int64{
		"rd_req":   100,
		"rd_bytes": 102400,
		"wr_req":   200,
		"wr_bytes": 204800,
		"errs":     0,
	}, nil
}

func (c *Client) DomainInterfaceStatsFlags(domain *MockDomain, path string, flags uint32) (map[string]int64, error) {
	return map[string]int64{
		"rx_bytes":   102400,
		"rx_packets": 1000,
		"tx_bytes":   204800,
		"tx_packets": 2000,
		"errs":       0,
		"drop":       0,
	}, nil
}

func (c *Client) DomainGetOSType(domain *MockDomain) (string, error) {
	return "linux", nil
}

func (c *Client) DomainGetID(domain *MockDomain) (uint32, error) {
	return uint32(domain.State), nil
}

func (c *Client) DomainIsActive(domain *MockDomain) (bool, error) {
	return domain.State == 1, nil
}

func (c *Client) DomainIsPersistent(domain *MockDomain) (bool, error) {
	return true, nil
}

func (c *Client) DomainIsUpdated(domain *MockDomain) (bool, error) {
	return false, nil
}

func (c *Client) DomainGetSecurityLabel(domain *MockDomain) ([]byte, int, error) {
	return []byte{}, 0, nil
}

func (c *Client) NodeDeviceLookupByName(domain *MockDomain, name string) error {
	return nil
}

func (c *Client) ListNodeDevices(domain *MockDomain) ([]string, error) {
	return []string{}, nil
}

func (c *Client) DomainFSAttach(domain *MockDomain, disk string, source string, flags uint32) error {
	return nil
}

func (c *Client) DomainFSDetach(domain *MockDomain, disk string, flags uint32) error {
	return nil
}

func (c *Client) DomainFSInfo(domain *MockDomain) ([]string, string, error) {
	return []string{}, "", nil
}

func (c *Client) DomainBlockResize(domain *MockDomain, path string, size uint64, flags uint32) error {
	return nil
}

func (c *Client) DomainGetDiskErrors(domain *MockDomain) ([]string, int, error) {
	return []string{}, 0, nil
}

func (c *Client) DomainPinVCpuFlags(domain *MockDomain, vcpu uint, cpumap []byte, flags uint32) error {
	return nil
}

func (c *Client) DomainGetVcpuPeriod(domain *MockDomain, flags uint32) (int64, error) {
	return 100000, nil
}

func (c *Client) DomainSetVcpuPeriod(domain *MockDomain, period int64, flags uint32) error {
	return nil
}

func (c *Client) DomainGetEmulatorPinInfo(domain *MockDomain, flags uint32) ([]byte, error) {
	return []byte{}, nil
}

func (c *Client) DomainSetEmulatorPinInfo(domain *MockDomain, cpumap []byte, flags uint32) error {
	return nil
}

func (c *Client) DomainGetVcpus(domain *MockDomain, flags uint32) ([]int, []byte, error) {
	return []int{0, 1, 2, 3}, []byte{}, nil
}

func (c *Client) DomainGetVcpuBitmap(domain *MockDomain, id int, flags uint32) ([]byte, error) {
	return []byte{0xFF}, nil
}

func (c *Client) DomainPinVcpu(domain *MockDomain, vcpu uint, cpumap []byte) error {
	return nil
}

func (c *Client) DomainPinVcpuFlags(domain *MockDomain, vcpu uint, cpumap []byte, flags uint32) error {
	return nil
}

func (c *Client) DomainGetIOThreadInfo(domain *MockDomain, flags uint32) ([]int, error) {
	return []int{}, nil
}

func (c *Client) DomainSetIOThreadParams(domain *MockDomain, params map[string]int, flags uint32) error {
	return nil
}

func (c *Client) DomainAddIOThread(domain *MockDomain, flags uint32) error {
	return nil
}

func (c *Client) DomainDelIOThread(domain *MockDomain, flags uint32) error {
	return nil
}

func (c *Client) DomainSetBlockIoTune(domain *MockDomain, disk string, params map[string]int64, flags uint32) error {
	return nil
}

func (c *Client) DomainGetBlockIoTune(domain *MockDomain, disk string, flags uint32) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (c *Client) DomainSetInterfaceParameters(domain *MockDomain, path string, params map[string]int64, flags uint32) error {
	return nil
}

func (c *Client) DomainGetInterfaceParameters(domain *MockDomain, path string, flags uint32) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (c *Client) DomainMemoryPeek(domain *MockDomain, start uint64, size uint64, flags uint32) ([]byte, error) {
	return make([]byte, size), nil
}

func (c *Client) DomainBlockPeek(domain *MockDomain, path string, start uint64, size uint64, flags uint32) ([]byte, error) {
	return make([]byte, size), nil
}

func (c *Client) DomainGetBlockInfo(domain *MockDomain, path string, flags uint32) (uint64, uint64, uint64, error) {
	return 10737418240, 5368709120, 512, nil
}

func (c *Client) DomainGetTime(domain *MockDomain, flags uint32) (int64, error) {
	return time.Now().Unix(), nil
}

func (c *Client) DomainSetTime(domain *MockDomain, seconds int64, nseconds uint32, flags uint32) error {
	return nil
}

func (c *Client) DomainListGetAllDomainCaps(conn *MockConnection, flags uint32) ([]string, error) {
	return []string{}, nil
}

func (c *Client) NetworkCreateXML(xml string, flags uint32) error {
	return nil
}

func (c *Client) NetworkCreateXMLFrom(xml string, from string, flags uint32) error {
	return nil
}

func (c *Client) NetworkDefineXML(xml string, flags uint32) error {
	return nil
}

func (c *Client) NetworkUndefine(network string) error {
	return nil
}

func (c *Client) NetworkUpdate(network string, index uint32, xml string, flags uint32) error {
	return nil
}

func (c *Client) NetworkDestroy(network string) error {
	return nil
}

func (c *Client) NetworkGetXMLDesc(network string, flags uint32) (string, error) {
	return "", nil
}

func (c *Client) NetworkGetAutostart(network string) (bool, error) {
	return false, nil
}

func (c *Client) NetworkSetAutostart(network string, autostart int) error {
	return nil
}

func (c *Client) NetworkIsActive(network string) (bool, error) {
	return true, nil
}

func (c *Client) NetworkIsPersistent(network string) (bool, error) {
	return true, nil
}

func (c *Client) NetworkBridgeInterface(network string, name string) error {
	return nil
}

func (c *Client) NetworkGetBridgeInterface(network string) (string, error) {
	return "", nil
}

func (c *Client) NetworkGetPhysicalFunction(network string) (string, error) {
	return "", nil
}

func (c *Client) NetworkGetVirtualFunctions(network string) ([]string, error) {
	return []string{}, nil
}

func (c *Client) NetworkListAllPorts(network string, flags uint32) ([]string, error) {
	return []string{}, nil
}

func (c *Client) StoragePoolCreateXML(xml string, flags uint32) error {
	return nil
}

func (c *Client) StoragePoolCreateXMLFrom(xml string, from string, flags uint32) error {
	return nil
}

func (c *Client) StoragePoolDefineXML(xml string, flags uint32) error {
	return nil
}

func (c *Client) StoragePoolUndefine(pool string) error {
	return nil
}

func (c *Client) StoragePoolUpdate(pool string, index uint32, xml string, flags uint32) error {
	return nil
}

func (c *Client) StoragePoolDestroy(pool string) error {
	return nil
}

func (c *Client) StoragePoolDelete(pool string, flags uint32) error {
	return nil
}

func (c *Client) StoragePoolGetXMLDesc(pool string, flags uint32) (string, error) {
	return "", nil
}

func (c *Client) StoragePoolGetAutostart(pool string) (bool, error) {
	return false, nil
}

func (c *Client) StoragePoolSetAutostart(pool string, autostart int) error {
	return nil
}

func (c *Client) StoragePoolIsActive(pool string) (bool, error) {
	return true, nil
}

func (c *Client) StoragePoolIsPersistent(pool string) (bool, error) {
	return true, nil
}

func (c *Client) StoragePoolListAllVolumes(pool string, flags uint32) ([]string, error) {
	return []string{}, nil
}

func (c *Client) StoragePoolRefresh(pool string, flags uint32) error {
	return nil
}

func (c *Client) StoragePoolBuild(pool string, flags uint32) error {
	return nil
}

func (c *Client) StorageVolLookupByName(pool string, name string) error {
	return nil
}

func (c *Client) StorageVolLookupByKey(key string) error {
	return nil
}

func (c *Client) StorageVolLookupByPath(path string) error {
	return nil
}

func (c *Client) StorageVolCreateXML(pool string, xml string, flags uint32) error {
	return nil
}

func (c *Client) StorageVolCreateXMLFrom(pool string, xml string, from string, flags uint32) error {
	return nil
}

func (c *Client) StorageVolDelete(vol string, flags uint32) error {
	return nil
}

func (c *Client) StorageVolResize(vol string, capacity uint64, flags uint32) error {
	return nil
}

func (c *Client) StorageVolGetXMLDesc(vol string, flags uint32) (string, error) {
	return "", nil
}

func (c *Client) StorageVolGetInfo(vol string, flags uint32) (uint64, uint64, error) {
	return 10737418240, 5368709120, nil
}

func (c *Client) StorageVolGetPath(vol string) (string, error) {
	return "", nil
}

func (c *Client) StorageVolDefFindByName(pool string, vol string) error {
	return nil
}

func (c *Client) StorageVolDefFindByKey(key string) error {
	return nil
}

func (c *Client) StorageVolDefFindByPath(path string) error {
	return nil
}

func (c *Client) ListDomains() ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	domains := make([]string, 0, len(c.Domains))
	for _, d := range c.Domains {
		domains = append(domains, d.Name)
	}
	return domains, nil
}

func (c *Client) ListDefinedDomains() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListDefinedStoragePools() ([]string, error) {
	return []string{"default", "images"}, nil
}

func (c *Client) ListDefinedInterfaces() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListDefinedNetworks() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListNWFilters() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListDomainCaps() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListNetworkCaps() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListStoragePoolCaps() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListInterfaceCaps() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListHostname() (string, error) {
	return "mock-host", nil
}

func (c *Client) ConnectGetAllDomainStats(domains []*MockDomain, statsTypes []string, flags uint16) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

func (c *Client) ConnectDomainEventAgentLifecycleCallback(domain *MockDomain, state int, reason int) error {
	return nil
}

func (c *Client) ConnectDomainEventTunableCallback(domain *MockDomain, params map[string]string) error {
	return nil
}

func (c *Client) ConnectDomainEventGraphicsCallback(domain *MockDomain, phase int, family int, addr string, port int, socket string, localAddr string, localPort int) error {
	return nil
}

func (c *Client) ConnectDomainEventBlockJobCallback(domain *MockDomain, disk string, jobType int, status int) error {
	return nil
}

func (c *Client) ConnectListAllSecrets() ([]string, error) {
	return []string{}, nil
}

func (c *Client) ListSecretLookupByUsage(usageType int, usageID string) error {
	return nil
}

func (c *Client) ListSecretDefineXML(xml string, flags uint32) error {
	return nil
}

func (c *Client) ListSecretSetValue(secret string, value []byte, flags uint32) error {
	return nil
}

func (c *Client) ListSecretGetValue(secret string, flags uint32) ([]byte, error) {
	return []byte{}, nil
}

func (c *Client) ListSecretUndefine(secret string) error {
	return nil
}

func (c *Client) ListSecretGetXMLDesc(secret string, flags uint32) (string, error) {
	return "", nil
}

func (c *Client) ListNWFilterLookupByName(name string) error {
	return nil
}

func (c *Client) ListNWFilterDefineXML(xml string, flags uint32) error {
	return nil
}

func (c *Client) ListNWFilterUndefine(nwfilter string) error {
	return nil
}

func (c *Client) ListNWFilterGetXMLDesc(nwfilter string, flags uint32) (string, error) {
	return "", nil
}
