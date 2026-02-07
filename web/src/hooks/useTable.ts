import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'

interface UseTableOptions<T> {
  api: (params?: any) => Promise<{ data: T[]; meta: any }>
  initialPageSize?: number
}

interface UseTableResult<T> {
  data: T[]
  loading: boolean
  pagination: {
    current: number
    pageSize: number
    total: number
    onChange: (page: number, pageSize: number) => void
  }
  refresh: () => void
  search: string
  setSearch: (value: string) => void
  filters: Record<string, any>
  setFilters: (filters: Record<string, any>) => void
}

export function useTable<T>(options: UseTableOptions<T>): UseTableResult<T> {
  const { t } = useTranslation()
  const { api, initialPageSize = 20 } = options

  const [data, setData] = useState<T[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: initialPageSize,
    total: 0
  })
  const [search, setSearch] = useState('')
  const [filters, setFilters] = useState<Record<string, any>>({})

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: any = {
        page: pagination.current,
        page_size: pagination.pageSize
      }

      if (search) {
        params.search = search
      }

      Object.keys(filters).forEach(key => {
        if (filters[key] !== undefined && filters[key] !== '') {
          params[key] = filters[key]
        }
      })

      const response = await api(params)
      setData(response.data)
      setPagination(prev => ({
        ...prev,
        total: response.meta?.total || 0
      }))
    } catch (error) {
      console.error('Failed to fetch data:', error)
    } finally {
      setLoading(false)
    }
  }, [api, pagination.current, pagination.pageSize, search, filters])

  const handlePaginationChange = (page: number, pageSize: number) => {
    setPagination(prev => ({
      ...prev,
      current: page,
      pageSize
    }))
  }

  return {
    data,
    loading,
    pagination: {
      ...pagination,
      onChange: handlePaginationChange
    },
    refresh: fetchData,
    search,
    setSearch,
    filters,
    setFilters
  }
}
