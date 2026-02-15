package libvirt

import (
	"crypto/rand"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/libvirt/libvirt-go"
)

type Client struct {
	conn *libvirt.Connect
	uri  string
}

type Domain struct {
	domain *libvirt.Domain
	Name   string
	UUID   string
	State  int
}

func NewClient(uri string) (*Client, error) {
	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt: %w", err)
	}

	log.Printf("[LIBVIRT] Connected to libvirt: %s", uri)

	return &Client{
		conn: conn,
		uri:  uri,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		_, err := c.conn.Close()
		return err
	}
	return nil
}

func (c *Client) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	alive, err := c.conn.IsAlive()
	return err == nil && alive
}

func (c *Client) LookupByName(name string) (*Domain, error) {
	domain, err := c.conn.LookupDomainByName(name)
	if err != nil {
		return nil, err
	}
	return c.wrapDomain(domain), nil
}

func (c *Client) LookupByUUID(uuid string) (*Domain, error) {
	domain, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return nil, err
	}
	return c.wrapDomain(domain), nil
}

func (c *Client) UndefineDomain(uuid string) error {
	domain, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()
	return domain.Undefine()
}

func (c *Client) UndefineDomainByName(name string) error {
	domain, err := c.conn.LookupDomainByName(name)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()
	return domain.Undefine()
}

func (c *Client) DomainCreateXML(xmlData string) (*Domain, error) {
	domain, err := c.conn.DomainCreateXML(xmlData, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create domain: %w", err)
	}

	log.Printf("[LIBVIRT] Domain created successfully")
	return c.wrapDomain(domain), nil
}

func (c *Client) DefineXML(xmlData string) (*Domain, error) {
	domain, err := c.conn.DomainDefineXML(xmlData)
	if err != nil {
		return nil, fmt.Errorf("failed to define domain: %w", err)
	}

	log.Printf("[LIBVIRT] Domain defined successfully")
	return c.wrapDomain(domain), nil
}

func (c *Client) wrapDomain(domain *libvirt.Domain) *Domain {
	uuid, _ := domain.GetUUIDString()
	name, _ := domain.GetName()
	state, _, _ := domain.GetState()

	return &Domain{
		Name:   name,
		UUID:   uuid,
		State:  int(state),
		domain: domain,
	}
}

func (d *Domain) GetName() (string, error) {
	return d.domain.GetName()
}

func (d *Domain) GetUUIDString() (string, error) {
	return d.domain.GetUUIDString()
}

func (d *Domain) GetState() (int, uint32, error) {
	state, _, err := d.domain.GetState()
	return int(state), 0, err
}

func (d *Domain) Create() error {
	return d.domain.Create()
}

func (d *Domain) Destroy() error {
	return d.domain.Destroy()
}

func (d *Domain) Shutdown() error {
	return d.domain.Shutdown()
}

func (d *Domain) Reset() error {
	return d.domain.Reset(0)
}

func (d *Domain) Suspend() error {
	return d.domain.Suspend()
}

func (d *Domain) Resume() error {
	return d.domain.Resume()
}

func (d *Domain) Free() error {
	return d.domain.Free()
}

func (d *Domain) GetXMLDesc() (string, error) {
	return d.domain.GetXMLDesc(0)
}

func (c *Client) ListStoragePools() ([]string, error) {
	return c.conn.ListStoragePools()
}

func (c *Client) ListNetworks() ([]string, error) {
	return c.conn.ListNetworks()
}

func (c *Client) NetworkLookupByName(name string) error {
	_, err := c.conn.LookupNetworkByName(name)
	return err
}

func (c *Client) NetworkDefineXML(xml string) error {
	_, err := c.conn.NetworkDefineXML(xml)
	return err
}

func (c *Client) NetworkCreateXML(xml string) error {
	_, err := c.conn.NetworkCreateXML(xml)
	return err
}

func (c *Client) NetworkUndefine(network string) error {
	net, err := c.conn.LookupNetworkByName(network)
	if err != nil {
		return err
	}
	return net.Undefine()
}

func (c *Client) NetworkDestroy(network string) error {
	net, err := c.conn.LookupNetworkByName(network)
	if err != nil {
		return err
	}
	return net.Destroy()
}

