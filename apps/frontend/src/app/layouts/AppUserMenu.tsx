import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, LogOut, Settings, UserRound } from 'lucide-react'
import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useNavigate } from 'react-router-dom'
import { logoutOperator, operatorSessionQueryKey } from '../../features/auth/api/operatorSession'
import { useOperatorSession } from '../../features/auth/hooks/useOperatorSession'
import { cn } from '../../shared/ui/cn'

function userInitials(email: string): string {
  const name = email.split('@')[0] ?? email
  if (name.length >= 2) {
    return name.slice(0, 2).toUpperCase()
  }
  return name.slice(0, 1).toUpperCase()
}

type MenuItemProps = {
  icon: ReactNode
  label: string
  onClick: () => void
}

function MenuItem({ icon, label, onClick }: MenuItemProps) {
  return (
    <button
      type="button"
      role="menuitem"
      className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left text-[0.8125rem] text-[#D5DDEA] transition-colors hover:bg-white/[0.06]"
      onClick={onClick}
    >
      <span className="text-[#8A98AB]" aria-hidden>
        {icon}
      </span>
      {label}
    </button>
  )
}

function MenuDivider() {
  return <div className="my-1 border-t border-line/80" role="separator" />
}

export function AppUserMenu() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const session = useOperatorSession()
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  const logoutMutation = useMutation({
    mutationFn: logoutOperator,
    onSuccess: async () => {
      setOpen(false)
      await queryClient.invalidateQueries({ queryKey: operatorSessionQueryKey })
      if (session.data?.system === 'users' || session.data?.mode === 'passphrase') {
        navigate('/login', { replace: true })
      }
    },
  })

  const userEmail = session.data?.user?.email
  const showMenu =
    session.data?.authenticated === true &&
    (session.data.system === 'users' || session.data.mode === 'passphrase')

  useLayoutEffect(() => {
    if (!open || !triggerRef.current || !menuRef.current) {
      return
    }
    const rect = triggerRef.current.getBoundingClientRect()
    const menu = menuRef.current
    const gap = 8
    menu.style.position = 'fixed'
    menu.style.top = `${rect.bottom + gap}px`
    const menuWidth = menu.offsetWidth > 0 ? menu.offsetWidth : 208
    menu.style.left = `${Math.max(8, rect.right - menuWidth)}px`
    menu.style.zIndex = '50'
  }, [open])

  useEffect(() => {
    if (!open) {
      return undefined
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
      }
    }
    const onPointerDown = (event: MouseEvent) => {
      const target = event.target as Node
      if (triggerRef.current?.contains(target) || menuRef.current?.contains(target)) {
        return
      }
      setOpen(false)
    }
    window.addEventListener('keydown', onKeyDown)
    window.addEventListener('mousedown', onPointerDown)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('mousedown', onPointerDown)
    }
  }, [open])

  if (!showMenu) {
    return null
  }

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        className="flex items-center gap-2 rounded-[7px] border border-line bg-white/[0.025] py-1 pl-1 pr-2 text-left transition-colors hover:bg-white/[0.04]"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((value) => !value)}
        title={userEmail}
        aria-label={userEmail ? `Menu do usuário ${userEmail}` : 'Menu do usuário'}
      >
        <span
          className="grid size-8 shrink-0 place-items-center rounded-md border border-cobalt/30 bg-accent-deep text-[0.6875rem] font-semibold text-cobalt"
          aria-hidden
        >
          {userEmail ? userInitials(userEmail) : <UserRound className="size-4" />}
        </span>
        <ChevronDown className={cn('size-3.5 text-[#8592A7] transition-transform', open && 'rotate-180')} />
      </button>
      {open
        ? createPortal(
            <div
              ref={menuRef}
              id={menuId}
              role="menu"
              className="w-52 rounded-lg border border-line bg-[#1A222E] p-1.5 shadow-xl"
            >
              <MenuItem
                icon={<UserRound className="size-4" />}
                label="Meu perfil"
                onClick={() => {
                  setOpen(false)
                  navigate('/profile')
                }}
              />
              <MenuItem
                icon={<Settings className="size-4" />}
                label="Configurações"
                onClick={() => {
                  setOpen(false)
                  navigate('/settings')
                }}
              />
              <MenuDivider />
              <MenuItem
                icon={<LogOut className="size-4" />}
                label={logoutMutation.isPending ? 'Saindo…' : 'Sair'}
                onClick={() => {
                  if (!logoutMutation.isPending) {
                    logoutMutation.mutate()
                  }
                }}
              />
            </div>,
            document.body,
          )
        : null}
    </>
  )
}
