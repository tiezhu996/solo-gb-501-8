import { useCallback, useState } from 'react'
import { batchAPI, type PageParams } from '../api'
import type { BatchStatus, PageResult, ProductionBatch } from '../types/domain'

export function useBatchStore() {
  const [data, setData] = useState<PageResult<ProductionBatch>>({ items: [], total: 0, page: 1, pageSize: 10 })
  const [loading, setLoading] = useState(false)
  const load = useCallback(async (params: PageParams & { status?: BatchStatus } = {}) => {
    setLoading(true)
    try { setData(await batchAPI.list(params)) } finally { setLoading(false) }
  }, [])
  return { data, loading, load }
}

