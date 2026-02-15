# VMManager 开发计划

## 项目目标

VMManager 主要实现 X86 平台虚拟机管理，支持基于模板（国产凝思系统 arm64）创建虚拟机直接启动，以及 ISO 上传和基于 ISO 安装系统的虚拟机创建功能。

## 当前实现状态

> **检查日期**: 2026-02-16
> **检查方式**: 代码审查和功能验证

### 已实现功能

#### 虚拟机管理
- ✅ 虚拟机创建（基于模板）- 已验证 [vm.go:82-178](internal/api/handlers/vm.go)
- ✅ 虚拟机生命周期管理（启动、停止、重启、暂停、恢复）- 已验证 [vm.go](internal/api/handlers/vm.go)
- ✅ VNC/SPICE 控制台访问 - 已验证 [Console.tsx](web/src/pages/VMs/Console.tsx)
- ✅ 虚拟机资源监控 - 已验证 [Monitor.tsx](web/src/pages/VMs/Monitor.tsx)
- ✅ 快照管理 - 已验证 [Snapshots.tsx](web/src/pages/VMs/Snapshots.tsx)
- ✅ 批量操作 - 已验证 [batch.go](internal/api/handlers/batch.go)
- ✅ 虚拟机列表和详情展示 - 已验证
- ✅ 虚拟机搜索和筛选 - 已验证
- ✅ 虚拟机克隆 - 已验证 [vm.go:CloneVM](internal/api/handlers/vm.go)
- ✅ 虚拟机备份和恢复 - 已验证 [backup.go](internal/api/handlers/backup.go)
- ✅ 虚拟机热插拔 - 已验证 [HotplugModal.tsx](web/src/components/VM/HotplugModal.tsx)
- ✅ ISO 挂载/卸载 - 已验证 [vm.go](internal/api/handlers/vm.go)

#### 模板管理
- ✅ 模板上传（QCOW2、VMDK、OVA 格式）- 已验证 [template.go](internal/api/handlers/template.go)
  - 支持分片上传、断点续传
  - 支持 MD5 校验
- ✅ 模板列表和详情 - 已验证
- ✅ 模板分类和筛选 - 已验证
- ✅ 模板使用统计 - 已验证
- ✅ 模板下载功能 - 已验证

#### ISO 管理
- ✅ ISO 文件上传 - 已验证 [iso.go](internal/api/handlers/iso.go)
  - 支持分片上传、断点续传
- ✅ ISO 文件列表管理 - 已验证 [List.tsx](web/src/pages/ISO/List.tsx)
- ✅ ISO 文件删除 - 已验证
- ✅ ISO 文件校验 - 已验证

#### 用户管理
- ✅ 用户注册和登录 - 已验证 [auth.go](internal/api/handlers/auth.go)
- ✅ RBAC 权限控制 - 已验证（admin/user 角色）
- ✅ 资源配额管理 - 已验证 [admin.go](internal/api/handlers/admin.go)
- ✅ 审计日志 - 已验证 [AuditLogs.tsx](web/src/pages/Admin/AuditLogs.tsx)
- ✅ 用户头像上传 - 已验证 [Profile.tsx](web/src/pages/Settings/Profile.tsx)
- ✅ 用户资料管理 - 已验证

#### 系统管理
- ✅ 系统监控（CPU、内存、磁盘）- 已验证 [index.tsx](web/src/pages/Dashboard/index.tsx)
- ✅ 告警规则配置 - 已验证 [AlertRules.tsx](web/src/pages/Admin/AlertRules.tsx)
- ✅ 告警历史记录 - 已验证 [AlertHistory.tsx](web/src/pages/Admin/AlertHistory.tsx)
- ✅ 通知管理（邮件、钉钉、Webhook）- 已验证 [manager.go](internal/notification/manager.go)

#### 网络管理
- ✅ 虚拟网络创建 - 已验证 [virtual_network.go](internal/api/handlers/virtual_network.go)
- ✅ 虚拟网络编辑 - 已验证
- ✅ 虚拟网络删除 - 已验证
- ✅ 虚拟网络列表 - 已验证 [VirtualNetworks.tsx](web/src/pages/Admin/VirtualNetworks.tsx)
- ✅ 虚拟网络启动/停止 - 已验证

