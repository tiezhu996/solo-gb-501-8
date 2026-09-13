import { useCallback, useState } from 'react'
import { inspectionAPI, type PageParams } from '../api'
import type { InspectionSample, PageResult } from '../types/domain'

export function useInspectionStore() {
  const [data, setData] = useState<PageResult<InspectionSample>>({ items: [], total: 0, page: 1, pageSize: 10 })
  const [loading, setLoading] = useState(false)
  const load = useCallback(async (params: PageParams & { result?: string } = {}) => {
    setLoading(true)
    try { setData(await inspectionAPI.list(params)) } finally { setLoading(false) }
  }, [])
  return { data, loading, load }
}

