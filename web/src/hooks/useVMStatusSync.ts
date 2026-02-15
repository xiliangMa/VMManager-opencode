import { useEffect, useState, useCallback, useRef } from 'react'

export interface VMStatusInfo {
  vmId: string
  status: string
  cpuUsage?: number
  memoryUsage?: number
  diskUsage?: number
  networkIn?: number
  networkOut?: number
  lastUpdated?: string
}

export interface StatusChangeEvent {
  type: 'status_change'
  vmId: string
  oldStatus: string
  newStatus: string
  timestamp: string
  info?: VMStatusInfo
}

export interface VMStatusMessage {
  type: 'status_change' | 'full_sync' | 'error'
  vmId?: string
  oldStatus?: string
  newStatus?: string
  timestamp?: string
  info?: VMStatusInfo
  statuses?: Record<string, VMStatusInfo>
  message?: string
}

export function useVMStatusSync() {
  const [statuses, setStatuses] = useState<Record<string, VMStatusInfo>>({})
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const maxReconnectAttempts = 5

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.hostname
    const port = window.location.port || (protocol === 'wss:' ? '443' : '80')
    const wsUrl = `${protocol}//${host}:${port}/ws/vm-status`

    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setConnected(true)
        setError(null)
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const data: VMStatusMessage = JSON.parse(event.data)
          
          switch (data.type) {
            case 'status_change':
              if (data.vmId && data.newStatus) {
                setStatuses(prev => ({
                  ...prev,
                  [data.vmId!]: {
                    vmId: data.vmId!,
                    status: data.newStatus!,
                    lastUpdated: data.timestamp,
                    ...data.info
                  }
                }))
              }
              break
              
            case 'full_sync':
              if (data.statuses) {
                setStatuses(data.statuses)
              }
              break
              
            case 'error':
              setError(data.message || 'Unknown error')
              break
          }
        } catch (e) {
          console.error('[VMStatusSync] Failed to parse message:', e)
        }
      }

      ws.onerror = () => {
        setError('WebSocket connection error')
      }

      ws.onclose = () => {
        setConnected(false)

        if (reconnectAttemptsRef.current < maxReconnectAttempts) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000)
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttemptsRef.current++
            connect()
          }, delay)
        } else {
          setError('Max reconnection attempts reached')
        }
      }
    } catch (e) {
      setError('Failed to create WebSocket connection')
    }
  }, [])

  useEffect(() => {
    connect()

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
    }
  }, [connect])

  const getVMStatus = useCallback((vmId: string): VMStatusInfo | undefined => {
    return statuses[vmId]
  }, [statuses])

  const reconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0
    if (wsRef.current) {
      wsRef.current.close()
    }
    connect()
  }, [connect])

  return {
    statuses,
    connected,
    error,
    getVMStatus,
    reconnect
  }
}
