import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { I18nextProvider } from 'react-i18next'
import i18n from '../../shared/i18n'
import { TooltipProvider } from '../../shared/ui'
import { appQueryClient } from './queryClient'

type AppProvidersProps = {
  children: ReactNode
}

export function AppProviders({ children }: AppProvidersProps) {
  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={appQueryClient}>
        <TooltipProvider>{children}</TooltipProvider>
      </QueryClientProvider>
    </I18nextProvider>
  )
}
