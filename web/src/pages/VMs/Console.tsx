import React, { useEffect, useRef, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button, Space, Card, Tag, Typography, message, Tooltip, Input } from 'antd'
import { ArrowLeftOutlined, DisconnectOutlined, ReloadOutlined, CompressOutlined, ExpandOutlined } from '@ant-design/icons'
import RFB from '@novnc/novnc'
import { vmsApi } from '../../api/client'
import { useAuthStore } from '../../stores/authStore'

const VMConsole: React.FC = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { token } = useAuthStore()
  
  const containerRef = useRef<HTMLDivElement>(null)
  const rfbref = useRef<any>(null)
  const [vmStatus, setVmStatus] = useState<string>('unknown')
  const [connected, setConnected] = useState(false)
  const [fullscreen, setFullscreen] = useState(false)
  const [clipboardText, setClipboardText] = useState('')

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

  useEffect(() => {
    if (!containerRef.current || vmStatus !== 'running') return

    const wsUrl = `ws://${window.location.host}/ws/vnc/${id}`
    const tokenValue = token || ''

    const rfb = new (RFB as any)(containerRef.current, wsUrl, {
      credentials: { password: tokenValue },
      retry: true,
      reconnectDelay: 500,
      reconnectTimeout: 10000
    })

    rfbref.current = rfb

    rfb.addEventListener('connect', () => {
      setConnected(true)
      message.success(t('console.connected'))
      rfb.focus()
    })

    rfb.addEventListener('disconnect', () => {
      setConnected(false)
      message.warning(t('console.disconnected'))
    })

    rfb.addEventListener('desktopname', (e: any) => {
      console.log('Desktop name:', e.detail.name)
    })

    rfb.addEventListener('securityfailure', (e: any) => {
      console.error('Security failure:', e.detail.reason)
      message.error('VNC connection security failure')
    })

    return () => {
      rfb.disconnect()
      rfbref.current = null
    }
  }, [id, vmStatus, token, t])

  const handleRefresh = () => {
    if (rfbref.current) {
      rfbref.current.disconnect()
      setTimeout(() => {
        if (vmStatus === 'running') {
          setVmStatus('connecting')
          const wsUrl = `ws://${window.location.host}/ws/vnc/${id}`
          const rfb = new (RFB as any)(containerRef.current!, wsUrl, {
            credentials: { password: token || '' },
            retry: true,
            reconnectDelay: 500,
            reconnectTimeout: 10000
          })
          rfbref.current = rfb
        }
      }, 500)
    }
  }

  const handleFullscreen = () => {
    if (!containerRef.current) return

    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen()
      setFullscreen(true)
    } else {
      document.exitFullscreen()
      setFullscreen(false)
    }
  }

  const handleDisconnect = () => {
    if (rfbref.current) {
      rfbref.current.disconnect()
    }
    navigate(`/vms/${id}`)
  }

  const handleCtrlAltDel = () => {
    if (rfbref.current) {
      rfbref.current.sendCtrlAltDel()
    }
  }

  const handleClipboardPaste = () => {
    if (rfbref.current && clipboardText) {
      rfbref.current.clipboardPaste(clipboardText)
      setClipboardText('')
      message.success('Clipboard text sent')
    }
  }

  const statusColors: Record<string, string> = {
    running: 'green',
    stopped: 'red',
    creating: 'processing'
  }

  const isOperational = vmStatus === 'running' && connected

  return (
    <div>
      <Card
        title={
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/vms/${id}`)}>
              {t('common.back')}
            </Button>
            <span>VM Console - {id}</span>
            {vmStatus && (
              <Tag color={statusColors[vmStatus] || 'default'}>
                {vmStatus === 'running' 
                  ? (connected ? t('console.connected') : t('console.connecting'))
                  : t('console.disconnected')}
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
          {isOperational && (
            <Space>
              <Input
                placeholder="Paste text to send to VM..."
                value={clipboardText}
                onChange={(e) => setClipboardText(e.target.value)}
                style={{ width: 300 }}
                onPressEnter={handleClipboardPaste}
              />
              <Button onClick={handleClipboardPaste}>Send to VM</Button>
            </Space>
          )}

          <div
            ref={containerRef}
            style={{
              width: '100%',
              height: '600px',
              backgroundColor: '#000',
              border: '1px solid #d9d9d9',
              borderRadius: '4px',
              overflow: 'hidden'
            }}
          />

          {!isOperational && (
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
                {vmStatus === 'running' 
                  ? 'Click refresh to try connecting again'
                  : 'VM must be running to access the console'}
              </Typography.Text>
              <br /><br />
              <Space>
                <Button type="primary" onClick={() => navigate(`/vms/${id}`)}>
                  {t('common.back')}
                </Button>
                {vmStatus === 'running' && (
                  <Button onClick={handleRefresh}>
                    {t('common.refresh')}
                  </Button>
                )}
              </Space>
            </div>
          )}
        </Space>
      </Card>
    </div>
  )
}

export default VMConsole
