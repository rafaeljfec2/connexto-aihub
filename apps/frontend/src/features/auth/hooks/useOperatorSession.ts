import { useQuery } from '@tanstack/react-query'
import { fetchOperatorSession, operatorSessionQueryKey } from '../api/operatorSession'

export function useOperatorSession() {
  return useQuery({
    queryKey: operatorSessionQueryKey,
    queryFn: fetchOperatorSession,
    staleTime: 30_000,
  })
}
