import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { UserPlus } from 'lucide-react'
import { useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
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
  operatorSessionQueryKey,
  requiresOperatorLogin,
  signUpUser,
} from '../api/operatorSession'
import { createSignUpSchema } from '../lib/authSchemas'

type FormValues = z.infer<ReturnType<typeof createSignUpSchema>>

export function SignUpPage() {
  const { t, i18n } = useTranslation('auth')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const redirectTarget = searchParams.get('redirect') ?? '/'

  const schema = useMemo(() => createSignUpSchema(), [i18n.language])

  const sessionQuery = useQuery({
    queryKey: operatorSessionQueryKey,
    queryFn: fetchOperatorSession,
  })

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', password: '', confirm: '' },
  })

  const signUpMutation = useMutation({
    mutationFn: (values: FormValues) => signUpUser(values.email, values.password),
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
        <Panel className="motion-enter w-full max-w-md border-line/80 bg-surface/90 p-8 shadow-xl">
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

  if (sessionQuery.isSuccess && !isUsersAuthSystem(sessionQuery.data)) {
    return <Navigate to="/login" replace />
  }

  const emailTaken =
    signUpMutation.error instanceof ApiError && signUpMutation.error.status === 409

  return (
    <div className="grid min-h-svh place-items-center bg-graphite px-4 py-10">
      <Panel className="motion-enter w-full max-w-md border-line/80 bg-surface/90 p-8 shadow-xl">
        <div className="mb-6 flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-lg border border-cobalt/30 bg-accent-deep text-cobalt">
            <UserPlus className="size-5" aria-hidden />
          </span>
          <div>
            <h1 className="text-lg font-semibold text-primary">{t('createAccountTitle')}</h1>
            <p className="text-xs text-secondary">{t('createAccountLead')}</p>
          </div>
        </div>

        {emailTaken ? (
          <Banner variant="error" className="mb-4">
            {t('emailTaken')}
          </Banner>
        ) : null}

        {signUpMutation.error && !emailTaken ? (
          <Banner variant="error" className="mb-4">
            {t('signUpFailed')}
          </Banner>
        ) : null}

        <form
          className="grid gap-4"
          onSubmit={form.handleSubmit((values) => signUpMutation.mutate(values))}
        >
          <FieldLabel>
            {t('email')}
            <Input
              type="email"
              autoComplete="email"
              autoFocus
              disabled={signUpMutation.isPending}
              {...form.register('email')}
            />
            {form.formState.errors.email ? (
              <span className="text-xs text-failure">{form.formState.errors.email.message}</span>
            ) : null}
          </FieldLabel>
          <FieldLabel>
            {t('password')}
            <Input
              type="password"
              autoComplete="new-password"
              disabled={signUpMutation.isPending}
              {...form.register('password')}
            />
            {form.formState.errors.password ? (
              <span className="text-xs text-failure">{form.formState.errors.password.message}</span>
            ) : null}
          </FieldLabel>
          <FieldLabel>
            {t('confirmPassword')}
            <Input
              type="password"
              autoComplete="new-password"
              disabled={signUpMutation.isPending}
              {...form.register('confirm')}
            />
            {form.formState.errors.confirm ? (
              <span className="text-xs text-failure">{form.formState.errors.confirm.message}</span>
            ) : null}
          </FieldLabel>
          <Button type="submit" variant="primary" size="lg" disabled={signUpMutation.isPending}>
            {signUpMutation.isPending ? t('creatingAccount') : t('createAccount')}
          </Button>
        </form>

        <p className="mt-6 text-center text-xs text-secondary">
          {t('alreadyHaveAccount')}{' '}
          <Link className="text-cobalt hover:underline" to="/login">
            {t('signIn')}
          </Link>
        </p>
      </Panel>
    </div>
  )
}
