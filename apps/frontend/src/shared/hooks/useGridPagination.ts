import { useEffect, useMemo, useState } from 'react'
import {
  buildGridPaginationMeta,
  clampGridPage,
  DEFAULT_GRID_PAGE_SIZE,
  normalizeGridPageSize,
  sliceGridPageItems,
  type GridPageSize,
} from '../lib/gridPagination'

type UseGridPaginationOptions = {
  initialPageSize?: number
  /** When this key changes, the current page resets to 1. */
  resetKey?: string
}

export function useGridPagination<T>(
  items: readonly T[],
  options?: UseGridPaginationOptions,
) {
  const [pageSize, setPageSizeState] = useState<GridPageSize>(() =>
    normalizeGridPageSize(options?.initialPageSize ?? DEFAULT_GRID_PAGE_SIZE),
  )
  const [page, setPage] = useState(1)
  const resetKey = options?.resetKey ?? ''

  useEffect(() => {
    setPage(1)
  }, [items.length, pageSize, resetKey])

  const meta = useMemo(
    () => buildGridPaginationMeta(items.length, page, pageSize),
    [items.length, page, pageSize],
  )

  useEffect(() => {
    if (page !== meta.page) {
      setPage(meta.page)
    }
  }, [meta.page, page])

  const pageItems = useMemo(
    () => sliceGridPageItems(items, meta.page, pageSize),
    [items, meta.page, pageSize],
  )

  const setPageSize = (nextPageSize: number) => {
    setPageSizeState(normalizeGridPageSize(nextPageSize))
  }

  return {
    meta,
    pageItems,
    pageSize,
    setPage,
    setPageSize,
    goToPreviousPage: () => setPage((current) => clampGridPage(current - 1, meta.totalPages)),
    goToNextPage: () => setPage((current) => clampGridPage(current + 1, meta.totalPages)),
  }
}