func (c *Client) StoragePoolLookupByName(name string) error {
	_, err := c.conn.LookupStoragePoolByName(name)
	return err
}

func (c *Client) StorageVolLookupByPath(path string) error {
	_, err := c.conn.LookupStorageVolByPath(path)
	return err
}

func (c *Client) SecretLookupByUsage(usageType libvirt.SecretUsageType, usageID string) error {
	_, err := c.conn.LookupSecretByUsage(usageType, usageID)
	return err
}

func (c *Client) ListSecrets() ([]string, error) {
	return c.conn.ListSecrets()
}

func (c *Client) ListDomains() ([]uint32, error) {
	return c.conn.ListDomains()
}

func (c *Client) ListDefinedDomains() ([]string, error) {
	return c.conn.ListDefinedDomains()
}

func (c *Client) GetHostname() (string, error) {
	return c.conn.GetHostname()
}

func (c *Client) GetLibVersion() (uint32, error) {
	return c.conn.GetLibVersion()
}

func (c *Client) GetFreeMemory() (uint64, error) {
	return c.conn.GetFreeMemory()
}

func (c *Client) GetNodeInfo() (*libvirt.NodeInfo, error) {
	return c.conn.GetNodeInfo()
}

func (c *Client) AttachISO(domainUUID string, isoPath string) error {
	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to get domain XML: %w", err)
	}

	newXML, err := updateCDROMXML(xmlDesc, isoPath)
	if err != nil {
		return fmt.Errorf("failed to update CDROM XML: %w", err)
	}

	_, err = c.conn.DomainDefineXML(newXML)
	if err != nil {
		return fmt.Errorf("failed to define domain with new ISO: %w", err)
	}

	log.Printf("[LIBVIRT] ISO attached to domain %s: %s", domainUUID, isoPath)
	return nil
}

func (c *Client) DetachISO(domainUUID string) error {
	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to get domain XML: %w", err)
	}

	newXML, err := updateCDROMXML(xmlDesc, "")
	if err != nil {
		return fmt.Errorf("failed to update CDROM XML: %w", err)
	}

	_, err = c.conn.DomainDefineXML(newXML)
	if err != nil {
		return fmt.Errorf("failed to define domain without ISO: %w", err)
	}

	log.Printf("[LIBVIRT] ISO detached from domain %s", domainUUID)
	return nil
}

func (c *Client) GetMountedISO(domainUUID string) (string, error) {
	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return "", fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return "", fmt.Errorf("failed to get domain XML: %w", err)
	}

	return extractISOPath(xmlDesc), nil
}

func updateCDROMXML(xmlDesc string, isoPath string) (string, error) {
	cdromRegex := regexp.MustCompile(`(?s)<disk type='file' device='cdrom'>.*?</disk>`)

	if !cdromRegex.MatchString(xmlDesc) {
		if isoPath == "" {
			return xmlDesc, nil
		}

		targetRegex := regexp.MustCompile(`</devices>`)
		cdromXML := fmt.Sprintf(`<disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>`, isoPath)
		return targetRegex.ReplaceAllString(xmlDesc, cdromXML+"</devices>"), nil
	}

	if isoPath == "" {
		newDiskXML := `<disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>`
		return cdromRegex.ReplaceAllString(xmlDesc, newDiskXML), nil
	}

	sourceRegex := regexp.MustCompile(`(?s)(<disk type='file' device='cdrom'>.*?<driver[^/]*/>)(.*?)(</disk>)`)
	if sourceRegex.MatchString(xmlDesc) {
		newSource := fmt.Sprintf(`$1
      <source file='%s'/>
      $3`, isoPath)
		return sourceRegex.ReplaceAllString(xmlDesc, newSource), nil
	}

	driverRegex := regexp.MustCompile(`(?s)(<disk type='file' device='cdrom'>.*?<driver[^/]*/>)(</disk>)`)
	if driverRegex.MatchString(xmlDesc) {
		newXML := fmt.Sprintf(`$1
      <source file='%s'/>
      $2`, isoPath)
		return driverRegex.ReplaceAllString(xmlDesc, newXML), nil
	}

	return xmlDesc, nil
}

