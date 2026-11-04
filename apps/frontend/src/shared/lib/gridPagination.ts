/** Default page size for operational data grids across the control plane UI. */
export const DEFAULT_GRID_PAGE_SIZE = 15

/** Selectable page sizes; grid body height stays fixed and scrolls internally. */
export const GRID_PAGE_SIZE_OPTIONS = [15, 25, 35, 50] as const

export type GridPageSize = (typeof GRID_PAGE_SIZE_OPTIONS)[number]

export function normalizeGridPageSize(value: number): GridPageSize {
  if ((GRID_PAGE_SIZE_OPTIONS as readonly number[]).includes(value)) {
    return value as GridPageSize
  }
  return DEFAULT_GRID_PAGE_SIZE
}

export type GridPaginationMeta = {
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
  rangeStart: number
  rangeEnd: number
}

export function clampGridPage(page: number, totalPages: number): number {
  if (totalPages <= 0) {
    return 1
  }
  return Math.min(Math.max(page, 1), totalPages)
}

export function buildGridPaginationMeta(
  totalItems: number,
  page: number,
  pageSize: number = DEFAULT_GRID_PAGE_SIZE,
): GridPaginationMeta {
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize) || 1)
  const safePage = clampGridPage(page, totalPages)
  const rangeStart = totalItems === 0 ? 0 : (safePage - 1) * pageSize + 1
  const rangeEnd = totalItems === 0 ? 0 : Math.min(totalItems, safePage * pageSize)

  return {
    page: safePage,
    pageSize,
    totalItems,
    totalPages,
    rangeStart,
    rangeEnd,
  }
}

export function sliceGridPageItems<T>(
  items: readonly T[],
  page: number,
  pageSize: number = DEFAULT_GRID_PAGE_SIZE,
): T[] {
  const meta = buildGridPaginationMeta(items.length, page, pageSize)
  const start = (meta.page - 1) * pageSize
  return items.slice(start, start + pageSize)
}
