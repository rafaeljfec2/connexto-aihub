import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { KeyRound } from 'lucide-react'
import { useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { z } from 'zod'
import { ApiError } from '../../../shared/api/client'
import { Banner } from '../../../shared/ui/banner'
import { Button } from '../../../shared/ui/button'
import { FieldLabel, Input } from '../../../shared/ui/input'
import { Panel } from '../../../shared/ui/panel'
import { resetPassword } from '../api/operatorSession'
import { createResetPasswordSchema } from '../lib/authSchemas'

type FormValues = z.infer<ReturnType<typeof createResetPasswordSchema>>

export function ResetPasswordPage() {
  const { t, i18n } = useTranslation('auth')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''

  const schema = useMemo(() => createResetPasswordSchema(), [i18n.language])

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { password: '', confirm: '' },
  })

  const mutation = useMutation({
    mutationFn: (values: FormValues) => resetPassword(token, values.password),
    onSuccess: () => navigate('/login', { replace: true }),
  })

  const invalidToken = mutation.error instanceof ApiError && mutation.error.status === 401

  return (
    <div className="grid min-h-svh place-items-center bg-graphite px-4 py-10">
      <Panel className="motion-enter w-full max-w-md border-line/80 bg-surface/90 p-8 shadow-xl">
        <div className="mb-6 flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-lg border border-cobalt/30 bg-accent-deep text-cobalt">
            <KeyRound className="size-5" aria-hidden />
          </span>
          <div>
            <h1 className="text-lg font-semibold text-primary">{t('resetPasswordTitle')}</h1>
            <p className="text-xs text-secondary">{t('resetPasswordLead')}</p>
          </div>
        </div>

        {!token ? (
          <Banner variant="error" className="mb-4">
            {t('missingResetToken')}
          </Banner>
        ) : null}

        {invalidToken ? (
          <Banner variant="error" className="mb-4">
            {t('invalidResetToken')}
          </Banner>
        ) : null}

        <form className="grid gap-4" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
          <FieldLabel>
            {t('newPassword')}
            <Input type="password" autoComplete="new-password" autoFocus disabled={mutation.isPending || !token} {...form.register('password')} />
            {form.formState.errors.password ? (
              <span className="text-xs text-failure">{form.formState.errors.password.message}</span>
            ) : null}
          </FieldLabel>
          <FieldLabel>
            {t('confirmPassword')}
            <Input type="password" autoComplete="new-password" disabled={mutation.isPending || !token} {...form.register('confirm')} />
            {form.formState.errors.confirm ? (
              <span className="text-xs text-failure">{form.formState.errors.confirm.message}</span>
            ) : null}
          </FieldLabel>
          <Button type="submit" variant="primary" size="lg" disabled={mutation.isPending || !token}>
            {mutation.isPending ? t('updating') : t('updatePassword')}
          </Button>
        </form>

        <p className="mt-6 text-center text-xs text-secondary">
          <Link className="text-cobalt hover:underline" to="/login">
            {t('backToSignIn')}
          </Link>
        </p>
      </Panel>
    </div>
  )
}
