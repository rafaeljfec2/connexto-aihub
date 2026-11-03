import { useMutation, useQueryClient } from '@tanstack/react-query'
import { overviewQueryKeys } from '../../overview'
import {
  disableProvider,
  enableProvider,
  providersQueryKeys,
  updateProvider,
  validateProvider,
} from '../api/queries'
import type { UpdateProviderInput } from '../types'

export function useProviderMutations(providerId: string) {
  const queryClient = useQueryClient()

  const invalidate = async (): Promise<void> => {
    await queryClient.invalidateQueries({ queryKey: providersQueryKeys.list })
    await queryClient.invalidateQueries({ queryKey: providersQueryKeys.detail(providerId) })
    await queryClient.invalidateQueries({ queryKey: overviewQueryKeys.providerDiagnostics })
  }

  const validate = useMutation({
    mutationFn: () => validateProvider(providerId),
    onSuccess: invalidate,
  })

  const enable = useMutation({
    mutationFn: () => enableProvider(providerId),
    onSuccess: invalidate,
  })

  const disable = useMutation({
    mutationFn: () => disableProvider(providerId),
    onSuccess: invalidate,
  })

  const patch = useMutation({
    mutationFn: (input: UpdateProviderInput) => updateProvider(providerId, input),
    onSuccess: invalidate,
  })

  return { validate, enable, disable, patch }
}
