import React, { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Card, Row, Col, Statistic, Button, Space, Tag, Descriptions, Tabs, message, Popconfirm } from 'antd'
import { ArrowLeftOutlined, PoweroffOutlined, DeleteOutlined, CloudUploadOutlined } from '@ant-design/icons'
import { vmsApi, VM } from '../../api/client'
import dayjs from 'dayjs'

const VMDetail: React.FC = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [vm, setVm] = useState<VM | null>(null)

  const fetchVm = async () => {
    if (!id) return
    try {
      const response = await vmsApi.get(id)
      setVm(response.data || response)
    } catch (error) {
      message.error('Failed to fetch VM')
    }
  }

  useEffect(() => {
    fetchVm()
  }, [id])

  const handleStart = async () => {
    try {
      await vmsApi.start(id!)
      message.success(t('vm.start') + ' ' + t('common.success'))
      fetchVm()
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  const handleStop = async () => {
    try {
      await vmsApi.stop(id!)
      message.success(t('vm.stop') + ' ' + t('common.success'))
      fetchVm()
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  const handleRestart = async () => {
    try {
      await vmsApi.restart(id!)
      message.success(t('vm.restart') + ' ' + t('common.success'))
      fetchVm()
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  const handleDelete = async () => {
    try {
      await vmsApi.delete(id!)
      message.success(t('vm.delete') + ' ' + t('common.success'))
      navigate('/vms')
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  const statusColors: Record<string, string> = {
    running: 'green',
    stopped: 'red',
    suspended: 'orange',
    pending: 'blue',
    creating: 'processing',
    error: 'error'
  }

  if (!vm) return <div>Loading...</div>

  const tabItems = [
    {
      key: 'info',
      label: 'Information',
      children: (
        <Descriptions bordered column={2}>
          <Descriptions.Item label="ID">{vm.id}</Descriptions.Item>
          <Descriptions.Item label={t('vm.name')}>{vm.name}</Descriptions.Item>
          <Descriptions.Item label={t('vm.status')}>
            <Tag color={statusColors[vm.status]}>{vm.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="CPU">{vm.cpuAllocated} vCPU</Descriptions.Item>
          <Descriptions.Item label={t('vm.memory')}>{vm.memoryAllocated} MB</Descriptions.Item>
          <Descriptions.Item label={t('vm.disk')}>{vm.diskAllocated} GB</Descriptions.Item>
          <Descriptions.Item label={t('vm.ipAddress')}>{vm.ipAddress || '-'}</Descriptions.Item>
          <Descriptions.Item label="MAC Address">{vm.macAddress || '-'}</Descriptions.Item>
          <Descriptions.Item label="VNC Port">{vm.vncPort || '-'}</Descriptions.Item>
          <Descriptions.Item label="Created At">{dayjs(vm.createdAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
          <Descriptions.Item label="Updated At">{dayjs(vm.updatedAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
        </Descriptions>
      )
    },
    {
      key: 'snapshots',
      label: (
        <span>
          <CloudUploadOutlined />
          Snapshots
        </span>
      ),
      children: (
        <Card size="small" style={{ marginTop: 8 }}>
          <Space direction="vertical" align="center" style={{ width: '100%' }}>
            <p>Manage VM snapshots</p>
            <Button type="primary" onClick={() => navigate(`/vms/${id}/snapshots`)}>
              Manage Snapshots
            </Button>
          </Space>
        </Card>
      )
    },
    {
      key: 'logs',
      label: 'Logs',
      children: <div>{t('common.noData')}</div>
    }
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/vms')}>
          {t('common.back')}
        </Button>
      </Space>

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('vm.status')}
              value={vm.status}
              valueStyle={{ color: statusColors[vm.status] === 'green' ? '#3f8600' : '#cf1322' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title={t('vm.cpu')} value={vm.cpuAllocated} suffix="vCPU" />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title={t('vm.memory')} value={vm.memoryAllocated} suffix="MB" />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title={t('vm.disk')} value={vm.diskAllocated} suffix="GB" />
          </Card>
        </Col>
      </Row>

      <Card
        title={vm.name}
        extra={
          <Space>
            <Button icon={<PoweroffOutlined />} onClick={() => navigate(`/vms/${id}/console`)}>
              {t('console.fullscreen')}
            </Button>
            <Popconfirm title="Are you sure?" onConfirm={handleDelete}>
              <Button danger icon={<DeleteOutlined />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          </Space>
        }
      >
        <Space style={{ marginBottom: 16 }}>
          {vm.status === 'running' ? (
            <>
              <Button onClick={handleStop}>{t('vm.stop')}</Button>
              <Button onClick={handleRestart}>{t('vm.restart')}</Button>
              <Button onClick={() => navigate(`/vms/${id}/console`)}>Console</Button>
            </>
          ) : vm.status === 'stopped' ? (
            <Button type="primary" onClick={handleStart}>{t('vm.start')}</Button>
          ) : (
            <Button onClick={handleStart}>{t('vm.resume')}</Button>
          )}
        </Space>

        <Tabs items={tabItems} />
      </Card>
    </div>
  )
}

export default VMDetail
