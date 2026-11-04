import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { LockKeyhole } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { z } from 'zod'
import { ApiError } from '../../../shared/api/client'
import { Banner } from '../../../shared/ui/banner'
import { Button } from '../../../shared/ui/button'
import { FieldLabel, Input } from '../../../shared/ui/input'
import { Panel } from '../../../shared/ui/panel'
import {
  fetchOperatorSession,
  loginOperator,
  operatorSessionQueryKey,
  requiresOperatorLogin,
} from '../api/operatorSession'

const loginSchema = z.object({
  passphrase: z.string().min(1, 'Enter the operator passphrase'),
})

type LoginFormValues = z.infer<typeof loginSchema>

export function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const redirectTarget = searchParams.get('redirect') ?? '/'

  const sessionQuery = useQuery({
    queryKey: operatorSessionQueryKey,
    queryFn: fetchOperatorSession,
  })

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { passphrase: '' },
  })

  const loginMutation = useMutation({
    mutationFn: (values: LoginFormValues) => loginOperator(values.passphrase),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: operatorSessionQueryKey })
      navigate(redirectTarget.startsWith('/') ? redirectTarget : '/', { replace: true })
    },
  })

  if (sessionQuery.data?.authenticated) {
    return <Navigate to={redirectTarget.startsWith('/') ? redirectTarget : '/'} replace />
  }

  if (sessionQuery.isSuccess && !requiresOperatorLogin(sessionQuery.data)) {
    return <Navigate to="/" replace />
  }

  const invalidCredentials =
    loginMutation.error instanceof ApiError && loginMutation.error.status === 401

  return (
    <div className="grid min-h-svh place-items-center bg-graphite px-4 py-10">
      <Panel className="w-full max-w-md border-line/80 bg-surface/90 p-8 shadow-xl">
        <div className="mb-6 flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-lg border border-cobalt/30 bg-accent-deep text-cobalt">
            <LockKeyhole className="size-5" aria-hidden />
          </span>
          <div>
            <h1 className="text-lg font-semibold text-primary">Operator sign-in</h1>
            <p className="text-xs text-secondary">Passphrase required for this deployment.</p>
          </div>
        </div>

        {invalidCredentials ? (
          <Banner variant="error" className="mb-4">
            Invalid passphrase. Try again.
          </Banner>
        ) : null}

        {loginMutation.error && !invalidCredentials ? (
          <Banner variant="error" className="mb-4">
            Sign-in failed. Verify hub configuration and try again.
          </Banner>
        ) : null}

        <form
          className="grid gap-4"
          onSubmit={form.handleSubmit((values) => loginMutation.mutate(values))}
        >
          <FieldLabel>
            Passphrase
            <Input
              type="password"
              autoComplete="current-password"
              autoFocus
              disabled={loginMutation.isPending}
              {...form.register('passphrase')}
            />
            {form.formState.errors.passphrase ? (
              <span className="text-xs text-failure">{form.formState.errors.passphrase.message}</span>
            ) : null}
          </FieldLabel>
          <Button type="submit" variant="primary" size="lg" disabled={loginMutation.isPending}>
            {loginMutation.isPending ? 'Signing in…' : 'Sign in'}
          </Button>
        </form>
      </Panel>
    </div>
  )
}