#### 存储管理
- ✅ 存储池创建 - 已验证 [storage.go](internal/api/handlers/storage.go)
- ✅ 存储池编辑 - 已验证
- ✅ 存储池删除 - 已验证
- ✅ 存储池列表 - 已验证 [StoragePools.tsx](web/src/pages/Admin/StoragePools.tsx)
- ✅ 存储池启动/停止/刷新 - 已验证
- ✅ 存储卷管理 - 已验证

#### 操作历史
- ✅ 登录历史记录 - 已验证 [operation_history.go](internal/api/handlers/operation_history.go)
- ✅ 资源变更历史 - 已验证
- ✅ 虚拟机操作历史 - 已验证
- ✅ 操作历史前端页面 - 已验证 [OperationHistory.tsx](web/src/pages/Admin/OperationHistory.tsx)

#### 国际化
- ✅ 中文支持 - 已验证 [zh/translation.json](web/public/locales/zh/translation.json)
- ✅ 英文支持 - 已验证 [en/translation.json](web/public/locales/en/translation.json)
- ✅ 后端国际化 - 已验证 [zh-CN.json](translations/zh-CN.json)

### 发现的问题

1. **虚拟机创建后的实际启动**
   - 位置：`internal/api/handlers/vm.go`
   - 问题：创建虚拟机后只保存到数据库，未实际调用 Libvirt 创建域
   - 建议：需要实现异步任务来创建实际的虚拟机

### 未实现功能

#### X86 架构支持
- ❌ X86 虚拟机创建优化
- ❌ X86 模板管理优化
- ❌ X86 架构检测和适配

#### 基于 ISO 的虚拟机安装
- ✅ ISO 选择界面 - 已实现
- ✅ 安装向导 - 已实现 [Installation.tsx](web/src/pages/VMs/Installation.tsx)
- ⚠️ 自动化安装脚本 - 需要完善
- ✅ 安装进度监控 - 已实现

#### 高级虚拟机功能
- ❌ 虚拟机迁移
- ❌ 虚拟机导入导出

#### IP 地址管理
- ❌ IP 地址池管理
- ❌ IP 地址分配
- ❌ IP 地址冲突检测

## 开发计划

### 第一阶段：ISO 管理和安装功能（优先级：高）✅ 已完成

**目标**：实现 ISO 文件管理和基于 ISO 创建虚拟机

#### 1.1 ISO 文件管理 ✅ 已完成

**后端开发**：
- [x] 创建 ISO 数据模型
  - 文件位置：`internal/models/models.go`
- [x] 实现 ISO 上传接口
  - 文件位置：`internal/api/handlers/iso.go`
  - 功能：支持大文件上传、断点续传、MD5 校验
- [x] 实现 ISO 列表接口
- [x] 实现 ISO 删除接口
- [x] 实现 ISO 详情接口
- [x] 创建 ISO 存储仓库

**前端开发**：
- [x] 创建 ISO 管理页面
  - 文件位置：`web/src/pages/ISO/List.tsx`
- [x] 实现 ISO 上传组件
- [x] 添加路由和菜单项

#### 1.2 基于 ISO 创建虚拟机 ✅ 已完成

**后端开发**：
- [x] 扩展虚拟机创建接口
  - 文件位置：`internal/api/handlers/vm.go`
- [x] 实现 ISO 安装逻辑
- [x] 实现安装进度监控
- [ ] 实现自动化安装脚本支持（待完善）
  - 支持 Cloud-init
  - 支持 Kickstart（X86）
  - 支持 Preseed（Debian/Ubuntu）

**前端开发**：
- [x] 扩展虚拟机创建表单
  - 文件位置：`web/src/pages/VMs/Create.tsx`
- [x] 实现安装向导界面
  - 文件位置：`web/src/pages/VMs/Installation.tsx`
- [x] 实现安装进度显示

### 第二阶段：高级虚拟机功能（优先级：高）

**目标**：实现虚拟机克隆、备份、快照等高级功能

#### 2.1 虚拟机克隆 ✅ 已完成

**后端开发**：
- [x] 实现虚拟机克隆接口
  - 完整克隆（复制所有磁盘）
  - 链接克隆（使用 backing file）
- [x] 实现克隆进度监控
- [x] 实现克隆后配置修改

**前端开发**：
- [x] 虚拟机克隆界面

#### 2.2 虚拟机备份和恢复 ✅ 已完成

**后端开发**：
- [x] 实现虚拟机备份接口
  - 完整备份
  - 增量备份
- [x] 实现备份存储管理
- [x] 实现虚拟机恢复接口
- [x] 实现备份计划管理

