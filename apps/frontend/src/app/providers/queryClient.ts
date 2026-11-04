import { QueryClient } from '@tanstack/react-query'
import { ApiError } from '../../shared/api/client'
import { operatorSessionQueryKey } from '../../features/auth/api/operatorSession'

export const appQueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchInterval: false,
      refetchOnWindowFocus: false,
    },
    mutations: {
      onError: (error) => {
        if (error instanceof ApiError && error.status === 401) {
          void appQueryClient.invalidateQueries({ queryKey: operatorSessionQueryKey })
        }
      },
    },
  },
})
