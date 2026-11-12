import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { Mail } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { Banner } from '../../../shared/ui/banner'
import { Button } from '../../../shared/ui/button'
import { FieldLabel, Input } from '../../../shared/ui/input'
import { Panel } from '../../../shared/ui/panel'
import { requestPasswordReset } from '../api/operatorSession'

const schema = z.object({
  email: z.string().email('Enter a valid email'),
})

type FormValues = z.infer<typeof schema>

export function ForgotPasswordPage() {
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '' },
  })

  const mutation = useMutation({
    mutationFn: (values: FormValues) => requestPasswordReset(values.email),
  })

  return (
    <div className="grid min-h-svh place-items-center bg-graphite px-4 py-10">
      <Panel className="w-full max-w-md border-line/80 bg-surface/90 p-8 shadow-xl">
        <div className="mb-6 flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-lg border border-cobalt/30 bg-accent-deep text-cobalt">
            <Mail className="size-5" aria-hidden />
          </span>
          <div>
            <h1 className="text-lg font-semibold text-primary">Forgot password</h1>
            <p className="text-xs text-secondary">We will email reset instructions when configured.</p>
          </div>
        </div>

        {mutation.isSuccess ? (
          <Banner variant="info" className="mb-4">
            {mutation.data.message}
          </Banner>
        ) : null}

        <form className="grid gap-4" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
          <FieldLabel>
            Email
            <Input type="email" autoComplete="email" autoFocus disabled={mutation.isPending} {...form.register('email')} />
            {form.formState.errors.email ? (
              <span className="text-xs text-failure">{form.formState.errors.email.message}</span>
            ) : null}
          </FieldLabel>
          <Button type="submit" variant="primary" size="lg" disabled={mutation.isPending}>
            {mutation.isPending ? 'Sending…' : 'Send reset link'}
          </Button>
        </form>

        <p className="mt-6 text-center text-xs text-secondary">
          <Link className="text-cobalt hover:underline" to="/login">
            Back to sign in
          </Link>
        </p>
      </Panel>
    </div>
  )
}