**前端开发**：
- [x] 备份管理界面
  - 文件位置：`web/src/pages/VMs/Backups.tsx`

#### 2.3 快照管理 ✅ 已完成

**后端开发**：
- [x] 实现快照创建接口
- [x] 实现快照恢复接口
- [x] 实现快照删除接口

**前端开发**：
- [x] 快照管理界面
  - 文件位置：`web/src/pages/VMs/Snapshots.tsx`

### 第三阶段：网络和存储管理（优先级：中）✅ 已完成

**目标**：实现虚拟网络和存储池管理功能

#### 3.1 虚拟网络管理 ✅ 已完成

**后端开发**：
- [x] 创建虚拟网络模型
- [x] 实现虚拟网络创建接口
  - NAT 网络
  - 桥接网络
  - 隔离网络
- [x] 实现虚拟网络编辑接口
- [x] 实现虚拟网络删除接口
- [x] 实现虚拟网络列表接口
- [x] 实现虚拟网络启动/停止接口

**前端开发**：
- [x] 虚拟网络管理界面
  - 文件位置：`web/src/pages/Admin/VirtualNetworks.tsx`

#### 3.2 存储池管理 ✅ 已完成

**后端开发**：
- [x] 创建存储池模型
- [x] 实现存储池创建接口
- [x] 实现存储池编辑接口
- [x] 实现存储池删除接口
- [x] 实现存储池列表接口
- [x] 实现存储池启动/停止/刷新接口

**前端开发**：
- [x] 存储池管理界面
  - 文件位置：`web/src/pages/Admin/StoragePools.tsx`

#### 3.3 存储卷管理 ✅ 已完成

**后端开发**：
- [x] 实现存储卷创建接口
- [x] 实现存储卷删除接口

**前端开发**：
- [x] 存储卷管理界面

### 第四阶段：操作历史记录（优先级：中）✅ 已完成

**目标**：实现操作历史记录功能

#### 4.1 操作历史记录 ✅ 已完成

**后端开发**：
- [x] 创建操作历史数据模型
  - 登录历史表
  - 资源变更历史表
  - 虚拟机操作历史表
- [x] 实现登录历史记录
- [x] 实现资源变更历史记录
- [x] 实现虚拟机操作历史记录
- [x] 实现操作历史查询接口

**前端开发**：
- [x] 操作历史管理界面
  - 文件位置：`web/src/pages/Admin/OperationHistory.tsx`

### 第五阶段：系统优化和完善（优先级：低）

**目标**：系统优化和功能完善

#### 5.1 虚拟机迁移

**后端开发**：
- [ ] 实现虚拟机迁移接口
  - 在线迁移（Live Migration）
  - 离线迁移
- [ ] 实现迁移前置检查
- [ ] 实现迁移进度监控

**前端开发**：
- [ ] 虚拟机迁移界面

**预计工时**：5-7 天

#### 5.2 IP 地址管理

**后端开发**：
- [ ] 实现 IP 地址池管理
- [ ] 实现 IP 地址分配
- [ ] 实现 IP 地址回收
- [ ] 实现 IP 地址冲突检测

**前端开发**：
- [ ] IP 地址管理界面

**预计工时**：3-4 天

#### 5.3 自动化安装脚本

**后端开发**：
- [ ] 支持 Cloud-init
- [ ] 支持 Kickstart（X86）
- [ ] 支持 Preseed（Debian/Ubuntu）

**前端开发**：
- [ ] 自动化安装脚本配置界面

**预计工时**：3-4 天

## 技术要点

### ISO 上传实现

```go
// 大文件上传，支持断点续传
func (h *ISOHandler) UploadISO(c *gin.Context) {
    // 1. 接收文件
    file, header, err := c.Request.FormFile("file")
    
    // 2. 计算文件 MD5
    md5Hash := md5.New()
    tee := io.TeeReader(file, md5Hash)
    
    // 3. 保存文件
    isoPath := filepath.Join(h.isoStoragePath, header.Filename)
    dst, _ := os.Create(isoPath)
    io.Copy(dst, tee)
    
    // 4. 保存到数据库
    iso := models.ISO{
        Name: header.Filename,
        Size: header.Size,
        MD5: hex.EncodeToString(md5Hash.Sum(nil)),
        Status: "active",
    }
    h.isoRepo.Create(c, &iso)
    
    c.JSON(200, gin.H{"code": 0, "data": iso})
}
```

