import React from 'react'
import { Progress, Card, Tag, Typography, Space, Alert } from 'antd'
import { CheckCircleOutlined, SyncOutlined, CloseCircleOutlined, PauseCircleOutlined, ClockCircleOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useInstallProgress } from '../../hooks/useInstallProgress'

const { Text } = Typography

interface InstallProgressProps {
  vmId: string
  onComplete?: () => void
}

const statusConfig: Record<string, { color: string; icon: React.ReactNode }> = {
  pending: { color: 'default', icon: <ClockCircleOutlined /> },
  installing: { color: 'processing', icon: <SyncOutlined spin /> },
  completed: { color: 'success', icon: <CheckCircleOutlined /> },
  failed: { color: 'error', icon: <CloseCircleOutlined /> },
  paused: { color: 'warning', icon: <PauseCircleOutlined /> },
}

const InstallProgressCard: React.FC<InstallProgressProps> = ({ vmId, onComplete }) => {
  const { t } = useTranslation()
  const { progress, connected, error } = useInstallProgress(vmId)

  React.useEffect(() => {
    if (progress?.status === 'completed' && onComplete) {
      onComplete()
    }
  }, [progress?.status, onComplete])

  if (!progress) {
    return null
  }

  const config = statusConfig[progress.status] || statusConfig.pending

  return (
    <Card
      title={
        <Space>
          <span>{t('installProgress.title')}</span>
          <Tag color={config.color} icon={config.icon}>
            {t(`installStatus.${progress.status}`)}
          </Tag>
        </Space>
      }
      size="small"
      style={{ marginBottom: 16 }}
    >
      {error && (
        <Alert
          message={t('installProgress.connectionError')}
          description={error}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      <Space direction="vertical" style={{ width: '100%' }}>
        <div>
          <Text type="secondary">{t('installProgress.status')}: </Text>
          <Text strong>{progress.message}</Text>
        </div>

        <Progress
          percent={progress.progress}
          status={
            progress.status === 'completed' ? 'success' :
            progress.status === 'failed' ? 'exception' :
            progress.status === 'installing' ? 'active' : 'normal'
          }
          strokeColor={{
            '0%': '#108ee9',
            '100%': '#87d068',
          }}
        />

        {progress.status === 'installing' && (
          <div>
            <Text type="secondary">
              {t('installProgress.step')}: {progress.currentStep} ({progress.totalSteps})
            </Text>
          </div>
        )}

        {progress.status === 'failed' && progress.errorMessage && (
          <Alert
            message={t('installProgress.failed')}
            description={progress.errorMessage}
            type="error"
            showIcon
          />
        )}

        {progress.status === 'completed' && progress.completedAt && (
          <Alert
            message={t('installProgress.completed')}
            description={t('installProgress.completedDesc')}
            type="success"
            showIcon
          />
        )}

        <Text type="secondary" style={{ fontSize: 12 }}>
          {connected ? t('installProgress.connected') : t('installProgress.disconnected')}
        </Text>
      </Space>
    </Card>
  )
}

export default InstallProgressCard
