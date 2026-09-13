import { useCallback, useState } from 'react'
import { lineAPI, type PageParams } from '../api'
import type { PackagingLine, PageResult } from '../types/domain'

const empty: PageResult<PackagingLine> = { items: [], total: 0, page: 1, pageSize: 10 }

export function useLineStore() {
  const [data, setData] = useState(empty)
  const [loading, setLoading] = useState(false)
  const load = useCallback(async (params: PageParams = {}) => {
    setLoading(true)
    try { setData(await lineAPI.list(params)) } finally { setLoading(false) }
  }, [])
  return { data, loading, load }
}

