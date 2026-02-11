import React from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Avatar, Dropdown, Button, theme } from 'antd'
import {
  DashboardOutlined,
  DesktopOutlined,
  FileOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  TranslationOutlined,
  BellOutlined,
  MonitorOutlined,
  HistoryOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../stores/authStore'

const { Header, Sider, Content } = Layout

const MainLayout: React.FC = () => {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore()
  const [collapsed, setCollapsed] = React.useState(false)
  const {
    token: { colorBgContainer, borderRadiusLG }
  } = theme.useToken()

  const menuItems = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: t('common.dashboard')
    },
    {
      key: '/dashboard',
      icon: <MonitorOutlined />,
      label: t('common.systemMonitor')
    },
    {
      key: '/vms',
      icon: <DesktopOutlined />,
      label: t('vm.vmList')
    },
    {
      key: '/templates',
      icon: <FileOutlined />,
      label: t('template.templateList')
    },
    ...(user?.role === 'admin' ? [
      {
        key: '/admin/users',
        icon: <UserOutlined />,
        label: t('admin.userManagement')
      },
      {
        key: '/admin/audit-logs',
        icon: <BellOutlined />,
        label: t('common.auditLogs')
      },
      {
        key: '/admin/alerts',
        icon: <BellOutlined />,
        label: t('common.alertRules')
      },
      {
        key: '/admin/alert-history',
        icon: <HistoryOutlined />,
        label: t('alerts.alertHistory')
      }
    ] : [])
  ]

  const userMenuItems = [
    {
      key: 'profile',
      icon: <SettingOutlined />,
      label: t('common.profile')
    },
    {
      key: 'language',
      icon: <TranslationOutlined />,
      label: i18n.language === 'zh-CN' ? 'English' : '中文'
    },
    {
      type: 'divider' as const
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('auth.logout'),
      danger: true
    }
  ]

  const handleMenuClick = (key: string) => {
    if (key === 'logout') {
      logout()
      navigate('/login')
    } else if (key === 'language') {
      i18n.changeLanguage(i18n.language === 'zh-CN' ? 'en-US' : 'zh-CN')
    } else if (key === 'profile') {
      navigate('/settings/profile')
    } else {
      navigate(key)
    }
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div style={{ 
          height: 64, 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          color: '#fff',
          fontSize: collapsed ? 16 : 20,
          fontWeight: 'bold'
        }}>
          {collapsed ? t('app.vm') : t('app.vmManager')}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ 
          padding: '0 24px', 
          background: colorBgContainer, 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'space-between'
        }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          
          <Dropdown menu={{ items: userMenuItems, onClick: ({ key }) => handleMenuClick(key) }}>
            <div style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}>
              <Avatar icon={<UserOutlined />} />
              <span>{user?.username}</span>
            </div>
          </Dropdown>
        </Header>
        <Content style={{ 
          margin: 24, 
          padding: 24, 
          minHeight: 280,
          background: colorBgContainer,
          borderRadius: borderRadiusLG
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout
