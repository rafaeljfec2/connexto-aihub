import { useOperatorSession } from '../../auth/hooks/useOperatorSession'
import { Panel } from '../../../shared/ui/panel'

export function ProfilePage() {
  const session = useOperatorSession()
  const user = session.data?.user

  return (
    <div className="space-y-4 p-1">
      <div>
        <h1 className="text-lg font-semibold text-primary">Meu perfil</h1>
        <p className="text-sm text-secondary">Dados da sua conta no AI Hub.</p>
      </div>
      <Panel className="max-w-lg space-y-3 border-line/80 p-6">
        <div>
          <p className="text-xs text-secondary">E-mail</p>
          <p className="text-sm text-primary">{user?.email ?? '—'}</p>
        </div>
        {user?.role ? (
          <div>
            <p className="text-xs text-secondary">Papel</p>
            <p className="text-sm text-primary">{user.role === 'admin' ? 'Administrador' : user.role}</p>
          </div>
        ) : null}
        <div>
          <p className="text-xs text-secondary">ID</p>
          <p className="break-all font-mono text-xs text-secondary">{user?.id ?? '—'}</p>
        </div>
      </Panel>
    </div>
  )
}
