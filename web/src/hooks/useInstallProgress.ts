import { useEffect, useState, useCallback, useRef } from 'react'

export interface InstallProgress {
  vmId: string
  vmName: string
  status: 'pending' | 'installing' | 'completed' | 'failed' | 'paused'
  progress: number
  message: string
  currentStep: string
  totalSteps: number
  completedAt?: string
  startedAt?: string
  errorMessage?: string
}

export function useInstallProgress(vmId: string | null) {
  const [progress, setProgress] = useState<InstallProgress | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const connect = useCallback(() => {
    if (!vmId) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.hostname
    const port = window.location.port || (protocol === 'wss:' ? '443' : '80')
    const wsUrl = `${protocol}//${host}:${port}/ws/install/${vmId}`

    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setConnected(true)
        setError(null)
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.type === 'install_progress' && data.payload) {
            setProgress(data.payload)
          }
        } catch (e) {
          console.error('[InstallProgress] Failed to parse message:', e)
        }
      }

      ws.onerror = (event) => {
        console.error('[InstallProgress] WebSocket error:', event)
        setError('WebSocket connection error')
      }

      ws.onclose = () => {
        setConnected(false)

        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current)
        }

        reconnectTimeoutRef.current = setTimeout(() => {
          if (vmId) {
            connect()
          }
        }, 3000)
      }
    } catch (e) {
      console.error('[InstallProgress] Failed to create WebSocket:', e)
      setError('Failed to create WebSocket connection')
    }
  }, [vmId])

  useEffect(() => {
    if (vmId) {
      connect()
    }

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
    }
  }, [vmId, connect])

  return { progress, connected, error }
}
