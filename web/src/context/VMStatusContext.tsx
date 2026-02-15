import { createContext, useContext, ReactNode } from 'react'
import { useVMStatusSync, VMStatusInfo } from '../hooks/useVMStatusSync'

interface VMStatusContextType {
  statuses: Record<string, VMStatusInfo>
  connected: boolean
  error: string | null
  getVMStatus: (vmId: string) => VMStatusInfo | undefined
  reconnect: () => void
}

const VMStatusContext = createContext<VMStatusContextType | null>(null)

export function VMStatusProvider({ children }: { children: ReactNode }) {
  const syncData = useVMStatusSync()

  return (
    <VMStatusContext.Provider value={syncData}>
      {children}
    </VMStatusContext.Provider>
  )
}

export function useVMStatus() {
  const context = useContext(VMStatusContext)
  if (!context) {
    throw new Error('useVMStatus must be used within a VMStatusProvider')
  }
  return context
}

export function useVMStatusById(vmId: string) {
  const { getVMStatus, connected, error } = useVMStatus()
  const status = getVMStatus(vmId)
  return { status, connected, error }
}
