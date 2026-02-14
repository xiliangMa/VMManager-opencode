import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Card, Steps, Button, Space, message, Alert, Progress, Result, Modal, Input } from 'antd'
import { ArrowLeftOutlined, PlayCircleOutlined, CheckCircleOutlined, CloudUploadOutlined, SettingOutlined, KeyOutlined } from '@ant-design/icons'
import { vmsApi } from '../../api/client'

const { TextArea } = Input

interface InstallationStatus {
  is_installed: boolean
  install_status: string
  install_progress: number
  agent_installed: boolean
  boot_order: string
  current_status: string
}

const VMInstallation: React.FC = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [currentStep, setCurrentStep] = useState(0)
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState<InstallationStatus | null>(null)
  const [agentScript, setAgentScript] = useState('')
  const [showAgentModal, setShowAgentModal] = useState(false)

  const fetchStatus = async () => {
    if (!id) return
    try {
      const response = await vmsApi.getInstallationStatus(id)
      const data = response.data || response
      setStatus(data)
      
      if (data.install_status === 'installing') {
        setCurrentStep(1)
      } else if (data.install_status === 'completed' && !data.agent_installed) {
        setCurrentStep(2)
      } else if (data.agent_installed) {
        setCurrentStep(3)
      }
    } catch (error) {
      console.error('Failed to fetch installation status:', error)
    }
  }

  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 3000)
    return () => clearInterval(interval)
  }, [id])

  const handleStartInstallation = async () => {
    if (!id) return
    setLoading(true)
    try {
      await vmsApi.startInstallation(id)
      message.success(t('installation.started'))
      setCurrentStep(1)
      fetchStatus()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  const handleFinishInstallation = async () => {
    if (!id) return
    setLoading(true)
    try {
      await vmsApi.finishInstallation(id)
      message.success(t('installation.completed'))
      fetchStatus()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  const handleInstallAgent = async () => {
    if (!id) return
    setLoading(true)
    try {
      await vmsApi.installAgent(id, { 
        agent_type: 'spice-vdagent',
        script: agentScript 
      })
      message.success(t('installation.agentPrepared'))
      setShowAgentModal(false)
      fetchStatus()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  const steps = [
    {
      title: t('installation.prepare'),
      icon: <SettingOutlined />
    },
    {
      title: t('installation.installing'),
      icon: <CloudUploadOutlined />
    },
    {
      title: t('installation.installAgent'),
      icon: <KeyOutlined />
    },
    {
      title: t('installation.done'),
      icon: <CheckCircleOutlined />
    }
  ]

  return (
    <Card
      title={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/vms/${id}`)} />
          {t('installation.title')}
        </Space>
      }
      style={{ margin: 24 }}
    >
      <Steps current={currentStep} items={steps} style={{ marginBottom: 32 }} />

      {currentStep === 0 && (
        <Result
          status="info"
          title={t('installation.prepareTitle')}
          subTitle={t('installation.prepareDesc')}
          extra={[
            <Button 
              type="primary" 
              icon={<PlayCircleOutlined />} 
              size="large"
              loading={loading}
              onClick={handleStartInstallation}
              key="start"
            >
              {t('installation.startInstallation')}
            </Button>,
            <Button 
              onClick={() => navigate(`/vms/${id}/console`)}
              key="console"
            >
              {t('vm.openConsole')}
            </Button>
          ]}
        />
      )}

      {currentStep === 1 && (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <Progress 
            type="circle" 
            percent={status?.install_progress || 0} 
            status="active"
            format={() => status?.install_status || t('installation.installing')}
          />
          <Alert
            message={t('installation.installingTip')}
            description={t('installation.installingTipDesc')}
            type="info"
            showIcon
            style={{ marginTop: 24, maxWidth: 600, margin: '24px auto' }}
          />
          <Space style={{ marginTop: 24 }}>
            <Button 
              type="primary"
              onClick={() => navigate(`/vms/${id}/console`)}
            >
              {t('installation.openConsole')}
            </Button>
            <Button 
              type="primary" 
              danger
              loading={loading}
              onClick={handleFinishInstallation}
            >
              {t('installation.finishInstallation')}
            </Button>
          </Space>
        </div>
      )}

      {currentStep === 2 && (
        <Result
          status="success"
          title={t('installation.systemInstalled')}
          subTitle={t('installation.systemInstalledDesc')}
          extra={[
            <Button 
              type="primary" 
              icon={<KeyOutlined />} 
              size="large"
              onClick={() => setShowAgentModal(true)}
              key="installAgent"
            >
              {t('installation.installSpiceAgent')}
            </Button>,
            <Button 
              onClick={() => navigate(`/vms/${id}/console`)}
              key="console"
            >
              {t('vm.openConsole')}
            </Button>
          ]}
        />
      )}

      {currentStep === 3 && (
        <Result
          status="success"
          title={t('installation.allCompleted')}
          subTitle={t('installation.allCompletedDesc')}
          extra={[
            <Button 
              type="primary"
              onClick={() => navigate(`/vms/${id}`)}
              key="back"
            >
              {t('installation.backToVm')}
            </Button>,
            <Button 
              onClick={() => navigate(`/vms/${id}/console`)}
              key="console"
            >
              {t('vm.openConsole')}
            </Button>
          ]}
        />
      )}

      <Modal
        title={t('installation.agentScript')}
        open={showAgentModal}
        onOk={handleInstallAgent}
        onCancel={() => setShowAgentModal(false)}
        confirmLoading={loading}
        width={700}
      >
        <Alert
          message={t('installation.agentScriptTip')}
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <TextArea
          rows={12}
          value={agentScript || `#!/bin/bash
# Install SPICE vdagent for better clipboard and resolution support
apt-get update && apt-get install -y spice-vdagent 2>/dev/null || \\
yum install -y spice-vdagent 2>/dev/null || \\
zypper install -y spice-vdagent 2>/dev/null
systemctl enable spice-vdagent 2>/dev/null || true
systemctl start spice-vdagent 2>/dev/null || true
echo "SPICE vdagent installation completed"`}
          onChange={(e) => setAgentScript(e.target.value)}
        />
        <p style={{ marginTop: 8, color: '#666' }}>
          {t('installation.agentScriptNote')}
        </p>
      </Modal>
    </Card>
  )
}

export default VMInstallation
