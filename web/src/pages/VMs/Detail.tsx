import React, { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Card, Row, Col, Statistic, Button, Space, Tag, Descriptions, Tabs, message, Popconfirm, Modal, Select, Spin, Input } from 'antd'
import { ArrowLeftOutlined, PoweroffOutlined, DeleteOutlined, CloudUploadOutlined, EditOutlined, SyncOutlined, SettingOutlined, FileOutlined, LinkOutlined, DisconnectOutlined, CopyOutlined, ClockCircleOutlined, CameraOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { vmsApi, isosApi, ISO } from '../../api/client'
import type { VMDetail } from '../../api/client'
import VMBackups from './Backups'
import VMSnapshots from './Snapshots'
import HotplugModal from '../../components/VM/HotplugModal'
import dayjs from 'dayjs'

const VMDetail: React.FC = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [vm, setVm] = useState<VMDetail | null>(null)
  const [locked, setLocked] = useState(false)
  const [mountedISO, setMountedISO] = useState<{ mounted: boolean; isoId: string; isoName: string; isoPath: string } | null>(null)
  const [isoModalVisible, setIsoModalVisible] = useState(false)
  const [isoList, setIsoList] = useState<ISO[]>([])
  const [selectedISO, setSelectedISO] = useState<string>('')
  const [isoLoading, setIsoLoading] = useState(false)
  const [cloneModalVisible, setCloneModalVisible] = useState(false)
  const [cloneName, setCloneName] = useState('')
  const [cloneDescription, setCloneDescription] = useState('')
  const [cloneLoading, setCloneLoading] = useState(false)
  const [hotplugModalVisible, setHotplugModalVisible] = useState(false)

  const fetchVm = async () => {
    if (!id) return
    try {
      const response = await vmsApi.get(id)
      const newVm = response.data || response
      setVm(newVm)
      // 只在状态从中间状态变为最终状态时解锁
      if (locked && !['starting', 'stopping', 'creating', 'pending'].includes(newVm.status)) {
        setLocked(false)
      }
    } catch (error) {
      message.error(t('message.failedToLoad') + ' VM')
    }
  }

  const fetchMountedISO = async () => {
    if (!id) return
    try {
      const response = await vmsApi.getMountedISO(id)
      const data = response.data || response
      setMountedISO(data)
    } catch (_error) {
    }
  }

  useEffect(() => {
    fetchVm()
    fetchMountedISO()
  }, [id])

  // 当 VM 处于中间状态时，自动轮询更新
  useEffect(() => {
    if (vm && ['starting', 'stopping', 'creating'].includes(vm.status)) {
      const interval = setInterval(() => {
        fetchVm()
      }, 2000)
      return () => clearInterval(interval)
    }
  }, [vm?.status])

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

  const handleOpenISOModal = async () => {
    setIsoLoading(true)
    setIsoModalVisible(true)
    try {
      const response = await isosApi.list({ page: 1, page_size: 100 })
      const data = response.data || response
      setIsoList(data.items || data || [])
      if (mountedISO?.isoId) {
        setSelectedISO(mountedISO.isoId)
      }
    } catch (error) {
      message.error(t('message.failedToLoad') + ' ISO')
    } finally {
      setIsoLoading(false)
    }
  }

  const handleMountISO = async () => {
    if (!selectedISO) {
      message.warning(t('iso.selectISO'))
      return
    }
    setIsoLoading(true)
    try {
      await vmsApi.mountISO(id!, selectedISO)
      message.success(t('iso.mountSuccess'))
      setIsoModalVisible(false)
      fetchMountedISO()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setIsoLoading(false)
    }
  }

  const handleUnmountISO = async () => {
    setIsoLoading(true)
    try {
      await vmsApi.unmountISO(id!)
      message.success(t('iso.unmountSuccess'))
      fetchMountedISO()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setIsoLoading(false)
    }
  }

  const handleOpenCloneModal = () => {
    if (vm) {
      setCloneName(`${vm.name}-clone`)
      setCloneDescription('')
      setCloneModalVisible(true)
    }
  }

  const handleCloneVM = async () => {
    if (!cloneName.trim()) {
      message.warning(t('vm.cloneNameRequired'))
      return
    }
    setCloneLoading(true)
    try {
      const response = await vmsApi.clone(id!, { 
        name: cloneName, 
        description: cloneDescription 
      })
      message.success(t('vm.cloneSuccess'))
      setCloneModalVisible(false)
      navigate(`/vms/${response.data?.id || response.id}`)
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setCloneLoading(false)
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
          <CameraOutlined />
          {t('snapshot.snapshots')}
        </span>
      ),
      children: vm?.id ? <VMSnapshots vmId={vm.id} vmStatus={vm.status} /> : <div>{t('common.loading')}</div>
    },
    {
      key: 'backups',
      label: (
        <span>
          <ClockCircleOutlined />
          {t('backup.backups')}
        </span>
      ),
      children: vm?.id ? <VMBackups vmId={vm.id} /> : <div>{t('common.loading')}</div>
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
            <Button icon={<CopyOutlined />} onClick={handleOpenCloneModal} disabled={locked}>
              {t('vm.clone')}
            </Button>
            <Button icon={<ThunderboltOutlined />} onClick={() => setHotplugModalVisible(true)} disabled={locked}>
              {t('hotplug.title')}
            </Button>
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
              {mountedISO?.mounted ? (
                <Popconfirm
                  title={t('iso.unmountConfirm')}
                  onConfirm={handleUnmountISO}
                >
                  <Button icon={<DisconnectOutlined />} danger>
                    {t('iso.unmount')}: {mountedISO.isoName}
                  </Button>
                </Popconfirm>
              ) : (
                <Button icon={<LinkOutlined />} onClick={handleOpenISOModal}>
                  {t('iso.mount')}
                </Button>
              )}
              {(vm.install_status === 'installing') && (
                <Tag color="processing">{t('installation.installing')}</Tag>
              )}
            </>
          ) : vm.status === 'stopped' && !locked ? (
            <Space>
              <Button type="primary" onClick={handleStart}>{t('vm.start')}</Button>
              {mountedISO?.mounted ? (
                <Popconfirm
                  title={t('iso.unmountConfirm')}
                  onConfirm={handleUnmountISO}
                >
                  <Button icon={<DisconnectOutlined />} danger>
                    {t('iso.unmount')}: {mountedISO.isoName}
                  </Button>
                </Popconfirm>
              ) : (
                <Button icon={<LinkOutlined />} onClick={handleOpenISOModal}>
                  {t('iso.mount')}
                </Button>
              )}
              {(!vm.is_installed || vm.install_status === '') && (
                <Button 
                  icon={<SettingOutlined />} 
                  onClick={() => navigate(`/vms/${id}/installation`)}
                >
                  {t('installation.install')}
                </Button>
              )}
              {vm.install_status === 'completed' && !vm.agent_installed && (
                <Button 
                  icon={<SettingOutlined />} 
                  onClick={() => navigate(`/vms/${id}/installation`)}
                >
                  {t('installation.installAgent')}
                </Button>
              )}
              {vm.agent_installed && (
                <Tag color="success" icon={<CloudUploadOutlined />}>
                  {t('installation.agentInstalled')}
                </Tag>
              )}
            </Space>
          ) : locked || ['starting', 'stopping', 'creating', 'pending'].includes(vm.status) ? (
            <Tag color="processing" icon={<SyncOutlined spin />}>
              {t('vm.operationInProgress')}
            </Tag>
          ) : (
            <Tag color={statusColors[vm.status]}>
              {t(`vm.${vm.status}`)}
            </Tag>
          )}
        </Space>

        <Tabs items={tabItems} />
      </Card>

      <Modal
        title={t('iso.mount')}
        open={isoModalVisible}
        onCancel={() => setIsoModalVisible(false)}
        onOk={handleMountISO}
        confirmLoading={isoLoading}
        okText={t('iso.mount')}
        cancelText={t('common.cancel')}
      >
        <Spin spinning={isoLoading}>
          <div style={{ marginBottom: 16 }}>
            <p>{t('iso.selectISOHint')}</p>
          </div>
          <Select
            style={{ width: '100%' }}
            placeholder={t('iso.selectISO')}
            value={selectedISO}
            onChange={setSelectedISO}
            showSearch
            optionFilterProp="children"
          >
            {isoList.map((iso) => (
              <Select.Option key={iso.id} value={iso.id}>
                <Space>
                  <FileOutlined />
                  {iso.name} ({iso.osType} - {(iso.fileSize / 1024 / 1024 / 1024).toFixed(2)} GB)
                </Space>
              </Select.Option>
            ))}
          </Select>
        </Spin>
      </Modal>

      <Modal
        title={t('vm.clone')}
        open={cloneModalVisible}
        onCancel={() => setCloneModalVisible(false)}
        onOk={handleCloneVM}
        confirmLoading={cloneLoading}
        okText={t('vm.clone')}
        cancelText={t('common.cancel')}
      >
        <Spin spinning={cloneLoading}>
          <div style={{ marginBottom: 16 }}>
            <p style={{ marginBottom: 8 }}>{t('vm.cloneHint')}</p>
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', marginBottom: 4 }}>{t('vm.name')}</label>
            <Input
              value={cloneName}
              onChange={(e) => setCloneName(e.target.value)}
              placeholder={t('vm.namePlaceholder')}
            />
          </div>
          <div>
            <label style={{ display: 'block', marginBottom: 4 }}>{t('common.description')}</label>
            <Input.TextArea
              value={cloneDescription}
              onChange={(e) => setCloneDescription(e.target.value)}
              placeholder={t('vm.descriptionPlaceholder')}
              rows={3}
            />
          </div>
        </Spin>
      </Modal>

      <HotplugModal
        vmId={id || ''}
        vmName={vm?.name || ''}
        visible={hotplugModalVisible}
        onClose={() => setHotplugModalVisible(false)}
        onSuccess={fetchVm}
      />
    </div>
  )
}

export default VMDetail
