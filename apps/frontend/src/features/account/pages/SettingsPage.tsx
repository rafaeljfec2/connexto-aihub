import { Link } from 'react-router-dom'
import { Panel } from '../../../shared/ui/panel'

export function SettingsPage() {
  return (
    <div className="space-y-4 p-1">
      <div>
        <h1 className="text-lg font-semibold text-primary">Configurações</h1>
        <p className="text-sm text-secondary">Preferências da interface e da conta.</p>
      </div>
      <Panel className="max-w-lg space-y-4 border-line/80 p-6">
        <p className="text-sm text-secondary">
          Alteração de senha e notificações ficam disponíveis quando o envio de e-mail estiver
          configurado no hub.
        </p>
        <Link className="text-sm text-cobalt hover:underline" to="/forgot-password">
          Redefinir senha por e-mail
        </Link>
      </Panel>
    </div>
  )
}
