import { useCallback, useState } from 'react'
import { releaseAPI, type PageParams } from '../api'
import type { PageResult, ReleaseDecision } from '../types/domain'

export function useReleaseStore() {
  const [data, setData] = useState<PageResult<ReleaseDecision>>({ items: [], total: 0, page: 1, pageSize: 10 })
  const [loading, setLoading] = useState(false)
  const load = useCallback(async (params: PageParams = {}) => {
    setLoading(true)
    try { setData(await releaseAPI.list(params)) } finally { setLoading(false) }
  }, [])
  return { data, loading, load }
}

