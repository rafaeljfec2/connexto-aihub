import { useQueryClient } from '@tanstack/react-query'
import { overviewQueryKeys } from '../../features/overview'
import { useOperationalEvents } from '../../shared/realtime/useOperationalEvents'
import { AppShell } from './AppShell'

export function RootLayout() {
  const queryClient = useQueryClient()
  const { connectionState, reconnect } = useOperationalEvents(queryClient, {
    readiness: overviewQueryKeys.readiness,
    providerDiagnostics: overviewQueryKeys.providerDiagnostics,
  })

  return <AppShell connectionState={connectionState} onReconnect={reconnect} />
}
