import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react'
import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import {
  GRID_PAGE_SIZE_OPTIONS,
  type GridPaginationMeta,
} from '../lib/gridPagination'
import { cn } from './cn'

export type GridPaginationProps = {
  meta: GridPaginationMeta
  onPreviousPage: () => void
  onNextPage: () => void
  pageSize: number
  onPageSizeChange: (pageSize: number) => void
  pageSizeOptions?: readonly number[]
  className?: string
}

type MenuPosition = {
  left: number
  top: number
  minWidth: number
  placement: 'above' | 'below'
}

type GridPageSizeSelectProps = {
  value: number
  options: readonly number[]
  onChange: (pageSize: number) => void
}

const triggerClass =
  'inline-flex min-w-[52px] items-center justify-between gap-1.5 rounded-md border border-line bg-[#171E29] px-2 py-1 font-mono text-[0.625rem] text-[#AFBBCD] outline-none transition-colors hover:bg-white/[0.04] focus-visible:border-cobalt focus-visible:shadow-[0_0_0_3px_rgb(120_150_232_/_0.12)]'

const optionClass =
  'block w-full cursor-pointer rounded-[5px] border-0 bg-transparent px-2 py-1.5 text-left font-mono text-[0.625rem] text-[#AFBBCD] outline-none transition-colors hover:bg-white/[0.06] focus-visible:bg-white/[0.06]'

function computeMenuPosition(trigger: HTMLButtonElement): MenuPosition {
  const rect = trigger.getBoundingClientRect()
  const minWidth = Math.max(rect.width, 56)
  const left = rect.right - minWidth
  const estimatedMenuHeight = 132
  const gap = 6
  const spaceAbove = rect.top
  const spaceBelow = window.innerHeight - rect.bottom
  const placement =
    spaceAbove >= estimatedMenuHeight + gap || spaceAbove >= spaceBelow ? 'above' : 'below'

  const top =
    placement === 'above' ? Math.max(gap, rect.top - gap) : Math.min(window.innerHeight - gap, rect.bottom + gap)

  return { left, top, minWidth, placement }
}

function GridPageSizeSelect({ value, options, onChange }: GridPageSizeSelectProps) {
  const [open, setOpen] = useState(false)
  const [menuPosition, setMenuPosition] = useState<MenuPosition | null>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLUListElement>(null)
  const listboxId = useId()

  const syncMenuPosition = () => {
    const trigger = triggerRef.current
    if (!trigger) {
      return
    }
    setMenuPosition(computeMenuPosition(trigger))
  }

  useLayoutEffect(() => {
    if (!open) {
      setMenuPosition(null)
      return undefined
    }

    syncMenuPosition()

    const onViewportChange = () => syncMenuPosition()
    window.addEventListener('resize', onViewportChange)
    window.addEventListener('scroll', onViewportChange, true)
    return () => {
      window.removeEventListener('resize', onViewportChange)
      window.removeEventListener('scroll', onViewportChange, true)
    }
  }, [open])

  useEffect(() => {
    if (!open) {
      return undefined
    }

    const onPointerDown = (event: MouseEvent) => {
      const target = event.target as Node
      if (triggerRef.current?.contains(target) || menuRef.current?.contains(target)) {
        return
      }
      setOpen(false)
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
      }
    }

    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  const menu =
    open && menuPosition
      ? createPortal(
          <ul
            ref={menuRef}
            id={listboxId}
            role="listbox"
            aria-label="Registros por página"
            className="fixed z-[200] m-0 list-none overflow-hidden rounded-md border border-line bg-[#171E29] p-1 shadow-[0_10px_28px_rgb(0_0_0_/_0.5)]"
            style={{
              left: menuPosition.left,
              top: menuPosition.top,
              minWidth: menuPosition.minWidth,
              transform: menuPosition.placement === 'above' ? 'translateY(-100%)' : undefined,
            }}
          >
            {options.map((option) => {
              const isActive = option === value
              return (
                <li key={option} role="presentation">
                  <button
                    type="button"
                    role="option"
                    aria-selected={isActive}
                    className={cn(optionClass, isActive && 'bg-cobalt/15 text-[#EAF0FC]')}
                    onClick={() => {
                      onChange(option)
                      setOpen(false)
                    }}
                  >
                    {option}
                  </button>
                </li>
              )
            })}
          </ul>,
          document.body,
        )
      : null

  return (
    <>
      <div className="mr-1 flex items-center gap-1.5">
        <span className="font-mono text-[0.5625rem] text-[#8290A5]">Por página</span>
        <button
          ref={triggerRef}
          type="button"
          className={triggerClass}
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-controls={listboxId}
          aria-label="Registros por página"
          onClick={() => setOpen((current) => !current)}
        >
          <span>{value}</span>
          <ChevronDown
            className={cn('size-3 shrink-0 opacity-70 transition-transform', open && 'rotate-180')}
            aria-hidden
          />
        </button>
      </div>
      {menu}
    </>
  )
}

const navButtonClass =
  'inline-flex size-7 shrink-0 items-center justify-center rounded-md border border-line bg-[#171E29] text-[#AFBBCD] transition-colors hover:bg-white/[0.04] disabled:cursor-not-allowed disabled:opacity-40'

export function GridPagination({
  meta,
  onPreviousPage,
  onNextPage,
  pageSize,
  onPageSizeChange,
  pageSizeOptions = GRID_PAGE_SIZE_OPTIONS,
  className,
}: GridPaginationProps) {
  if (meta.totalItems === 0) {
    return null
  }

  const canGoPrevious = meta.page > 1
  const canGoNext = meta.page < meta.totalPages

  return (
    <div
      className={cn('flex flex-wrap items-center justify-between gap-3', className)}
      role="navigation"
      aria-label="Paginação"
    >
      <span className="font-mono text-[0.5625rem] text-[#8592A7]">
        {meta.rangeStart}–{meta.rangeEnd} de {meta.totalItems.toLocaleString('pt-BR')}
      </span>
      <div className="flex flex-wrap items-center justify-end gap-1.5">
        <GridPageSizeSelect value={pageSize} options={pageSizeOptions} onChange={onPageSizeChange} />
        <button
          type="button"
          className={navButtonClass}
          disabled={!canGoPrevious}
          onClick={onPreviousPage}
          aria-label="Página anterior"
        >
          <ChevronLeft className="size-3.5" aria-hidden />
        </button>
        <span className="min-w-[52px] text-center font-mono text-[0.5625rem] text-[#A9B4C4]">
          {meta.page} / {meta.totalPages}
        </span>
        <button
          type="button"
          className={navButtonClass}
          disabled={!canGoNext}
          onClick={onNextPage}
          aria-label="Próxima página"
        >
          <ChevronRight className="size-3.5" aria-hidden />
        </button>
      </div>
    </div>
  )
}