func extractISOPath(xmlDesc string) string {
	sourceRegex := regexp.MustCompile(`<source file='([^']+)'[^/]*/?\s*>`)
	matches := sourceRegex.FindAllStringSubmatch(xmlDesc, -1)

	for _, match := range matches {
		if len(match) > 1 && strings.HasSuffix(strings.ToLower(match[1]), ".iso") {
			return match[1]
		}
	}

	cdromRegex := regexp.MustCompile(`(?s)<disk type='file' device='cdrom'>.*?<source file='([^']+)'.*?</disk>`)
	matches = cdromRegex.FindAllStringSubmatch(xmlDesc, -1)

	for _, match := range matches {
		if len(match) > 1 {
			return match[1]
		}
	}

	return ""
}

func (c *Client) CloneVM(sourceUUID string, newName string, newDiskPath string) (string, error) {
	sourceDomain, err := c.conn.LookupDomainByUUIDString(sourceUUID)
	if err != nil {
		return "", fmt.Errorf("source domain not found: %w", err)
	}
	defer sourceDomain.Free()

	xmlDesc, err := sourceDomain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return "", fmt.Errorf("failed to get domain XML: %w", err)
	}

	newUUID := generateUUID()
	newMAC, err := generateMACAddress()
	if err != nil {
		return "", fmt.Errorf("failed to generate MAC address: %w", err)
	}

	newXML, err := modifyCloneXML(xmlDesc, newName, newUUID, newMAC, newDiskPath)
	if err != nil {
		return "", fmt.Errorf("failed to modify XML: %w", err)
	}

	newDomain, err := c.conn.DomainDefineXML(newXML)
	if err != nil {
		return "", fmt.Errorf("failed to define cloned domain: %w", err)
	}
	defer newDomain.Free()

	log.Printf("[LIBVIRT] VM cloned: %s -> %s (UUID: %s)", sourceUUID, newName, newUUID)

	return newUUID, nil
}

func (c *Client) GetDomainXML(uuid string) (string, error) {
	domain, err := c.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return "", fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return "", fmt.Errorf("failed to get domain XML: %w", err)
	}

	return xmlDesc, nil
}

func modifyCloneXML(xmlDesc string, newName string, newUUID string, newMAC string, newDiskPath string) (string, error) {
	uuidRegex := regexp.MustCompile(`<uuid>[^<]+</uuid>`)
	xmlDesc = uuidRegex.ReplaceAllString(xmlDesc, fmt.Sprintf("<uuid>%s</uuid>", newUUID))

	nameRegex := regexp.MustCompile(`<name>[^<]+</name>`)
	xmlDesc = nameRegex.ReplaceAllString(xmlDesc, fmt.Sprintf("<name>%s</name>", newName))

	macRegex := regexp.MustCompile(`<mac address='[^']+'`)
	xmlDesc = macRegex.ReplaceAllString(xmlDesc, fmt.Sprintf("<mac address='%s'", newMAC))

	if newDiskPath != "" {
		diskSourceRegex := regexp.MustCompile(`(<disk type='file' device='disk'>.*?<source file=')[^']+(')`)
		xmlDesc = diskSourceRegex.ReplaceAllString(xmlDesc, fmt.Sprintf("${1}%s${2}", newDiskPath))
	}

	return xmlDesc, nil
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint32(b[4])<<8|uint32(b[5]),
		(uint32(b[6])<<8|uint32(b[7]))&0x4fff|0x4000,
		(uint32(b[8])<<8|uint32(b[9]))&0x3fff|0x8000,
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	)
}

func generateMACAddress() (string, error) {
	b := make([]byte, 3)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", b[0], b[1], b[2]), nil
}

func (c *Client) NetworkCreate(name string) error {
	net, err := c.conn.LookupNetworkByName(name)
	if err != nil {
		return fmt.Errorf("network not found: %w", err)
	}
	defer net.Free()

	return net.Create()
}

