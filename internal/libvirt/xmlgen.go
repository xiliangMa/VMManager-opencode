package libvirt

import (
	"fmt"
	"strings"
)

type VMConfig struct {
	Name            string
	UUID            string
	MemoryMB        int
	CPU             int
	Architecture    string
	MachineType     string
	DiskPath        string
	DiskFormat      string
	ISOPath         string
	BootOrder       string
	MACAddress      string
	VNCPort         int
	VNCPassword     string
	NetworkBridge   string
	NetworkModel    string
	UEFI            bool
	CPUType         string
	InstallMode     string
}

func GenerateVMXML(config *VMConfig) string {
	arch := config.Architecture
	if arch == "" {
		arch = "x86_64"
	}

	machineType := config.MachineType
	if machineType == "" {
		if arch == "aarch64" {
			machineType = "virt"
		} else {
			machineType = "q35"
		}
	}

	cpuType := config.CPUType
	if cpuType == "" {
		if arch == "aarch64" {
			cpuType = "host"
		} else {
			cpuType = "host-passthrough"
		}
	}

	networkModel := config.NetworkModel
	if networkModel == "" {
		networkModel = "virtio"
	}

	networkBridge := config.NetworkBridge
	if networkBridge == "" {
		networkBridge = "virbr0"
	}

	bootDevs := parseBootOrder(config.BootOrder)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit='MiB'>%d</memory>
  <currentMemory unit='MiB'>%d</currentMemory>
  <vcpu placement='static'>%d</vcpu>
`, config.Name, config.UUID, config.MemoryMB, config.MemoryMB, config.CPU))

	sb.WriteString(generateOSXML(arch, machineType, bootDevs, config.UEFI))

	sb.WriteString(generateFeaturesXML(arch))

	sb.WriteString(fmt.Sprintf(`  <cpu mode='%s' check='none'/>
  <clock offset='utc'>
    <timer name='rtc' tickpolicy='catchup'/>
    <timer name='pit' tickpolicy='delay'/>
    <timer name='hpet' present='no'/>
  </clock>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <pm>
    <suspend-to-mem enabled='no'/>
    <suspend-to-disk enabled='no'/>
  </pm>
`, cpuType))

	sb.WriteString(generateDevicesXML(config, arch, networkModel, networkBridge, bootDevs))

	sb.WriteString(`</domain>`)

	return sb.String()
}

func parseBootOrder(bootOrder string) []string {
	if bootOrder == "" {
		return []string{"hd", "cdrom"}
	}

	parts := strings.Split(bootOrder, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "hd" || p == "cdrom" || p == "network" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"hd", "cdrom"}
	}
	return result
}

func generateOSXML(arch, machineType string, bootDevs []string, uefi bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`  <os>
    <type arch='%s' machine='%s'>hvm</type>
`, arch, machineType))

	if uefi && arch == "x86_64" {
		sb.WriteString(`    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE.fd</loader>
    <nvram>/var/lib/libvirt/qemu/nvram/TEMPLATE_VARS.fd</nvram>
`)
	}

	for _, dev := range bootDevs {
		sb.WriteString(fmt.Sprintf("    <boot dev='%s'/>\n", dev))
	}

	sb.WriteString(`    <bios useserial='yes' rebootTimeout='0'/>
  </os>
`)

	return sb.String()
}

func generateFeaturesXML(arch string) string {
	var sb strings.Builder

	sb.WriteString(`  <features>
    <acpi/>
    <apic/>
`)

	if arch == "x86_64" {
		sb.WriteString(`    <vmport state='off'/>
`)
	}

	sb.WriteString(`  </features>
`)

	return sb.String()
}

func generateDevicesXML(config *VMConfig, arch, networkModel, networkBridge string, bootDevs []string) string {
	var sb strings.Builder

	sb.WriteString(`  <devices>
`)

	emulator := "/usr/bin/qemu-system-x86_64"
	if arch == "aarch64" {
		emulator = "/usr/bin/qemu-system-aarch64"
	}
	sb.WriteString(fmt.Sprintf("    <emulator>%s</emulator>\n", emulator))

	sb.WriteString(generateDiskXML(config, bootDevs))

	if config.ISOPath != "" {
		sb.WriteString(generateCDROMXML(config.ISOPath))
	}

	sb.WriteString(generateNetworkXML(config.MACAddress, networkModel, networkBridge))

	sb.WriteString(generateVNCXML(config.VNCPort, config.VNCPassword))

	sb.WriteString(generateConsoleXML())

	sb.WriteString(generateInputXML(arch))

	sb.WriteString(`  </devices>
`)

	return sb.String()
}

func generateDiskXML(config *VMConfig, bootDevs []string) string {
	diskFormat := config.DiskFormat
	if diskFormat == "" {
		diskFormat = "qcow2"
	}

	hasBoot := false
	for _, dev := range bootDevs {
		if dev == "hd" {
			hasBoot = true
			break
		}
	}

	bootAttr := ""
	if hasBoot {
		bootAttr = "\n      <boot order='1'/>"
	}

	return fmt.Sprintf(`    <disk type='file' device='disk'>
      <driver name='qemu' type='%s' cache='writeback' discard='unmap'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>%s
    </disk>
`, diskFormat, config.DiskPath, bootAttr)
}

func generateCDROMXML(isoPath string) string {
	return fmt.Sprintf(`    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
      <boot order='2'/>
    </disk>
`, isoPath)
}

func generateNetworkXML(macAddress, networkModel, networkBridge string) string {
	return fmt.Sprintf(`    <interface type='bridge'>
      <mac address='%s'/>
      <source bridge='%s'/>
      <model type='%s'/>
    </interface>
`, macAddress, networkBridge, networkModel)
}

func generateVNCXML(vncPort int, vncPassword string) string {
	port := vncPort
	if port == 0 {
		port = -1
	}

	listen := "127.0.0.1"

	var passwdAttr string
	if vncPassword != "" {
		passwdAttr = fmt.Sprintf(" passwd='%s'", vncPassword)
	}

	return fmt.Sprintf(`    <graphics type='vnc' port='%d' autoport='yes' listen='%s'%s>
      <listen type='address' address='%s'/>
    </graphics>
`, port, listen, passwdAttr, listen)
}

func generateConsoleXML() string {
	return `    <serial type='pty'>
      <target type='isa-serial' port='0'>
        <model name='isa-serial'/>
      </target>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
`
}

func generateInputXML(arch string) string {
	var sb strings.Builder

	sb.WriteString(`    <input type='tablet' bus='usb'/>
    <input type='mouse' bus='ps2'/>
    <input type='keyboard' bus='ps2'/>
`)

	if arch == "x86_64" {
		sb.WriteString(`    <input type='tablet' bus='usb'/>
`)
	}

	return sb.String()
}

func GenerateARM64VMXML(config *VMConfig) string {
	if config.Architecture == "" {
		config.Architecture = "aarch64"
	}
	if config.MachineType == "" {
		config.MachineType = "virt"
	}
	if config.CPUType == "" {
		config.CPUType = "host"
	}
	return GenerateVMXML(config)
}

func GenerateX86VMXML(config *VMConfig) string {
	if config.Architecture == "" {
		config.Architecture = "x86_64"
	}
	if config.MachineType == "" {
		config.MachineType = "q35"
	}
	if config.CPUType == "" {
		config.CPUType = "host-passthrough"
	}
	return GenerateVMXML(config)
}
