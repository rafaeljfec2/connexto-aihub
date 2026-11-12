import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { UserPlus } from 'lucide-react'
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
  operatorSessionQueryKey,
  requiresOperatorLogin,
  signUpUser,
} from '../api/operatorSession'

const schema = z
  .object({
    email: z.string().email('Enter a valid email'),
    password: z.string().min(8, 'Use at least 8 characters'),
    confirm: z.string().min(1, 'Confirm your password'),
  })
  .refine((data) => data.password === data.confirm, {
    message: 'Passwords do not match',
    path: ['confirm'],
  })

type FormValues = z.infer<typeof schema>

export function SignUpPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const redirectTarget = searchParams.get('redirect') ?? '/'

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
            <h1 className="text-lg font-semibold text-primary">Create account</h1>
            <p className="text-xs text-secondary">Register with your email and password.</p>
          </div>
        </div>

        {emailTaken ? (
          <Banner variant="error" className="mb-4">
            That email is already registered. Sign in or use another email.
          </Banner>
        ) : null}

        {signUpMutation.error && !emailTaken ? (
          <Banner variant="error" className="mb-4">
            Sign-up failed. Verify your details and try again.
          </Banner>
        ) : null}

        <form
          className="grid gap-4"
          onSubmit={form.handleSubmit((values) => signUpMutation.mutate(values))}
        >
          <FieldLabel>
            Email
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
            Password
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
            Confirm password
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
            {signUpMutation.isPending ? 'Creating account…' : 'Create account'}
          </Button>
        </form>

        <p className="mt-6 text-center text-xs text-secondary">
          Already have an account?{' '}
          <Link className="text-cobalt hover:underline" to="/login">
            Sign in
          </Link>
        </p>
      </Panel>
    </div>
  )
}