func (c *Client) NetworkSetAutostart(name string, autostart bool) error {
	net, err := c.conn.LookupNetworkByName(name)
	if err != nil {
		return fmt.Errorf("network not found: %w", err)
	}
	defer net.Free()

	return net.SetAutostart(autostart)
}

func (c *Client) NetworkGetInfo(name string) (map[string]interface{}, error) {
	net, err := c.conn.LookupNetworkByName(name)
	if err != nil {
		return nil, fmt.Errorf("network not found: %w", err)
	}
	defer net.Free()

	active, err := net.IsActive()
	if err != nil {
		return nil, err
	}

	autostart, err := net.GetAutostart()
	if err != nil {
		return nil, err
	}

	xmlDesc, err := net.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":      name,
		"active":    active,
		"autostart": autostart,
		"xml":       xmlDesc,
	}, nil
}

type StoragePoolInfo struct {
	Name      string
	Type      string
	Target    string
	Source    string
	Capacity  uint64
	Available uint64
	Used      uint64
	Active    bool
	Autostart bool
}

func (c *Client) StoragePoolDefineXML(xmlDef string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	_, err := c.conn.StoragePoolDefineXML(xmlDef, 0)
	return err
}

func (c *Client) StoragePoolCreate(name string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(name)
	if err != nil {
		return fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	return pool.Create(0)
}

func (c *Client) StoragePoolDestroy(name string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(name)
	if err != nil {
		return fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	return pool.Destroy()
}

func (c *Client) StoragePoolUndefine(name string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(name)
	if err != nil {
		return fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	return pool.Undefine()
}

func (c *Client) StoragePoolSetAutostart(name string, autostart bool) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(name)
	if err != nil {
		return fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	return pool.SetAutostart(autostart)
}

func (c *Client) StoragePoolGetInfo(name string) (*StoragePoolInfo, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(name)
	if err != nil {
		return nil, fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	info, err := pool.GetInfo()
	if err != nil {
		return nil, err
	}

	active, err := pool.IsActive()
	if err != nil {
		return nil, err
	}

	autostart, err := pool.GetAutostart()
	if err != nil {
		return nil, err
	}

	xmlDesc, err := pool.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	poolType := ""
	target := ""
	source := ""
	if xmlDesc != "" {
		typeMatch := regexp.MustCompile(`type=['"]([^'"]+)['"]`).FindStringSubmatch(xmlDesc)
		if len(typeMatch) > 1 {
			poolType = typeMatch[1]
		}
		targetMatch := regexp.MustCompile(`<path>([^<]+)</path>`).FindStringSubmatch(xmlDesc)
		if len(targetMatch) > 1 {
			target = targetMatch[1]
		}
		sourceMatch := regexp.MustCompile(`<dir\s+path=['"]([^'"]+)['"]`).FindStringSubmatch(xmlDesc)
		if len(sourceMatch) > 1 {
			source = sourceMatch[1]
		}
	}

	return &StoragePoolInfo{
		Name:      name,
		Type:      poolType,
		Target:    target,
		Source:    source,
		Capacity:  info.Capacity,
		Available: info.Available,
		Used:      info.Capacity - info.Available,
		Active:    active,
		Autostart: autostart,
	}, nil
}

func (c *Client) StoragePoolList() ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("libvirt connection is nil")
	}
	pools, err := c.conn.ListDefinedStoragePools()
	if err != nil {
		return nil, err
	}

	activePools, err := c.conn.ListStoragePools()
	if err != nil {
		return nil, err
	}

	return append(pools, activePools...), nil
}

func (c *Client) StoragePoolRefresh(name string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(name)
	if err != nil {
		return fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	return pool.Refresh(0)
}

type StorageVolumeInfo struct {
	Name       string
	Type       string
	Capacity   uint64
	Allocation uint64
	Path       string
}

func (c *Client) StorageVolumeList(poolName string) ([]StorageVolumeInfo, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	volumes, err := pool.ListAllStorageVolumes(0)
	if err != nil {
		return nil, err
	}

	var result []StorageVolumeInfo
	for _, vol := range volumes {
		name, err := vol.GetName()
		if err != nil {
			vol.Free()
			continue
		}

		info, err := vol.GetInfo()
		if err != nil {
			vol.Free()
			continue
		}

		path, err := vol.GetPath()
		if err != nil {
			path = ""
		}

		result = append(result, StorageVolumeInfo{
			Name:       name,
			Capacity:   info.Capacity,
			Allocation: info.Allocation,
			Path:       path,
		})

		vol.Free()
	}

	return result, nil
}

func (c *Client) StorageVolumeCreate(poolName, name string, capacity int64, format string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	xml := fmt.Sprintf(`<volume>
  <name>%s</name>
  <capacity unit="bytes">%d</capacity>
  <target>
    <format type="%s"/>
  </target>
</volume>`, name, capacity, format)

	_, err = pool.StorageVolCreateXML(xml, 0)
	return err
}

func (c *Client) StorageVolumeDelete(poolName, volumeName string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}
	pool, err := c.conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return fmt.Errorf("storage pool not found: %w", err)
	}
	defer pool.Free()

	vol, err := pool.LookupStorageVolByName(volumeName)
	if err != nil {
		return fmt.Errorf("volume not found: %w", err)
	}
	defer vol.Free()

	return vol.Delete(0)
}

func (c *Client) CreateSnapshot(domainUUID string, snapshotName string, description string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}

	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	snapshotXML := fmt.Sprintf(`<domainsnapshot>
  <name>%s</name>
  <description>%s</description>
</domainsnapshot>`, snapshotName, description)

	_, err = domain.CreateSnapshotXML(snapshotXML, 0)
	return err
}

