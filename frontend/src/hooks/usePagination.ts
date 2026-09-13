import { useState } from 'react'

export function usePagination(initialPageSize = 10) {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(initialPageSize)
  const update = (nextPage: number, nextPageSize: number) => {
    setPage(nextPageSize !== pageSize ? 1 : nextPage)
    setPageSize(nextPageSize)
  }
  const reset = () => setPage(1)
  return { page, pageSize, update, reset }
}

