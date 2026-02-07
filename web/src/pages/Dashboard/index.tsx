import React from 'react'
import { useNavigate } from 'react-router-dom'
import { Row, Col, Card, Statistic, Button } from 'antd'
import { useTranslation } from 'react-i18next'
import { 
  DesktopOutlined, 
  TeamOutlined, 
  FileOutlined,
  RocketOutlined 
} from '@ant-design/icons'
import { useAuthStore } from '../../stores/authStore'

const Dashboard: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user } = useAuthStore()

  const stats = [
    { 
      title: t('vm.vmList'), 
      value: 10, 
      icon: <DesktopOutlined style={{ color: '#1890ff' }} />,
      path: '/vms'
    },
    { 
      title: t('admin.totalUsers'), 
      value: 100, 
      icon: <TeamOutlined style={{ color: '#52c41a' }} />,
      path: '/admin/users'
    },
    { 
      title: t('template.upload'), 
      value: 5, 
      icon: <FileOutlined style={{ color: '#722ed1' }} />,
      path: '/templates'
    },
    { 
      title: t('vm.running'), 
      value: 8, 
      icon: <RocketOutlined style={{ color: '#fa8c16' }} />,
      path: '/vms'
    }
  ]

  return (
    <div>
      <h1>Welcome, {user?.username}</h1>
      
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {stats.map((stat, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card hoverable onClick={() => navigate(stat.path)}>
              <Statistic
                title={stat.title}
                value={stat.value}
                prefix={stat.icon}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]}>
        <Col span={24}>
          <Card title={t('vm.vmList')}>
            <Button type="primary" onClick={() => navigate('/vms/create')}>
              {t('vm.createVM')}
            </Button>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