func (c *Client) ListSnapshots(domainUUID string) ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("libvirt connection is nil")
	}

	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	snapshots, err := domain.ListAllSnapshots(0)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, snap := range snapshots {
		name, err := snap.GetName()
		if err == nil {
			names = append(names, name)
		}
		snap.Free()
	}

	return names, nil
}

type SnapshotInfo struct {
	Name        string
	Description string
	State       string
	IsCurrent   bool
	CreatedAt   string
}

func (c *Client) GetSnapshotInfo(domainUUID string, snapshotName string) (*SnapshotInfo, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("libvirt connection is nil")
	}

	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	snap, err := domain.SnapshotLookupByName(snapshotName, 0)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}
	defer snap.Free()

	name, _ := snap.GetName()
	xml, _ := snap.GetXMLDesc(0)
	isCurrent, _ := snap.IsCurrent(0)

	return &SnapshotInfo{
		Name:        name,
		Description: parseSnapshotDescription(xml),
		State:       parseSnapshotState(xml),
		IsCurrent:   isCurrent,
		CreatedAt:   parseSnapshotCreationTime(xml),
	}, nil
}

func (c *Client) RevertToSnapshot(domainUUID string, snapshotName string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}

	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	snap, err := domain.SnapshotLookupByName(snapshotName, 0)
	if err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}
	defer snap.Free()

	return snap.RevertToSnapshot(0)
}

func (c *Client) DeleteSnapshot(domainUUID string, snapshotName string) error {
	if c.conn == nil {
		return fmt.Errorf("libvirt connection is nil")
	}

	domain, err := c.conn.LookupDomainByUUIDString(domainUUID)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	snap, err := domain.SnapshotLookupByName(snapshotName, 0)
	if err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}
	defer snap.Free()

	return snap.Delete(0)
}

func parseSnapshotDescription(xml string) string {
	return findSubstring(xml, "<description>", "</description>")
}

func parseSnapshotState(xml string) string {
	state := findSubstring(xml, "<state>", "</state>")
	if state != "" {
		return state
	}
	return "unknown"
}

func parseSnapshotCreationTime(xml string) string {
	return findSubstring(xml, "<creationTime>", "</creationTime>")
}

func findSubstring(xml, startTag, endTag string) string {
	for i := 0; i < len(xml); i++ {
		if i+len(startTag) <= len(xml) && xml[i:i+len(startTag)] == startTag {
			end := i + len(startTag)
			for j := end; j < len(xml); j++ {
				if j+len(endTag) <= len(xml) && xml[j:j+len(endTag)] == endTag {
					return xml[end:j]
				}
			}
		}
	}
	return ""
}