### X86 虚拟机创建

```go
// X86 虚拟机 XML 配置
func generateX86XML(vm *models.VirtualMachine) string {
    return fmt.Sprintf(`
<domain type='kvm'>
  <name>%s</name>
  <memory unit='MiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os>
    <type arch='x86_64' machine='q35'>hvm</type>
    <boot dev='hd'/>
    <boot dev='cdrom'/>
  </os>
  <features>
    <acpi/>
    <apic/>
  </features>
  <cpu mode='host-passthrough'/>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='hda' bus='ide'/>
      <readonly/>
    </disk>
  </devices>
</domain>`, vm.Name, vm.MemoryAllocated, vm.CPUAllocated, vm.DiskPath, vm.ISOPath)
}
```

### ISO 安装流程

```go
// ISO 安装虚拟机
func (h *VMHandler) CreateVMFromISO(c *gin.Context) {
    // 1. 创建虚拟机记录
    vm := models.VirtualMachine{
        Name: req.Name,
        InstallationMode: "iso",
        ISOID: &isoID,
        InstallationStatus: "installing",
    }
    
    // 2. 创建磁盘文件
    diskPath := createDisk(vm.DiskAllocated)
    
    // 3. 生成虚拟机 XML（CD-ROM 启动）
    xml := generateVMXMLWithISO(vm, iso.Path)
    
    // 4. 定义并启动虚拟机
    domain := h.libvirtClient.DefineXML(xml)
    domain.Create()
    
    // 5. 启动安装监控
    go monitorInstallation(vm.ID, domain)
    
    c.JSON(200, gin.H{"code": 0, "data": vm})
}
```

## 开发环境

### Mac mini M4 开发环境

开发主要在 Mac mini M4 (ARM64) 上进行，需要注意：

1. **Libvirt 安装**：
   ```bash
   brew install libvirt qemu
   brew services start libvirt
   ```

2. **架构限制**：
   - Mac mini M4 (ARM64) 无法运行 X86 虚拟机
   - 可以运行 ARM64 虚拟机进行测试
   - X86 功能需要远程连接到 X86 服务器进行测试

3. **远程 Libvirt 配置（用于 X86 测试）**：
   ```yaml
   libvirt:
     uri: "qemu+ssh://user@x86-server/system"
   ```

4. **本地 ARM64 虚拟机测试**：
   - 可以使用本地 Libvirt 创建 ARM64 虚拟机
   - 支持测试虚拟机创建、启动、停止等基本功能
   - 支持测试 VNC 控制台访问

### 测试环境

建议搭建 X86 测试服务器：
- 安装 Libvirt 和 QEMU
- 配置 SSH 免密登录
- 配置防火墙规则

## 里程碑

- **里程碑 1**（预计 2 周）：完成 ISO 管理和基于 ISO 创建虚拟机功能
- **里程碑 2**（预计 3 周）：完成 X86 架构支持和国产凝思系统模板
- **里程碑 3**（预计 5 周）：完成虚拟机克隆、迁移、备份功能
- **里程碑 4**（预计 7 周）：完成网络管理功能
- **里程碑 5**（预计 8 周）：完成存储管理功能

## 风险和挑战

1. **跨架构兼容性**：ARM64 开发环境无法直接测试 X86 功能
   - 解决方案：搭建远程 X86 测试环境

2. **大文件上传**：ISO 文件通常较大（数 GB）
   - 解决方案：实现断点续传、分片上传

3. **安装自动化**：不同操作系统安装流程差异大
   - 解决方案：支持多种自动化安装方案（Cloud-init、Kickstart、Preseed）

4. **性能优化**：虚拟机操作可能耗时较长
   - 解决方案：使用异步任务、WebSocket 推送进度

## 后续规划

1. **容器支持**：支持 LXC 容器管理
2. **Kubernetes 集成**：支持 K8s 集群管理
3. **多租户增强**：更完善的租户隔离和资源限制
4. **监控告警增强**：集成 Prometheus、Grafana
5. **自动化运维**：支持 Ansible、Terraform 集成
6. **AI 运维**：智能资源调度、异常检测

## 参考资源

- [Libvirt 官方文档](https://libvirt.org/docs.html)
- [QEMU 官方文档](https://www.qemu.org/docs/master/)
- [Cloud-init 文档](https://cloudinit.readthedocs.io/)
- [Kickstart 文档](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/performing_an_advanced_rhel_installation/index)
