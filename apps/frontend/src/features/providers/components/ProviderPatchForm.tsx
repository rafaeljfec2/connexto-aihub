import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import type { AdminProvider } from '../types'

const patchSchema = z.object({
  name: z.string().min(1, 'Name is required').max(128),
  endpoint: z.string().url('Endpoint must be a valid URL'),
  credential: z.string().optional(),
})

type PatchFormValues = z.infer<typeof patchSchema>

type ProviderPatchFormProps = {
  provider: AdminProvider
  isPending: boolean
  errorMessage?: string
  onSubmit: (values: { name?: string; endpoint?: string; credential?: string }) => void
}

export function ProviderPatchForm({ provider, isPending, errorMessage, onSubmit }: ProviderPatchFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<PatchFormValues>({
    resolver: zodResolver(patchSchema),
    defaultValues: {
      name: provider.name,
      endpoint: provider.endpoint,
      credential: '',
    },
  })

  return (
    <form
      className="provider-patch-form"
      onSubmit={handleSubmit((values) => {
        const payload: { name?: string; endpoint?: string; credential?: string } = {}
        if (values.name !== provider.name) {
          payload.name = values.name
        }
        if (values.endpoint !== provider.endpoint) {
          payload.endpoint = values.endpoint
        }
        if (values.credential && values.credential.length > 0) {
          payload.credential = values.credential
        }
        onSubmit(payload)
      })}
    >
      <label>
        Name
        <input type="text" autoComplete="off" {...register('name')} />
        {errors.name ? <span className="field-error">{errors.name.message}</span> : null}
      </label>
      <label>
        Endpoint
        <input type="url" autoComplete="off" {...register('endpoint')} />
        {errors.endpoint ? <span className="field-error">{errors.endpoint.message}</span> : null}
      </label>
      <label>
        Credential (write-only)
        <input
          type="password"
          autoComplete="new-password"
          placeholder={provider.credential_configured ? 'Configured — enter to replace' : 'Not configured'}
          {...register('credential')}
        />
        {errors.credential ? <span className="field-error">{errors.credential.message}</span> : null}
      </label>
      {errorMessage ? <p className="field-error" role="alert">{errorMessage}</p> : null}
      <button type="submit" disabled={isPending}>
        {isPending ? 'Saving…' : 'Save changes'}
      </button>
    </form>
  )
}
