import React, { useEffect, useRef, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button, Space, Card, Tag, Typography, message, Tooltip } from 'antd'
import { ArrowLeftOutlined, DisconnectOutlined, ReloadOutlined, CompressOutlined, ExpandOutlined } from '@ant-design/icons'
import { vmsApi } from '../../api/client'
import { useAuthStore } from '../../stores/authStore'

const VMConsole: React.FC = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { token } = useAuthStore()
  
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [vmStatus, setVmStatus] = useState<string>('unknown')
  const [fullscreen, setFullscreen] = useState(false)

  const fetchVmStatus = useCallback(async () => {
    if (!id) return
    try {
      const response = await vmsApi.get(id!)
      setVmStatus(response.data?.status || response.status)
    } catch (err) {
      setVmStatus('unknown')
    }
  }, [id])

  useEffect(() => {
    fetchVmStatus()
    const interval = setInterval(fetchVmStatus, 10000)
    return () => clearInterval(interval)
  }, [fetchVmStatus])

  const handleRefresh = () => {
    if (iframeRef.current) {
      const iframe = iframeRef.current
      const currentSrc = iframe.src
      iframe.src = ''
      setTimeout(() => {
        iframe.src = currentSrc
      }, 100)
    }
  }

  const handleFullscreen = () => {
    if (!iframeRef.current) return

    if (!document.fullscreenElement) {
      iframeRef.current.requestFullscreen()
      setFullscreen(true)
    } else {
      document.exitFullscreen()
      setFullscreen(false)
    }
  }

  const handleDisconnect = () => {
    navigate(`/vms/${id}`)
  }

  const handleCtrlAltDel = () => {
    message.info(t('message.useCtrlAltDel'))
  }

  const statusColors: Record<string, string> = {
    running: 'green',
    stopped: 'red',
    creating: 'processing'
  }

  const vncUrl = `/novnc/vnc_lite.html?host=${window.location.hostname}&port=${window.location.port || '8080'}&path=ws/vnc/${id}&password=${token || ''}`

  return (
    <div>
      <Card
        title={
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/vms/${id}`)}>
              {t('common.back')}
            </Button>
            <span>{t('console.vncConsole')} - {id}</span>
            {vmStatus && (
              <Tag color={statusColors[vmStatus] || 'default'}>
                {vmStatus === 'running' ? t('console.connected') : t('console.disconnected')}
              </Tag>
            )}
          </Space>
        }
        extra={
          <Space>
            <Tooltip title={t('console.ctrlAltDel')}>
              <Button onClick={handleCtrlAltDel}>Ctrl+Alt+Del</Button>
            </Tooltip>
            <Tooltip title={t('common.refresh')}>
              <Button icon={<ReloadOutlined />} onClick={handleRefresh} />
            </Tooltip>
            <Tooltip title={t('console.fullscreen')}>
              <Button 
                icon={fullscreen ? <CompressOutlined /> : <ExpandOutlined />} 
                onClick={handleFullscreen} 
              />
            </Tooltip>
            <Tooltip title={t('console.disconnect')}>
              <Button icon={<DisconnectOutlined />} onClick={handleDisconnect} danger />
            </Tooltip>
          </Space>
        }
      >
        <Space direction="vertical" style={{ width: '100%' }} size="small">
          {vmStatus === 'running' && (
            <div
              style={{
                width: '100%',
                height: '600px',
                border: '1px solid #d9d9d9',
                borderRadius: '4px',
                overflow: 'hidden'
              }}
            >
              <iframe
                ref={iframeRef}
                src={vncUrl}
                style={{
                  width: '100%',
                  height: '100%',
                  border: 'none'
                }}
                title={t('console.vncConsole')}
              />
            </div>
          )}

          {vmStatus !== 'running' && (
            <div style={{ 
              textAlign: 'center', 
              padding: '40px',
              background: '#f5f5f5',
              borderRadius: '4px'
            }}>
              <Typography.Title level={4}>
                {vmStatus === 'running' ? t('console.connecting') : t('console.disconnected')}
              </Typography.Title>
              <Typography.Text type="secondary">
                {t('message.vmMustBeRunning')}
              </Typography.Text>
              <br /><br />
              <Space>
                <Button type="primary" onClick={() => navigate(`/vms/${id}`)}>
                  {t('common.back')}
                </Button>
              </Space>
            </div>
          )}
        </Space>
      </Card>
    </div>
  )
}

export default VMConsole
