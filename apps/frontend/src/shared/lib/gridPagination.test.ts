import { describe, expect, it } from 'vitest'
import {
  buildGridPaginationMeta,
  clampGridPage,
  DEFAULT_GRID_PAGE_SIZE,
  normalizeGridPageSize,
  sliceGridPageItems,
} from './gridPagination'

describe('gridPagination', () => {
  it('clamps page into valid range', () => {
    expect(clampGridPage(0, 5)).toBe(1)
    expect(clampGridPage(99, 5)).toBe(5)
  })

  it('normalizes page size to allowed options', () => {
    expect(normalizeGridPageSize(25)).toBe(25)
    expect(normalizeGridPageSize(99)).toBe(DEFAULT_GRID_PAGE_SIZE)
  })

  it('builds range metadata', () => {
    expect(buildGridPaginationMeta(25, 2, 10)).toEqual({
      page: 2,
      pageSize: 10,
      totalItems: 25,
      totalPages: 3,
      rangeStart: 11,
      rangeEnd: 20,
    })
  })

  it('slices items for the active page', () => {
    const items = [1, 2, 3, 4, 5]
    expect(sliceGridPageItems(items, 2, 2)).toEqual([3, 4])
  })
})
