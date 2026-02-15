import React, { useState, useEffect } from 'react'
import { Modal, Slider, Button, Space, message, Spin, Alert, Divider, Typography } from 'antd'
import { ThunderboltOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { vmsApi } from '../../api/client'

const { Text } = Typography

interface HotplugModalProps {
  vmId: string
  vmName: string
  visible: boolean
  onClose: () => void
  onSuccess?: () => void
}

interface HotplugStatus {
  vcpu_hotplug_enabled: boolean
  memory_hotplug_enabled: boolean
  current_vcpus: number
  current_memory: number
  max_vcpus: number
  max_memory: number
}

const HotplugModal: React.FC<HotplugModalProps> = ({ vmId, vmName, visible, onClose, onSuccess }) => {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState<HotplugStatus | null>(null)
  const [vcpus, setVcpus] = useState(1)
  const [memory, setMemory] = useState(512)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (visible && vmId) {
      fetchStatus()
    }
  }, [visible, vmId])

  const fetchStatus = async () => {
    setLoading(true)
    try {
      const response = await vmsApi.getHotplugStatus(vmId)
      const data = response.data || response
      setStatus(data)
      setVcpus(data.current_vcpus || 1)
      setMemory(data.current_memory || 512)
    } catch (error) {
      console.error('Failed to fetch hotplug status:', error)
      message.error(t('hotplug.fetchFailed'))
    } finally {
      setLoading(false)
    }
  }

  const handleHotplugCPU = async () => {
    if (!status?.vcpu_hotplug_enabled) {
      message.warning(t('hotplug.vcpuNotEnabled'))
      return
    }
    setSaving(true)
    try {
      await vmsApi.hotplugCPU(vmId, vcpus)
      message.success(t('hotplug.vcpuSuccess'))
      fetchStatus()
      onSuccess?.()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setSaving(false)
    }
  }

  const handleHotplugMemory = async () => {
    if (!status?.memory_hotplug_enabled) {
      message.warning(t('hotplug.memoryNotEnabled'))
      return
    }
    setSaving(true)
    try {
      await vmsApi.hotplugMemory(vmId, memory)
      message.success(t('hotplug.memorySuccess'))
      fetchStatus()
      onSuccess?.()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setSaving(false)
    }
  }

  const formatMemory = (mb: number) => {
    if (mb >= 1024) {
      return `${(mb / 1024).toFixed(1)} GB`
    }
    return `${mb} MB`
  }

  return (
    <Modal
      title={
        <Space>
          <ThunderboltOutlined />
          {t('hotplug.title')} - {vmName}
        </Space>
      }
      open={visible}
      onCancel={onClose}
      footer={null}
      width={500}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}>
          <Spin />
        </div>
      ) : status ? (
        <div>
          {!status.vcpu_hotplug_enabled && !status.memory_hotplug_enabled && (
            <Alert
              message={t('hotplug.notEnabled')}
              description={t('hotplug.notEnabledDesc')}
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}

          <Divider orientation="left">{t('hotplug.cpuSection')}</Divider>
          
          {status.vcpu_hotplug_enabled ? (
            <>
              <div style={{ marginBottom: 16 }}>
                <Text type="secondary">{t('hotplug.currentVcpus')}: </Text>
                <Text strong>{status.current_vcpus}</Text>
                <Text type="secondary"> / {t('hotplug.maxVcpus')}: </Text>
                <Text strong>{status.max_vcpus}</Text>
              </div>
              <div style={{ marginBottom: 8 }}>
                <Text>{t('hotplug.adjustVcpus')}</Text>
              </div>
              <Slider
                min={1}
                max={status.max_vcpus}
                value={vcpus}
                onChange={setVcpus}
                marks={{
                  1: '1',
                  [status.max_vcpus]: `${status.max_vcpus}`
                }}
                disabled={saving}
              />
              <div style={{ textAlign: 'center', marginBottom: 16 }}>
                <Text strong style={{ fontSize: 18 }}>{vcpus} {t('hotplug.cores')}</Text>
              </div>
              <Button
                type="primary"
                onClick={handleHotplugCPU}
                loading={saving}
                disabled={vcpus === status.current_vcpus}
                block
              >
                {t('hotplug.applyVcpu')}
              </Button>
            </>
          ) : (
            <Alert
              message={t('hotplug.vcpuNotEnabled')}
              type="warning"
              showIcon
            />
          )}

          <Divider orientation="left">{t('hotplug.memorySection')}</Divider>
          
          {status.memory_hotplug_enabled ? (
            <>
              <div style={{ marginBottom: 16 }}>
                <Text type="secondary">{t('hotplug.currentMemory')}: </Text>
                <Text strong>{formatMemory(status.current_memory)}</Text>
                <Text type="secondary"> / {t('hotplug.maxMemory')}: </Text>
                <Text strong>{formatMemory(status.max_memory)}</Text>
              </div>
              <div style={{ marginBottom: 8 }}>
                <Text>{t('hotplug.adjustMemory')}</Text>
              </div>
              <Slider
                min={256}
                max={status.max_memory}
                step={256}
                value={memory}
                onChange={setMemory}
                marks={{
                  256: '256 MB',
                  [Math.floor(status.max_memory / 2)]: formatMemory(Math.floor(status.max_memory / 2)),
                  [status.max_memory]: formatMemory(status.max_memory)
                }}
                disabled={saving}
              />
              <div style={{ textAlign: 'center', marginBottom: 16 }}>
                <Text strong style={{ fontSize: 18 }}>{formatMemory(memory)}</Text>
              </div>
              <Button
                type="primary"
                onClick={handleHotplugMemory}
                loading={saving}
                disabled={memory === status.current_memory}
                block
              >
                {t('hotplug.applyMemory')}
              </Button>
            </>
          ) : (
            <Alert
              message={t('hotplug.memoryNotEnabled')}
              type="warning"
              showIcon
            />
          )}
        </div>
      ) : (
        <Alert message={t('hotplug.fetchFailed')} type="error" />
      )}
    </Modal>
  )
}

export default HotplugModal
