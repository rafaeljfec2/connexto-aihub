import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { LockKeyhole } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { z } from 'zod'
import { ApiError } from '../../../shared/api/client'
import { Banner } from '../../../shared/ui/banner'
import { Button } from '../../../shared/ui/button'
import { FieldLabel, Input } from '../../../shared/ui/input'
import { Panel } from '../../../shared/ui/panel'
import { Skeleton } from '../../../shared/ui/skeleton'
import {
  fetchOperatorSession,
  isUsersAuthSystem,
  loginOperator,
  loginUser,
  operatorSessionQueryKey,
  requiresOperatorLogin,
} from '../api/operatorSession'

const passphraseSchema = z.object({
  passphrase: z.string().min(1, 'Enter the operator passphrase'),
})

const userSchema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(1, 'Enter your password'),
})

type PassphraseFormValues = z.infer<typeof passphraseSchema>
type UserFormValues = z.infer<typeof userSchema>

export function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const redirectTarget = searchParams.get('redirect') ?? '/'

  const sessionQuery = useQuery({
    queryKey: operatorSessionQueryKey,
    queryFn: fetchOperatorSession,
  })

  const usersMode = sessionQuery.isSuccess && isUsersAuthSystem(sessionQuery.data)

  const passphraseForm = useForm<PassphraseFormValues>({
    resolver: zodResolver(passphraseSchema),
    defaultValues: { passphrase: '' },
  })

  const userForm = useForm<UserFormValues>({
    resolver: zodResolver(userSchema),
    defaultValues: { email: '', password: '' },
  })

  const loginMutation = useMutation({
    mutationFn: async (values: PassphraseFormValues | UserFormValues) => {
      if (usersMode && 'email' in values) {
        await loginUser(values.email, values.password)
      } else if ('passphrase' in values) {
        await loginOperator(values.passphrase)
      }
    },
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

  if (sessionQuery.isPending) {
    return (
      <div className="grid min-h-svh place-items-center bg-graphite px-4 py-10">
        <Panel className="w-full max-w-md border-line/80 bg-surface/90 p-8 shadow-xl">
          <div className="mb-6 space-y-3">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-full" />
          </div>
          <Skeleton className="mb-4 h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </Panel>
      </div>
    )
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
            <h1 className="text-lg font-semibold text-primary">
              {usersMode ? 'Sign in' : 'Operator sign-in'}
            </h1>
            <p className="text-xs text-secondary">
              {usersMode ? 'Use your email and password.' : 'Passphrase required for this deployment.'}
            </p>
          </div>
        </div>

        {invalidCredentials ? (
          <Banner variant="error" className="mb-4">
            {usersMode ? 'Invalid email or password. Try again.' : 'Invalid passphrase. Try again.'}
          </Banner>
        ) : null}

        {loginMutation.error && !invalidCredentials ? (
          <Banner variant="error" className="mb-4">
            Sign-in failed. Verify hub configuration and try again.
          </Banner>
        ) : null}

        {usersMode ? (
          <form
            className="grid gap-4"
            onSubmit={userForm.handleSubmit((values) => loginMutation.mutate(values))}
          >
            <FieldLabel>
              Email
              <Input
                type="email"
                autoComplete="email"
                autoFocus
                disabled={loginMutation.isPending}
                {...userForm.register('email')}
              />
              {userForm.formState.errors.email ? (
                <span className="text-xs text-failure">{userForm.formState.errors.email.message}</span>
              ) : null}
            </FieldLabel>
            <FieldLabel>
              Password
              <Input
                type="password"
                autoComplete="current-password"
                disabled={loginMutation.isPending}
                {...userForm.register('password')}
              />
              {userForm.formState.errors.password ? (
                <span className="text-xs text-failure">{userForm.formState.errors.password.message}</span>
              ) : null}
            </FieldLabel>
            <div className="flex items-center justify-between gap-2">
              <Link className="text-xs text-cobalt hover:underline" to="/forgot-password">
                Forgot password?
              </Link>
            </div>
            <Button type="submit" variant="primary" size="lg" disabled={loginMutation.isPending}>
              {loginMutation.isPending ? 'Signing in…' : 'Sign in'}
            </Button>
          </form>
        ) : (
          <form
            className="grid gap-4"
            onSubmit={passphraseForm.handleSubmit((values) => loginMutation.mutate(values))}
          >
            <FieldLabel>
              Passphrase
              <Input
                type="password"
                autoComplete="current-password"
                autoFocus
                disabled={loginMutation.isPending}
                {...passphraseForm.register('passphrase')}
              />
              {passphraseForm.formState.errors.passphrase ? (
                <span className="text-xs text-failure">
                  {passphraseForm.formState.errors.passphrase.message}
                </span>
              ) : null}
            </FieldLabel>
            <Button type="submit" variant="primary" size="lg" disabled={loginMutation.isPending}>
              {loginMutation.isPending ? 'Signing in…' : 'Sign in'}
            </Button>
          </form>
        )}
      </Panel>
    </div>
  )
}
