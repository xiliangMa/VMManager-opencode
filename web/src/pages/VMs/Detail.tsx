import React, { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Card, Row, Col, Statistic, Button, Space, Tag, Descriptions, Tabs, message, Popconfirm } from 'antd'
import { ArrowLeftOutlined, PoweroffOutlined, DeleteOutlined, CloudUploadOutlined, EditOutlined, SyncOutlined } from '@ant-design/icons'
import { vmsApi, VM } from '../../api/client'
import dayjs from 'dayjs'

const VMDetail: React.FC = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [vm, setVm] = useState<VM | null>(null)
  const [locked, setLocked] = useState(false)

  const fetchVm = async () => {
    if (!id) return
    try {
      const response = await vmsApi.get(id)
      setVm(response.data || response)
      if (locked && ['running', 'stopped'].includes(response.data?.status || response.status)) {
        setLocked(false)
      }
    } catch (error) {
      message.error(t('message.failedToLoad') + ' VM')
    }
  }

  useEffect(() => {
    fetchVm()
  }, [id])

  const handleStart = async () => {
    setLocked(true)
    try {
      await vmsApi.start(id!)
      message.success(t('vm.start') + ' ' + t('common.success'))
      fetchVm()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
      setLocked(false)
    }
  }

  const handleStop = async () => {
    setLocked(true)
    try {
      await vmsApi.stop(id!)
      message.success(t('vm.stop') + ' ' + t('common.success'))
      fetchVm()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
      setLocked(false)
    }
  }

  const handleRestart = async () => {
    setLocked(true)
    try {
      await vmsApi.restart(id!)
      message.success(t('vm.restart') + ' ' + t('common.success'))
      fetchVm()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
      setLocked(false)
    }
  }

  const handleDelete = async () => {
    setLocked(true)
    try {
      await vmsApi.delete(id!)
      message.success(t('vm.delete') + ' ' + t('common.success'))
      navigate('/vms')
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
      setLocked(false)
    }
  }

  const statusColors: Record<string, string> = {
    running: 'green',
    stopped: 'red',
    suspended: 'orange',
    pending: 'blue',
    creating: 'processing',
    error: 'error',
    starting: 'processing',
    stopping: 'processing'
  }

  if (!vm) return <div>{t('common.loading')}</div>

  const tabItems = [
    {
      key: 'info',
      label: t('tab.information'),
      children: (
        <Descriptions bordered column={2}>
          <Descriptions.Item label={t('detail.id')}>{vm.id}</Descriptions.Item>
          <Descriptions.Item label={t('vm.name')}>{vm.name}</Descriptions.Item>
          <Descriptions.Item label={t('vm.status')}>
            <Tag color={statusColors[vm.status]}>{vm.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('table.vcpu')}>{vm.cpuAllocated} vCPU</Descriptions.Item>
          <Descriptions.Item label={t('vm.memory')}>{vm.memoryAllocated} MB</Descriptions.Item>
          <Descriptions.Item label={t('vm.disk')}>{vm.diskAllocated} GB</Descriptions.Item>
          <Descriptions.Item label={t('vm.ipAddress')}>{vm.ipAddress || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('detail.macAddress')}>{vm.macAddress || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('detail.vncPort')}>{vm.vncPort || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('detail.createdAt')}>{dayjs(vm.createdAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
          <Descriptions.Item label={t('detail.updatedAt')}>{dayjs(vm.updatedAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
        </Descriptions>
      )
    },
    {
      key: 'snapshots',
      label: (
        <span>
          <CloudUploadOutlined />
          {t('vm.snapshots')}
        </span>
      ),
      children: (
        <Card size="small" style={{ marginTop: 8 }}>
          <Space direction="vertical" align="center" style={{ width: '100%' }}>
            <p>{t('vm.snapshots')}</p>
            <Button type="primary" onClick={() => navigate(`/vms/${id}/snapshots`)}>
              {t('vm.snapshots')}
            </Button>
          </Space>
        </Card>
      )
    },
    {
      key: 'logs',
      label: t('vm.logs'),
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
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            {locked && (
              <Tag color="processing" icon={<SyncOutlined spin />}>{t('vm.operationInProgress')}</Tag>
            )}
            <a href={`/vms/${id}/edit`} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, color: locked ? '#999' : '#1677ff', textDecoration: 'none', pointerEvents: locked ? 'none' : 'auto' }}>
              <EditOutlined /> {t('common.edit')}
            </a>
            <Button icon={<PoweroffOutlined />} onClick={() => navigate(`/vms/${id}/console`)} disabled={vm.status !== 'running' || locked}>
              {t('console.fullscreen')}
            </Button>
            <Popconfirm title={t('popconfirm.areYouSure')} onConfirm={handleDelete} disabled={locked}>
              <Button danger icon={<DeleteOutlined />} disabled={locked}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          </div>
        }
      >
        <Space style={{ marginBottom: 16 }}>
          {vm.status === 'running' && !locked ? (
            <>
              <Button onClick={handleStop}>{t('vm.stop')}</Button>
              <Button onClick={handleRestart}>{t('vm.restart')}</Button>
              <Button onClick={() => navigate(`/vms/${id}/console`)}>{t('vm.console')}</Button>
            </>
          ) : !locked && (vm.status === 'stopped' || vm.status === 'pending' || vm.status === 'creating') ? (
            <Button type="primary" onClick={handleStart}>{t('vm.start')}</Button>
          ) : (
            <Tag color="processing" icon={locked ? <SyncOutlined spin /> : undefined}>
              {locked ? t('vm.operationInProgress') : t(`vm.${vm.status}`)}
            </Tag>
          )}
        </Space>

        <Tabs items={tabItems} />
      </Card>
    </div>
  )
}

export default VMDetail
