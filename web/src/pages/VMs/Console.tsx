import React, { useEffect, useRef, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button, Space, Card, Tag, Typography, message, Tooltip, Modal, Input, Dropdown, Alert, Badge, Select, Popover } from 'antd'
import { 
  ArrowLeftOutlined, 
  DisconnectOutlined, 
  ReloadOutlined, 
  SettingOutlined,
  ThunderboltOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  WifiOutlined,
  DesktopOutlined,
  KeyOutlined
} from '@ant-design/icons'
import { vmsApi } from '../../api/client'
import type { MenuProps } from 'antd'

interface ConsoleInfo {
  type: string
  host: string
  port: number
  password: string
  websocket_url: string
  expires_at: string
}

interface ConsoleSession {
  id: string
  connected: boolean
  connectedAt: Date
  bytesReceived: number
  bytesSent: number
}

const { TextArea } = Input
const { Option } = Select

const VMConsole: React.FC = () => {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [vmStatus, setVmStatus] = useState<string>('unknown')
  const [vmName, setVmName] = useState<string>('')
  const [fullscreen, setFullscreen] = useState(false)
  const [consoleInfo, setConsoleInfo] = useState<ConsoleInfo | null>(null)
  const [clipboardModalVisible, setClipboardModalVisible] = useState(false)
  const [clipboardText, setClipboardText] = useState('')
  const [settingsVisible, setSettingsVisible] = useState(false)
  const [scaleMode, setScaleMode] = useState<'local' | 'remote' | 'none'>('local')
  const [quality, setQuality] = useState<'low' | 'medium' | 'high'>('medium')
  const [session, setSession] = useState<ConsoleSession | null>(null)
  const [reconnecting, setReconnecting] = useState(false)
  const [connectionError, setConnectionError] = useState<string | null>(null)

  const fetchVmStatus = useCallback(async () => {
    if (!id) return
    try {
      const response = await vmsApi.get(id!)
      const vm = response.data || response
      setVmStatus(vm.status)
      setVmName(vm.name)
    } catch (err) {
      setVmStatus('unknown')
    }
  }, [id])

  const fetchConsoleInfo = useCallback(async () => {
    if (!id) return
    try {
      const response = await vmsApi.getConsole(id)
      setConsoleInfo(response.data)
      setConnectionError(null)
    } catch (err: any) {
      setConnectionError(err?.response?.data?.message || t('console.connectionFailed'))
    }
  }, [id, t])

  useEffect(() => {
    fetchVmStatus()
    fetchConsoleInfo()
    const interval = setInterval(fetchVmStatus, 10000)
    return () => clearInterval(interval)
  }, [fetchVmStatus, fetchConsoleInfo])

  useEffect(() => {
    if (vmStatus === 'running' && consoleInfo) {
      setSession({
        id: id || '',
        connected: true,
        connectedAt: new Date(),
        bytesReceived: 0,
        bytesSent: 0
      })
    }
  }, [vmStatus, consoleInfo, id])

  const handleRefresh = () => {
    setReconnecting(true)
    setConnectionError(null)
    if (iframeRef.current) {
      const iframe = iframeRef.current
      const currentSrc = iframe.src
      iframe.src = ''
      setTimeout(() => {
        iframe.src = currentSrc
        setReconnecting(false)
      }, 500)
    }
  }

  const handleFullscreen = () => {
    const container = iframeRef.current?.parentElement
    if (!container) return

    if (!document.fullscreenElement) {
      container.requestFullscreen()
      setFullscreen(true)
    } else {
      document.exitFullscreen()
      setFullscreen(false)
    }
  }

  const handleDisconnect = () => {
    navigate(`/vms/${id}`)
  }

  const handleSendSpecialKey = (key: string) => {
    message.info(t('console.sendKey', { key }))
  }

  const handleClipboardPaste = () => {
    setClipboardModalVisible(true)
  }

  const handleClipboardSubmit = () => {
    if (clipboardText) {
      message.success(t('console.clipboardSent'))
    }
    setClipboardModalVisible(false)
    setClipboardText('')
  }

  const specialKeysMenu: MenuProps['items'] = [
    {
      key: 'ctrl-alt-del',
      label: 'Ctrl+Alt+Del',
      onClick: () => handleSendSpecialKey('Ctrl+Alt+Del')
    },
    {
      key: 'ctrl-alt-backspace',
      label: 'Ctrl+Alt+Backspace',
      onClick: () => handleSendSpecialKey('Ctrl+Alt+Backspace')
    },
    {
      key: 'ctrl-alt-f1',
      label: 'Ctrl+Alt+F1',
      onClick: () => handleSendSpecialKey('Ctrl+Alt+F1')
    },
    {
      key: 'ctrl-alt-f2',
      label: 'Ctrl+Alt+F2',
      onClick: () => handleSendSpecialKey('Ctrl+Alt+F2')
    },
    {
      key: 'ctrl-alt-f7',
      label: 'Ctrl+Alt+F7',
      onClick: () => handleSendSpecialKey('Ctrl+Alt+F7')
    },
    { type: 'divider' },
    {
      key: 'tab',
      label: 'Tab',
      onClick: () => handleSendSpecialKey('Tab')
    },
    {
      key: 'escape',
      label: 'Escape',
      onClick: () => handleSendSpecialKey('Escape')
    },
    {
      key: 'printscreen',
      label: 'Print Screen',
      onClick: () => handleSendSpecialKey('PrintScreen')
    }
  ]

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

  const wsPath = consoleInfo?.websocket_url 
    ? consoleInfo.websocket_url.replace(/^(wss?|ws):\/\/[^\/]+\//, '')
    : `ws/vnc/${id}`
  
  const consoleType = consoleInfo?.type || 'vnc'
  
  const consoleUrl = consoleType === 'spice'
    ? `/spice/spice.html?path=${wsPath}&password=${consoleInfo?.password || ''}`
    : `/novnc/vnc_lite.html?path=${wsPath}&password=${consoleInfo?.password || ''}`

  const sessionInfo = session && (
    <div style={{ padding: 8 }}>
      <div style={{ marginBottom: 8 }}>
        <strong>{t('console.sessionInfo')}</strong>
      </div>
      <div style={{ fontSize: 12 }}>
        <div>{t('console.connectedAt')}: {session.connectedAt.toLocaleTimeString()}</div>
        <div>{t('console.duration')}: {Math.floor((Date.now() - session.connectedAt.getTime()) / 1000)}s</div>
        <div>{t('console.consoleType')}: {consoleType.toUpperCase()}</div>
      </div>
    </div>
  )

  return (
    <div>
      <Card
        title={
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/vms/${id}`)}>
              {t('common.back')}
            </Button>
            <DesktopOutlined />
            <span>{t('console.vncConsole')} - {vmName || id}</span>
            {vmStatus && (
              <Tag color={statusColors[vmStatus] || 'default'}>
                {vmStatus === 'running' ? t('console.connected') : t('console.disconnected')}
              </Tag>
            )}
          </Space>
        }
        extra={
          <Space>
            <Popover content={sessionInfo} title={t('console.sessionInfo')} trigger="hover">
              <Badge status={session?.connected ? 'success' : 'error'} />
              <WifiOutlined style={{ marginLeft: 4 }} />
            </Popover>
            
            <Dropdown menu={{ items: specialKeysMenu }} placement="bottomRight">
              <Button>
                <KeyOutlined /> {t('console.specialKeys')}
              </Button>
            </Dropdown>
            
            <Tooltip title={t('console.paste')}>
              <Button icon={<KeyOutlined />} onClick={handleClipboardPaste} />
            </Tooltip>
            
            <Tooltip title={t('common.settings')}>
              <Button icon={<SettingOutlined />} onClick={() => setSettingsVisible(true)} />
            </Tooltip>
            
            <Tooltip title={t('common.refresh')}>
              <Button 
                icon={<ReloadOutlined />} 
                onClick={handleRefresh}
                loading={reconnecting}
              />
            </Tooltip>
            
            <Tooltip title={fullscreen ? t('console.exitFullscreen') : t('console.fullscreen')}>
              <Button 
                icon={fullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />} 
                onClick={handleFullscreen} 
              />
            </Tooltip>
            
            <Tooltip title={t('console.disconnect')}>
              <Button icon={<DisconnectOutlined />} onClick={handleDisconnect} danger />
            </Tooltip>
          </Space>
        }
      >
        {connectionError && (
          <Alert
            message={t('console.connectionError')}
            description={connectionError}
            type="error"
            showIcon
            closable
            onClose={() => setConnectionError(null)}
            style={{ marginBottom: 16 }}
            action={
              <Button size="small" onClick={handleRefresh}>
                {t('console.reconnect')}
              </Button>
            }
          />
        )}

        {vmStatus === 'running' && consoleInfo && (
          <div
            style={{
              width: '100%',
              height: fullscreen ? 'calc(100vh - 200px)' : '600px',
              border: '1px solid #d9d9d9',
              borderRadius: '4px',
              overflow: 'hidden',
              background: '#000'
            }}
          >
            <iframe
              ref={iframeRef}
              src={consoleUrl}
              style={{
                width: '100%',
                height: '100%',
                border: 'none'
              }}
              title={consoleType === 'spice' ? 'SPICE Console' : t('console.vncConsole')}
              allow="fullscreen"
            />
          </div>
        )}

        {vmStatus !== 'running' && (
          <div style={{ 
            textAlign: 'center', 
            padding: '60px',
            background: '#f5f5f5',
            borderRadius: '4px'
          }}>
            <DesktopOutlined style={{ fontSize: 64, color: '#d9d9d9', marginBottom: 24 }} />
            <Typography.Title level={4}>
              {t('console.vmNotRunning')}
            </Typography.Title>
            <Typography.Text type="secondary">
              {t('message.vmMustBeRunning')}
            </Typography.Text>
            <br /><br />
            <Space>
              <Button type="primary" onClick={async () => {
                if (id) {
                  try {
                    await vmsApi.start(id)
                    message.success(t('vm.startSuccess'))
                    fetchVmStatus()
                  } catch (err) {
                    message.error(t('vm.startFailed'))
                  }
                }
              }}>
                <ThunderboltOutlined /> {t('vm.start')}
              </Button>
              <Button onClick={() => navigate(`/vms/${id}`)}>
                {t('common.back')}
              </Button>
            </Space>
          </div>
        )}
      </Card>

      <Modal
        title={t('console.clipboard')}
        open={clipboardModalVisible}
        onCancel={() => setClipboardModalVisible(false)}
        onOk={handleClipboardSubmit}
        okText={t('console.send')}
      >
        <div style={{ marginBottom: 16 }}>
          <Typography.Text type="secondary">
            {t('console.clipboardHint')}
          </Typography.Text>
        </div>
        <TextArea
          rows={6}
          value={clipboardText}
          onChange={(e) => setClipboardText(e.target.value)}
          placeholder={t('console.clipboardPlaceholder')}
        />
      </Modal>

      <Modal
        title={t('common.settings')}
        open={settingsVisible}
        onCancel={() => setSettingsVisible(false)}
        footer={null}
      >
        <div style={{ padding: '16px 0' }}>
          <div style={{ marginBottom: 24 }}>
            <Typography.Text strong>{t('console.scaleMode')}</Typography.Text>
            <div style={{ marginTop: 8 }}>
              <Select value={scaleMode} onChange={setScaleMode} style={{ width: '100%' }}>
                <Option value="local">{t('console.scaleLocal')}</Option>
                <Option value="remote">{t('console.scaleRemote')}</Option>
                <Option value="none">{t('console.scaleNone')}</Option>
              </Select>
            </div>
          </div>

          <div style={{ marginBottom: 24 }}>
            <Typography.Text strong>{t('console.quality')}</Typography.Text>
            <div style={{ marginTop: 8 }}>
              <Select value={quality} onChange={setQuality} style={{ width: '100%' }}>
                <Option value="low">{t('console.qualityLow')}</Option>
                <Option value="medium">{t('console.qualityMedium')}</Option>
                <Option value="high">{t('console.qualityHigh')}</Option>
              </Select>
            </div>
          </div>

          <div style={{ marginBottom: 16 }}>
            <Typography.Text strong>{t('console.consoleType')}</Typography.Text>
            <div style={{ marginTop: 8 }}>
              <Tag color={consoleType === 'spice' ? 'purple' : 'blue'}>
                {consoleType.toUpperCase()}
              </Tag>
              <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
                {consoleType === 'spice' ? t('console.spiceDesc') : t('console.vncDesc')}
              </Typography.Text>
            </div>
          </div>

          <div>
            <Typography.Text strong>{t('console.connectionInfo')}</Typography.Text>
            <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
              <div>{t('console.host')}: {consoleInfo?.host || 'localhost'}</div>
              <div>{t('console.port')}: {consoleInfo?.port || '-'}</div>
              <div>{t('console.expiresAt')}: {consoleInfo?.expires_at ? new Date(consoleInfo.expires_at).toLocaleString() : '-'}</div>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}

export default VMConsole
