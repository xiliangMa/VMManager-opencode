package libvirt

import (
	"fmt"
	"log"

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
