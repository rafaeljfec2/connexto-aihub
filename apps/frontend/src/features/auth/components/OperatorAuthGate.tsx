import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { Skeleton } from '../../../shared/ui/skeleton'
import { fetchOperatorSession, operatorSessionQueryKey, requiresOperatorLogin } from '../api/operatorSession'

type OperatorAuthGateProps = {
  children: ReactNode
}

export function OperatorAuthGate({ children }: OperatorAuthGateProps) {
  const location = useLocation()
  const sessionQuery = useQuery({
    queryKey: operatorSessionQueryKey,
    queryFn: fetchOperatorSession,
    retry: 1,
    staleTime: 30_000,
  })

  if (sessionQuery.isPending) {
    return (
      <div className="grid min-h-svh place-items-center bg-graphite px-4">
        <div className="w-full max-w-md space-y-3">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-2/3" />
        </div>
      </div>
    )
  }

  if (sessionQuery.isError) {
    return (
      <div className="grid min-h-svh place-items-center bg-graphite px-4 text-center">
        <p className="text-sm text-failure">Unable to verify operator session. Check that the hub is running.</p>
      </div>
    )
  }

  if (requiresOperatorLogin(sessionQuery.data)) {
    const redirect = encodeURIComponent(`${location.pathname}${location.search}`)
    return <Navigate to={`/login?redirect=${redirect}`} replace />
  }

  return children
}
