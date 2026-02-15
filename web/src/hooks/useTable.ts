import { useState, useCallback, useEffect, useRef } from 'react'

interface TableParams {
  page?: number
  page_size?: number
  search?: string
  [key: string]: string | number | undefined
}

interface ApiResponse<T> {
  data: T[]
  meta?: {
    total?: number
  }
}

interface UseTableOptions<T> {
  api: (params?: TableParams) => Promise<ApiResponse<T>>
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
  filters: Record<string, string | number | undefined>
  setFilters: (filters: Record<string, string | number | undefined>) => void
}

export function useTable<T>(options: UseTableOptions<T>): UseTableResult<T> {
  const { api, initialPageSize = 20 } = options

  const [data, setData] = useState<T[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: initialPageSize,
    total: 0
  })
  const [search, setSearch] = useState('')
  const [filters, setFilters] = useState<Record<string, string | number | undefined>>({})
  const mountedRef = useRef(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: TableParams = {
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

      if (response.data && Array.isArray(response.data)) {
        setData(response.data)
        setPagination(prev => ({
          ...prev,
          total: response.meta?.total || response.data.length
        }))
      } else {
        setData([])
        setPagination(prev => ({ ...prev, total: 0 }))
      }
    } catch (_error) {
      setData([])
      setPagination(prev => ({ ...prev, total: 0 }))
    } finally {
      setLoading(false)
    }
  }, [api, pagination.current, pagination.pageSize, search, filters])

  useEffect(() => {
    if (!mountedRef.current) {
      mountedRef.current = true
      fetchData()
    }
  }, [fetchData])

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
