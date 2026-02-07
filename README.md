# VMManager - SaaS虚拟机管理平台

## 概述

VMManager 是一个专业的SaaS虚拟机管理平台，支持用户通过Web界面管理ARM/x86架构的虚拟机实例，具备完整的生命周期管理、模板管理、用户管理等功能。

## 技术栈

### 后端
- **Go 1.21** - 高性能编译型语言
- **Gin** - 轻量级HTTP框架
- **Gorm** - 数据库ORM框架
- **Libvirt + QEMU** - 企业级虚拟化管理
- **PostgreSQL 15** - 关系型数据库
- **Redis 7** - 缓存和会话存储

### 前端
- **React 18** - UI框架
- **TypeScript 5** - 类型安全
- **Ant Design 5.x** - 企业级组件库
- **Zustand** - 状态管理
- **i18next** - 国际化支持
- **Vite 5** - 极速构建

## 功能特性

### 虚拟机管理
- 创建、启动、停止、重启、暂停、恢复虚拟机
- VNC/SPICE 控制台访问
- 资源监控和统计
- 快照管理
- 批量操作

### 模板管理
- 支持 QCOW2、VMDK、OVA 等格式
- 模板上传和下载
- 模板分类和筛选
- 模板使用统计

### 用户管理
- 用户注册和登录
- RBAC 权限控制
- 资源配额管理
- 审计日志

## 快速开始

### 环境要求
- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Libvirt & QEMU

### 使用 Docker 启动

```bash
# 克隆项目
git clone <repository-url>
cd VMManager

# 构建并启动服务
docker-compose up -d

# 访问管理界面
# http://localhost
```

### 本地开发

#### 后端
```bash
cd VMManager

# 安装依赖
go mod tidy

# 运行服务
go run ./cmd/server/
```

#### 前端
```bash
cd web

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

## 默认账号

| 角色 | 用户名 | 密码 |
|-----|-------|-----|
| 管理员 | admin | admin123 |
| 用户 | example | user123 |

## API 文档

启动服务后访问: http://localhost:8080/swagger/index.html

## 配置说明

配置文件: `config.yaml`

```yaml
# 应用配置
app:
  host: "0.0.0.0"
  http_port: 8080
  ws_port: 8081

# 数据库配置
database:
  host: "localhost"
  port: 5432
  username: "vmmanager"
  password: "vmmanager123"
  name: "vmmanager"

# Redis 配置
redis:
  host: "localhost"
  port: 6379

# Libvirt 配置
libvirt:
  uri: "qemu:///system"

# JWT 配置
jwt:
  secret: "vmmanager-jwt-secret-key"
  expiration: 24h
```

## 项目结构

```
VMManager/
├── cmd/                    # 程序入口
├── config/                 # 配置管理
├── internal/               # 内部包
│   ├── api/               # API处理层
│   ├── models/            # 数据模型
│   ├── services/          # 业务逻辑层
│   ├── repository/        # 数据访问层
│   ├── libvirt/           # Libvirt集成
│   ├── websocket/         # WebSocket处理
│   └── tasks/             # 定时任务
├── migrations/             # 数据库迁移
├── templates/              # 模板存储
├── uploads/                # 上传文件
├── web/                    # 前端项目
├── nginx/                  # Nginx配置
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 开发规范

### 后端
- 使用 `go fmt` 格式化代码
- 遵循 Go 模块规范
- 错误处理：使用 `pkg/errors`
- API 响应格式统一

### 前端
- ESLint + Prettier 代码格式化
- TypeScript 严格模式
- 组件命名：PascalCase
- i18n 国际化

## 许可证

MIT License
